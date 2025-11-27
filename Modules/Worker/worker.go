// worker.go: Main Function and Public API

package worker

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// ---------------------------------------------------------
// UNCHANGEABLE API FUNCTION
// ---------------------------------------------------------

// Note: third parameter name changed from '_' -> rdb so we can store the
// injected RedisStore in the WorkerPool. Type/signature otherwise unchanged.
func NewWorkerPool(opts *RunOptions, conc int, rdb RedisStore) *WorkerPool {
	if opts == nil {
		opts = &RunOptions{}
	}
	if opts.Timeout <= 0 {
		opts.Timeout = DefaultTimeout
	}
	if opts.EvalInterval <= 0 {
		opts.EvalInterval = DefaultEvalInterval
	}
	if opts.ScaleUpThreshold <= 0 {
		opts.ScaleUpThreshold = DefaultScaleUp
	}
	if opts.ScaleDownThreshold <= 0 {
		opts.ScaleDownThreshold = DefaultScaleDown
	}
	if opts.IdleGracePeriod <= 0 {
		opts.IdleGracePeriod = DefaultIdleGrace
	}
	if opts.MinWorkers < 1 {
		opts.MinWorkers = 1
	}

	if conc <= 0 {
		conc = runtime.NumCPU()
	}

	if opts.MaxWorkers <= 0 {
		opts.MaxWorkers = conc * 2
	}
	if conc < opts.MinWorkers {
		conc = opts.MinWorkers
	}
	if conc > opts.MaxWorkers {
		conc = opts.MaxWorkers
	}

	ctx, cancel := context.WithCancel(context.Background())
	wp := &WorkerPool{
		options:     opts,
		ctx:         ctx,
		cancel:      cancel,
		inflight:    make(map[string]int64),
		monitorStop: make(chan struct{}),
		redis:       rdb, // store injected Redis instance (may be nil)
	}

	// initialize workers
	for i := 0; i < conc; i++ {
		wp.addWorker()
	}

	if opts.AutoScale {
		go wp.monitor()
	}

	return wp
}

func (wp *WorkerPool) SubmitTask(f TaskFunc, p TaskPriority, weight int) (int64, <-chan WorkerResult, error) {
	if f == nil {
		return 0, nil, errors.New("nil task")
	}

	wp.mu.Lock()
	defer wp.mu.Unlock()

	if len(wp.workers) == 0 {
		return 0, nil, errors.New("no workers")
	}

	key := computeDedupeKey(wp.options, f)
	if key != "" {
		if id, ok := wp.inflight[key]; ok {
			// find existing task channel
			for _, w := range wp.workers {
				w.mu.Lock()
				for _, t := range w.taskQueue {
					if t.ID == id {
						ch := t.ResultCh
						w.mu.Unlock()
						return id, ch, nil
					}
				}
				w.mu.Unlock()
			}
			// fallback: inflight entry exists but task not local (rare)
		}
	}

	id := atomic.AddInt64(&wp.nextID, 1)
	ch := make(chan WorkerResult, 1)
	task := &Task{
		ID:       id,
		Func:     f,
		Priority: p,
		ResultCh: ch,
		Weight:   weight,
		Created:  time.Now(),
		Dedupe:   key,
	}

	if key != "" {
		wp.inflight[key] = id

		// Best-effort: mark inflight in Redis so other processes can see the dedupe.
		// We don't block on Redis — it's a helper for distributed dedupe only.
		if wp.redis != nil {
			dedupeKey := fmt.Sprintf("inflight:%s", key)
			go func(dk string) {
				ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
				// store id as string; set a reasonable TTL so abandoned entries expire.
				_ = wp.redis.SetValue(ctx, dk, fmt.Sprintf("%d", id), 10*time.Minute)
				cancel()
			}(dedupeKey)
		}
	}

	// round-robin distribution
	if len(wp.workers) == 0 {
		// shouldn't happen, but guard
		return 0, nil, errors.New("no available workers")
	}
	wp.rrIndex = (wp.rrIndex + 1) % len(wp.workers)
	w := wp.workers[wp.rrIndex]

	w.mu.Lock()
	heap.Push(&w.taskQueue, task)
	w.cond.Signal()
	w.mu.Unlock()

	return id, ch, nil
}

// Stop gracefully shuts down the pool.
func (wp *WorkerPool) Stop() {
	// delegate to internal shutdown helper
	stopWorkerPoolInternal(wp)
}

func (wp *WorkerPool) addWorker() {
	w := &workerInfo{
		ID:   len(wp.workers),
		stop: make(chan struct{}),
		wg:   &wp.wg,
	}
	w.cond = sync.NewCond(&w.mu)
	heap.Init(&w.taskQueue)

	wp.mu.Lock()
	wp.workers = append(wp.workers, w)
	wp.mu.Unlock()

	wp.wg.Add(1)
	go w.run(wp)
}

func (w *workerInfo) run(wp *WorkerPool) {
	defer w.wg.Done()
	for {
		w.mu.Lock()
		for w.taskQueue.Len() == 0 && !w.closing {
			select {
			case <-wp.ctx.Done():
				w.mu.Unlock()
				return
			case <-w.stop:
				w.mu.Unlock()
				return
			default:
			}
			w.cond.Wait()
		}
		if w.closing {
			w.mu.Unlock()
			return
		}
		item := heap.Pop(&w.taskQueue)
		w.mu.Unlock()

		if item == nil {
			continue
		}
		if t, ok := item.(*Task); ok {
			// wp.exec is implemented in worker_internal.go
			wp.exec(w, t)
		}
	}
}

func (wp *WorkerPool) log(msg string) {
	if wp.options == nil || wp.options.LogChannel == nil {
		return
	}
	logMessage(wp.options.LogChannel, msg, wp.options.NonBlockingLogs)
}
