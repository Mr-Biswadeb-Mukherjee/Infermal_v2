package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	config "github.com/official-biswadeb941/Infermal_v2/Modules/Config"
	domain_generator "github.com/official-biswadeb941/Infermal_v2/Modules/Domain_Generator"
	dnsengine "github.com/official-biswadeb941/Infermal_v2/Modules/DNS"
	wpkg "github.com/official-biswadeb941/Infermal_v2/Modules/Worker"
	progressBar "github.com/official-biswadeb941/Infermal_v2/Modules/Progressbar"

	cooldown "github.com/official-biswadeb941/Infermal_v2/Modules/Cooldown"
)

// Startup animation
func startAnimation(stopChan chan struct{}) {
	frames := []string{"|", "/", "-", "\\"}
	i := 0

	for {
		select {
		case <-stopChan:
			fmt.Print("\rStarting Infermal_v2 Engine ✓           \n")
			return
		default:
			fmt.Printf("\rStarting Infermal_v2 Engine %s", frames[i%len(frames)])
			i++
			time.Sleep(120 * time.Millisecond)
		}
	}
}

func main() {

	// ----------------------------------------------------
	// Start-up animation
	// ----------------------------------------------------
	animStop := make(chan struct{})
	go startAnimation(animStop)

	// ----------------------------------------------------
	// Load config
	// ----------------------------------------------------
	cfg, err := config.LoadOrCreateConfig("Setting/setting.conf")
	if err != nil {
		close(animStop)
		fmt.Println("\nError loading config:", err)
		os.Exit(1)
	}

	// ----------------------------------------------------
	// Build DNS engine
	// ----------------------------------------------------
	dns := dnsengine.New(dnsengine.Config{
		Upstream:  cfg.UpstreamDNS,
		Backup:    cfg.BackupDNS,
		Retries:   cfg.DNSRetries,
		TimeoutMS: cfg.DNSTimeoutMS,
	})

	// ----------------------------------------------------
	// Load keywords
	// ----------------------------------------------------
	keywords, err := domain_generator.LoadKeywordsCSV("Input/Keywords.csv")
	if err != nil {
		close(animStop)
		fmt.Fprintf(os.Stderr, "\nError loading Keywords.csv: %v\n", err)
		os.Exit(1)
	}

	// ----------------------------------------------------
	// Generate domains
	// ----------------------------------------------------
	var allGenerated []string
	for _, base := range keywords {
		groups := domain_generator.RunAll(base)
		for _, g := range groups {
			allGenerated = append(allGenerated, g...)
		}
	}

	total := int64(len(allGenerated))
	if total == 0 {
		close(animStop)
		fmt.Println("No domains generated. Exiting.")
		return
	}

	// ----------------------------------------------------
	// Open output CSV
	// ----------------------------------------------------
	f, err := os.OpenFile("Input/Malicious_Domains.csv", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		close(animStop)
		fmt.Fprintf(os.Stderr, "Error opening output file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	writer := csv.NewWriter(f)
	var writerMu sync.Mutex

	// ----------------------------------------------------
	// Worker pool initialization
	// ----------------------------------------------------
	opts := &wpkg.RunOptions{
		Timeout:         time.Duration(cfg.TimeoutSeconds) * time.Second,
		MaxRetries:      cfg.MaxRetries,
		AutoScale:       cfg.AutoScale,
		MinWorkers:      1,
		NonBlockingLogs: true,
	}
	startWorkers := runtime.NumCPU() * 4
	wp := wpkg.NewWorkerPool(opts, startWorkers)

	// Stop animation
	close(animStop)
	time.Sleep(150 * time.Millisecond)

	// ----------------------------------------------------
	// Runtime bookkeeping
	// ----------------------------------------------------
	var completed int64 = 0
	var resolved int64 = 0 // NEW COUNTER
	start := time.Now()
	done := make(chan struct{})

	// rate limiter
	rateLimiter := time.Tick(time.Second / time.Duration(cfg.RateLimit))

	// ----------------------------------------------------
	// Cooldown Manager
	// ----------------------------------------------------
	cdm := cooldown.NewManager()
	cdm.StartWatcher()

	// ----------------------------------------------------
	// Progress bar
	// ----------------------------------------------------
	pb := progressBar.NewProgressBar(int(total), "Resolving domains", "green")
	pb.StartAutoRender(func() (int64, int64, bool, int64) {
		cur := atomic.LoadInt64(&completed)
		return cur, total, cdm.Active(), cdm.Remaining()
	})

	// ----------------------------------------------------
	// Dispatch DNS tasks
	// ----------------------------------------------------
	var wg sync.WaitGroup

	for _, domain := range allGenerated {
		d := domain

		taskFunc := func(ctx context.Context) (interface{}, []string, []string, []error) {

			if cdm.Active() {
				<-cdm.Gate()
			}

			<-rateLimiter

			ok, _ := dns.Resolve(ctx, d)
			if ok {
				return d, nil, nil, nil
			}
			return nil, nil, nil, nil
		}

		_, resCh, err := wp.SubmitTask(taskFunc, wpkg.Medium, 0)
		if err != nil {
			atomic.AddInt64(&completed, 1)
			pb.Add(1)
			continue
		}

		wg.Add(1)
		go func(rc <-chan wpkg.WorkerResult) {
			defer wg.Done()

			res, ok := <-rc
			if !ok {
				atomic.AddInt64(&completed, 1)
				pb.Add(1)
				return
			}

			// If domain resolved successfully
			if s, ok := res.Result.(string); ok && s != "" {

				writerMu.Lock()
				_ = writer.Write([]string{s})
				if atomic.LoadInt64(&completed)%100 == 0 {
					writer.Flush()
				}
				writerMu.Unlock()

				// NEW: Count resolved domains
				atomic.AddInt64(&resolved, 1)

				// NEW: Live terminal display
				fmt.Printf("\rResolved Domains: %d / %d", atomic.LoadInt64(&resolved), total)
			}

			newCount := atomic.AddInt64(&completed, 1)
			pb.Add(1)

			if cfg.CooldownAfter > 0 && newCount%int64(cfg.CooldownAfter) == 0 {
				cdm.Trigger(int64(cfg.CooldownDuration))
			}

		}(resCh)
	}

	// Wait for workers
	wg.Wait()

	// Shutdown
	close(done)
	wp.Stop()

	writer.Flush()
	pb.StopAutoRender()
	pb.Finish()

	elapsed := time.Since(start).Truncate(time.Millisecond)

	fmt.Printf("\n✔ Resolution complete. Time: %s | Total checked: %d\n", elapsed, total)
	fmt.Printf("✔ Total Resolved Domains: %d\n", resolved)
	fmt.Println("✔ Valid domains appended to Input/Malicious_Domains.csv")
}
