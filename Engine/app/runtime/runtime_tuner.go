// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package runtime

import (
	"context"
	"sync/atomic"
	"time"
)

type runtimeCounters struct {
	submitted *int64
	completed *int64
	active    *int64
}

type runtimeLogger interface {
	Info(format string, v ...interface{})
	Warning(format string, v ...interface{})
}

type runtimeTuner struct {
	limiter      RateLimiter
	baseRate     int64
	rateCap      int64
	cooldownEach int64
	cooldownFor  int64
	lastCooldown int64
	timeoutNanos atomic.Int64
	logTick      atomic.Int64
}

func newRuntimeTuner(
	cfg Config,
	total int64,
	workers int,
	factory AdaptiveFactory,
	limiter RateLimiter,
) *runtimeTuner {
	_ = factory
	seedRate := seedRateLimit(cfg.RateLimit, total, workers, cfg.RateLimitCeiling)
	seedTimeout := seedTimeoutValue(cfg.TimeoutSeconds, total)

	tuner := &runtimeTuner{
		limiter:  limiter,
		baseRate: seedRate,
		rateCap:  normalizeRateLimitCap(cfg.RateLimitCeiling),
		cooldownEach: normalizeCooldownAfter(
			cfg.CooldownAfter,
		),
		cooldownFor: normalizeCooldownDuration(cfg.CooldownDuration),
	}
	tuner.timeoutNanos.Store(seedTimeout.Nanoseconds())
	return tuner
}

func (t *runtimeTuner) resolveTimeout() time.Duration {
	nanos := t.timeoutNanos.Load()
	if nanos <= 0 {
		return 2 * time.Second
	}
	return time.Duration(nanos)
}

func (t *runtimeTuner) observeTask(latency time.Duration, resolveErr error, denied int64) {
	_ = latency
	_ = resolveErr
	_ = denied
}

func (t *runtimeTuner) observeLimiterError() {
}

func (t *runtimeTuner) run(
	ctx context.Context,
	cdm CooldownManager,
	counters runtimeCounters,
	log runtimeLogger,
) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			snap, completed := buildSnapshot(counters, 0)
			decision := AdaptiveDecision{
				RateLimit: clampRateLimit(t.baseRate, t.rateCap),
				Timeout:   t.resolveTimeout(),
			}
			if !cdm.Active() && t.shouldTriggerCooldown(completed) {
				cdm.Trigger(t.cooldownFor)
				decision.Cooldown = time.Duration(t.cooldownFor) * time.Second
			}
			t.logDecision(log, decision, snap)
		}
	}
}

func (t *runtimeTuner) shouldTriggerCooldown(completed int64) bool {
	if t.cooldownEach <= 0 || t.cooldownFor <= 0 {
		return false
	}
	if completed <= 0 || completed%t.cooldownEach != 0 {
		return false
	}
	if t.lastCooldown == completed {
		return false
	}
	t.lastCooldown = completed
	return true
}

func (t *runtimeTuner) logDecision(log runtimeLogger, d AdaptiveDecision, s AdaptiveSnapshot) {
	if log == nil {
		return
	}

	tick := t.logTick.Add(1)
	if d.Cooldown == 0 && tick%6 != 0 {
		return
	}

	log.Info(
		"static control rate=%d timeout=%s queue=%d inflight=%d active=%d cooldown=%s",
		d.RateLimit,
		d.Timeout,
		s.QueueDepth,
		s.InFlight,
		s.ActiveWorkers,
		d.Cooldown,
	)
}

func buildSnapshot(c runtimeCounters, lastCompleted int64) (AdaptiveSnapshot, int64) {
	submitted := atomic.LoadInt64(c.submitted)
	completed := atomic.LoadInt64(c.completed)
	active := atomic.LoadInt64(c.active)
	if active < 0 {
		active = 0
	}

	inFlight := submitted - completed
	if inFlight < 0 {
		inFlight = 0
	}

	queueDepth := inFlight - active
	if queueDepth < 0 {
		queueDepth = 0
	}

	delta := completed - lastCompleted
	if delta < 0 {
		delta = 0
	}

	return AdaptiveSnapshot{
		QueueDepth:     queueDepth,
		InFlight:       inFlight,
		ActiveWorkers:  active,
		CompletedDelta: delta,
	}, completed
}

func seedRateLimit(configRate int, total int64, workers int, rateLimitCeiling int) int64 {
	if configRate > 0 {
		return clampRateLimit(int64(configRate), normalizeRateLimitCap(rateLimitCeiling))
	}
	base := int64(maxInt(1, workers) * 8)
	rate := base
	switch {
	case total >= 100000:
		rate = base * 3
	case total >= 30000:
		rate = base * 2
	}
	return clampRateLimit(rate, normalizeRateLimitCap(rateLimitCeiling))
}

func seedTimeoutValue(configTimeout int, total int64) time.Duration {
	if configTimeout > 0 {
		return time.Duration(configTimeout) * time.Second
	}
	switch {
	case total >= 100000:
		return 4 * time.Second
	case total >= 30000:
		return 3 * time.Second
	default:
		return 2 * time.Second
	}
}

func ceilSeconds(d time.Duration) int64 {
	if d <= 0 {
		return 0
	}
	seconds := d / time.Second
	if d%time.Second == 0 {
		return int64(seconds)
	}
	return int64(seconds + 1)
}

func maxInt(a, b int) int {
	if a >= b {
		return a
	}
	return b
}

func normalizeRateLimitCap(rateLimitCeiling int) int64 {
	if rateLimitCeiling <= 0 {
		return 0
	}
	return int64(rateLimitCeiling)
}

func normalizeCooldownAfter(v int) int64 {
	if v <= 0 {
		return 0
	}
	return int64(v)
}

func normalizeCooldownDuration(v int) int64 {
	if v <= 0 {
		return 0
	}
	return int64(v)
}

func clampRateLimit(rate int64, rateCap int64) int64 {
	if rate < 1 {
		rate = 1
	}
	if rateCap > 0 && rate > rateCap {
		return rateCap
	}
	return rate
}

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
