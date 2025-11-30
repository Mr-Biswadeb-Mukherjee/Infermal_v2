package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	config "github.com/official-biswadeb941/Infermal_v2/Modules/Config"
	cooldown "github.com/official-biswadeb941/Infermal_v2/Modules/Cooldown"
	dnsengine "github.com/official-biswadeb941/Infermal_v2/Modules/DNS"
	domain_generator "github.com/official-biswadeb941/Infermal_v2/Modules/Domain_Generator"
	filewriter "github.com/official-biswadeb941/Infermal_v2/Modules/Filewriter"
	progressBar "github.com/official-biswadeb941/Infermal_v2/Modules/Progressbar"
	ratelimiter "github.com/official-biswadeb941/Infermal_v2/Modules/Ratelimiter"
	redis "github.com/official-biswadeb941/Infermal_v2/Modules/Redis"
	ui "github.com/official-biswadeb941/Infermal_v2/Modules/UI"
	wpkg "github.com/official-biswadeb941/Infermal_v2/Modules/Worker"
)

// ------------------------------
//
//	Redis Interface (Only Here)
//
// ------------------------------
type RedisStore interface {
	GetValue(ctx context.Context, key string) (string, error)
	SetValue(ctx context.Context, key string, v interface{}, ttl time.Duration) error
	Eval(ctx context.Context, script string, keys []string, args ...interface{}) (interface{}, error)
}

func main() {

	// -------------------------
	// UI: Starting Animation
	// -------------------------
	animStop := make(chan struct{})
	go ui.Spinner(animStop, "Starting Infermal_v2 Engine")

	// Start banner (timestamp)
	startTime := ui.StartBanner()

	// -------------------------
	// Load Config
	// -------------------------
	cfg, err := config.LoadOrCreateConfig("Setting/setting.conf")
	if err != nil {
		close(animStop)
		fmt.Println("\nError loading config:", err)
		os.Exit(1)
	}

	// -------------------------
	// Init Redis
	// -------------------------
	if err := redis.Init(); err != nil {
		close(animStop)
		fmt.Println("\nError initializing Redis:", err)
		os.Exit(1)
	}

	var rdb RedisStore = redis.Client()

	// -------------------------
	// Rate Limiter Setup
	// -------------------------
	limit := cfg.RateLimit
	if limit <= 0 {
		limit = 999999999
	}

	ratelimiter.Init(rdb, time.Second, int64(limit))

	// -------------------------
	// DNS Engine Setup
	// -------------------------
	dns := dnsengine.New(dnsengine.Config{
		Upstream:  cfg.UpstreamDNS,
		Backup:    cfg.BackupDNS,
		Retries:   cfg.DNSRetries,
		TimeoutMS: cfg.DNSTimeoutMS,
	})

	// -------------------------
	// Domain Generation (NEW)
	// -------------------------
	allGenerated, err := domain_generator.GenerateFromCSV("Input/Keywords.csv")
	if err != nil {
		close(animStop)
		fmt.Fprintf(os.Stderr, "\nError processing Keywords.csv: %v\n", err)
		os.Exit(1)
	}

	total := int64(len(allGenerated))
	if total == 0 {
		close(animStop)
		fmt.Println("No domains generated. Exiting.")
		return
	}

	// -------------------------
	// CSV Writer
	// -------------------------
	fw, err := filewriter.SafeNewCSVWriter("Input/Domains.csv", filewriter.Overwrite)
	if err != nil {
		close(animStop)
		fmt.Println("Error opening CSV writer:", err)
		os.Exit(1)
	}

	// -------------------------
	// Worker Pool Setup
	// -------------------------
	opts := &wpkg.RunOptions{
		Timeout:         time.Duration(cfg.TimeoutSeconds) * time.Second,
		MaxRetries:      cfg.MaxRetries,
		AutoScale:       cfg.AutoScale,
		MinWorkers:      1,
		NonBlockingLogs: true,
	}

	wp := wpkg.NewWorkerPool(opts, runtime.NumCPU()*4, rdb)

	// Stop animation here (UI polish)
	close(animStop)
	time.Sleep(150 * time.Millisecond)

	// -------------------------
	// Resolve Domains
	// -------------------------
	var completed int64
	var resolved int64

	cdm := cooldown.NewManager()
	cdm.StartWatcher()

	pb := progressBar.NewProgressBar(int(total), "Resolving domains", "green")
	pb.StartAutoRender(func() (int64, int64, bool, int64) {
		cur := atomic.LoadInt64(&completed)
		return cur, total, cdm.Active(), cdm.Remaining()
	})

	var wg sync.WaitGroup

	successTTL := 48 * time.Hour
	failTTL := 10 * time.Second

	for _, domain := range allGenerated {
		d := domain

		taskFunc := func(ctx context.Context) (interface{}, []string, []string, []error) {

			// Redis cache lookup
			if rdb != nil {
				cacheCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
				cached, err := rdb.GetValue(cacheCtx, "dns:"+d)
				cancel()

				if err == nil {
					if cached == "1" {
						return d, nil, nil, nil
					}
					if cached == "0" {
						return nil, nil, nil, nil
					}
				}
			}

			// Cooldown
			if cdm.Active() {
				select {
				case <-ctx.Done():
					return nil, nil, nil, []error{ctx.Err()}
				case <-cdm.Gate():
				}
			}

			// Rate limiter
			for {
				select {
				case <-ctx.Done():
					return nil, nil, nil, []error{ctx.Err()}
				default:
				}

				allowed, err := ratelimiter.RateLimit(ctx, "dns-rate")
				if err != nil {
					break
				}
				if allowed {
					break
				}

				time.Sleep(10 * time.Millisecond)
			}

			// DNS resolve
			ok, _ := dns.Resolve(ctx, d)

			if ok {
				if rdb != nil {
					go func(domain string) {
						cctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
						_ = rdb.SetValue(cctx, "dns:"+domain, "1", successTTL)
						cancel()
					}(d)
				}
				return d, nil, nil, nil
			}

			if rdb != nil {
				go func(domain string) {
					cctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
					_ = rdb.SetValue(cctx, "dns:"+domain, "0", failTTL)
					cancel()
				}(d)
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

			if s, ok := res.Result.(string); ok && s != "" {
				fw.WriteRow([]string{s})
				atomic.AddInt64(&resolved, 1)
			}

			newCount := atomic.AddInt64(&completed, 1)
			pb.Add(1)

			if cfg.CooldownAfter > 0 && newCount%int64(cfg.CooldownAfter) == 0 {
				cdm.Trigger(int64(cfg.CooldownDuration))
			}

		}(resCh)
	}

	wg.Wait()
	wp.Stop()
	_ = fw.Close()

	pb.StopAutoRender()
	pb.Finish()

	// -------------------------
	// UI: End Banner / Summary
	// -------------------------
	ui.EndBanner(startTime, total, resolved)

	fmt.Println("✔ Valid domains written to Input/Domains.csv")
	redis.Close()
}
