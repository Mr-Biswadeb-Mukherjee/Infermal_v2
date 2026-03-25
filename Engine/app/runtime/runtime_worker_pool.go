// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package runtime

import (
	"sync/atomic"
	"time"
)

func newRuntimeWorkerPool(
	cfg Config,
	cache CacheStore,
	factory WorkerPoolFactory,
	workers int,
	tuner *runtimeTuner,
	active *int64,
) WorkerPool {
	opts := &WorkerPoolOptions{
		Timeout:         resolveWorkerTimeout(tuner),
		MaxRetries:      cfg.MaxRetries,
		AutoScale:       cfg.AutoScale,
		MinWorkers:      1,
		NonBlockingLogs: true,
		OnTaskStart:     activeStart(active),
		OnTaskFinish:    activeFinish(active),
	}
	return factory.NewWorkerPool(opts, workers, cache)
}

func resolveWorkerTimeout(tuner *runtimeTuner) time.Duration {
	timeout := tuner.resolveTimeout() * 4
	if timeout < 12*time.Second {
		return 12 * time.Second
	}
	return timeout
}

func activeStart(active *int64) func(int64) {
	return func(int64) {
		atomic.AddInt64(active, 1)
	}
}

func activeFinish(active *int64) func(int64, TaskResult) {
	return func(int64, TaskResult) {
		atomic.AddInt64(active, -1)
	}
}
