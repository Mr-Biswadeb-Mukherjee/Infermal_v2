// Updated app.go using future-proof SafeNewCSVWriter
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
    domain_generator "github.com/official-biswadeb941/Infermal_v2/Modules/Domain_Generator"
    dnsengine "github.com/official-biswadeb941/Infermal_v2/Modules/DNS"
    progressBar "github.com/official-biswadeb941/Infermal_v2/Modules/Progressbar"
    wpkg "github.com/official-biswadeb941/Infermal_v2/Modules/Worker"
    cooldown "github.com/official-biswadeb941/Infermal_v2/Modules/Cooldown"
    filewriter "github.com/official-biswadeb941/Infermal_v2/Modules/Filewriter"
)

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
    animStop := make(chan struct{})
    go startAnimation(animStop)

    // Load config
    cfg, err := config.LoadOrCreateConfig("Setting/setting.conf")
    if err != nil {
        close(animStop)
        fmt.Println("\nError loading config:", err)
        os.Exit(1)
    }

    // DNS engine
    dns := dnsengine.New(dnsengine.Config{
        Upstream:  cfg.UpstreamDNS,
        Backup:    cfg.BackupDNS,
        Retries:   cfg.DNSRetries,
        TimeoutMS: cfg.DNSTimeoutMS,
    })

    // Load keywords
    keywords, err := domain_generator.LoadKeywordsCSV("Input/Keywords.csv")
    if err != nil {
        close(animStop)
        fmt.Fprintf(os.Stderr, "\nError loading Keywords.csv: %v\n", err)
        os.Exit(1)
    }

    // Generate domains
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

    // Create async CSV writer (future-proof API)
    fw, err := filewriter.SafeNewCSVWriter("Input/Malicious_Domains.csv", filewriter.Overwrite)
    if err != nil {
        close(animStop)
        fmt.Println("Error opening CSV writer:", err)
        os.Exit(1)
    }

    // Worker pool setup
    opts := &wpkg.RunOptions{
        Timeout:         time.Duration(cfg.TimeoutSeconds) * time.Second,
        MaxRetries:      cfg.MaxRetries,
        AutoScale:       cfg.AutoScale,
        MinWorkers:      1,
        NonBlockingLogs: true,
    }

    wp := wpkg.NewWorkerPool(opts, runtime.NumCPU()*4)

    close(animStop)
    time.Sleep(150 * time.Millisecond) // Let animation settle

    var completed int64 = 0
    var resolved int64 = 0
    start := time.Now()

    rateLimiter := time.Tick(time.Second / time.Duration(cfg.RateLimit))

    // Cooldown manager
    cdm := cooldown.NewManager()
    cdm.StartWatcher()

    // Progress bar setup
    pb := progressBar.NewProgressBar(int(total), "Resolving domains", "green")
    pb.StartAutoRender(func() (int64, int64, bool, int64) {
        cur := atomic.LoadInt64(&completed)
        return cur, total, cdm.Active(), cdm.Remaining()
    })

    var wg sync.WaitGroup

    // Submit jobs
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

            if s, ok := res.Result.(string); ok && s != "" {
                fw.WriteRow([]string{s})     // async batched CSV write
                atomic.AddInt64(&resolved, 1)
            }

            newCount := atomic.AddInt64(&completed, 1)
            pb.Add(1)

            if cfg.CooldownAfter > 0 && newCount%int64(cfg.CooldownAfter) == 0 {
                cdm.Trigger(int64(cfg.CooldownDuration))
            }

        }(resCh)
    }

    // Wait for all workers
    wg.Wait()
    wp.Stop()

    // Finalize CSV writer (flush + close + atomic rename)
    fw.Close()

    // Stop progress bar
    pb.StopAutoRender()
    pb.Finish()

    // Summary
    elapsed := time.Since(start).Truncate(time.Millisecond)

    fmt.Printf("\n✔ Resolution complete. Time: %s | Total checked: %d\n", elapsed, total)
    fmt.Printf("✔ Total Resolved Domains: %d\n", resolved)
    fmt.Println("✔ Valid domains written to Input/Malicious_Domains.csv")
}
