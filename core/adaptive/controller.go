// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

// SPDX-License-Identifier: Apache-2.0

package adaptive

import (
	"sync"
	"time"
)

type Config struct {
	InitialRate      int64
	MinRate          int64
	MaxRate          int64
	InitialTimeout   time.Duration
	MinTimeout       time.Duration
	MaxTimeout       time.Duration
	EvalInterval     time.Duration
	TargetLatency    time.Duration
	CooldownMin      time.Duration
	CooldownMax      time.Duration
	CooldownMinGap   time.Duration
	StallTickTrigger int
}

type Snapshot struct {
	QueueDepth     int64
	InFlight       int64
	ActiveWorkers  int64
	CompletedDelta int64
}

type Decision struct {
	RateLimit int64
	Timeout   time.Duration
	Cooldown  time.Duration
	Pressure  float64
}

type sampleWindow struct {
	tasks        int64
	pressureErrs int64
	rateLimited  int64
	limiterErrs  int64
	latencyNanos int64
}

type Controller struct {
	mu sync.Mutex

	cfg Config

	rateLimit int64
	timeout   time.Duration

	latencyEWMA    float64
	errorEWMA      float64
	rateWaitEWMA   float64
	limiterErrEWMA float64

	lastCooldown time.Time
	stallTicks   int
	window       sampleWindow
}

func DefaultConfig(initialRate int64, initialTimeout time.Duration, workers int64) Config {
	if workers < 1 {
		workers = 1
	}
	if initialRate <= 0 {
		initialRate = workers * 12
	}
	if initialTimeout <= 0 {
		initialTimeout = 3 * time.Second
	}

	maxRate := maxInt64(initialRate*6, workers*64)
	minRate := maxInt64(8, initialRate/4)

	return Config{
		InitialRate:      initialRate,
		MinRate:          minRate,
		MaxRate:          maxRate,
		InitialTimeout:   initialTimeout,
		MinTimeout:       900 * time.Millisecond,
		MaxTimeout:       12 * time.Second,
		EvalInterval:     1 * time.Second,
		TargetLatency:    800 * time.Millisecond,
		CooldownMin:      2 * time.Second,
		CooldownMax:      20 * time.Second,
		CooldownMinGap:   10 * time.Second,
		StallTickTrigger: 3,
	}
}

func NewController(cfg Config) *Controller {
	return &Controller{
		cfg:         cfg,
		rateLimit:   cfg.InitialRate,
		timeout:     cfg.InitialTimeout,
		latencyEWMA: float64(cfg.TargetLatency),
	}
}

func (c *Controller) EvalInterval() time.Duration {
	return c.cfg.EvalInterval
}

func (c *Controller) ObserveTask(latency time.Duration, pressureErr bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.window.tasks++
	c.window.latencyNanos += latency.Nanoseconds()

	if pressureErr {
		c.window.pressureErrs++
	}
}

func (c *Controller) ObserveRateLimited(denied int64) {
	if denied <= 0 {
		return
	}

	c.mu.Lock()
	c.window.rateLimited += denied
	c.mu.Unlock()
}

func (c *Controller) ObserveLimiterError() {
	c.mu.Lock()
	c.window.limiterErrs++
	c.mu.Unlock()
}

func (c *Controller) Evaluate(s Snapshot) Decision {
	c.mu.Lock()
	defer c.mu.Unlock()

	w := c.consumeWindowLocked()
	c.updateAveragesLocked(w)
	c.updateStallLocked(s)

	pressure := c.pressureLocked(s)
	rate := c.tuneRateLocked(s, pressure)
	timeout := c.tuneTimeoutLocked(pressure)
	cooldown := c.cooldownLocked(pressure)

	// 🔥 cooldown forces rate reduction
	if cooldown > 0 {
		rate = maxInt64(rate/2, c.cfg.MinRate)
		c.rateLimit = rate
	}

	return Decision{
		RateLimit: rate,
		Timeout:   timeout,
		Cooldown:  cooldown,
		Pressure:  pressure,
	}
}

func (c *Controller) consumeWindowLocked() sampleWindow {
	w := c.window
	c.window = sampleWindow{}
	return w
}