// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package cooldown

import (
	"sync"
	"sync/atomic"
	"time"
)

// Manager controls cooldown state and gate blocking.
type Manager struct {
	active int32 // 0 = false, 1 = true
	until  int64 // unix timestamp in nanoseconds

	// gate channel blocks workers while cooldown is active
	gate chan struct{}

	mu        sync.RWMutex
	watchOnce sync.Once
}

// NewManager creates a new cooldown manager.
func NewManager() *Manager {
	return &Manager{
		active: 0,
		until:  0,
		gate:   make(chan struct{}),
	}
}

// Trigger activates cooldown for durSeconds (precise, millisecond-safe).
func (m *Manager) Trigger(durSeconds int64) {
	if durSeconds <= 0 {
		return
	}

	expireAt := time.Now().Add(time.Duration(durSeconds) * time.Second).UnixNano()
	m.mu.Lock()
	defer m.mu.Unlock()

	if atomic.LoadInt32(&m.active) == 0 {
		// transition into cooldown state with a fresh gate
		m.gate = make(chan struct{})
	}
	if expireAt > atomic.LoadInt64(&m.until) {
		atomic.StoreInt64(&m.until, expireAt)
	}
	atomic.StoreInt32(&m.active, 1)
}

// Active returns true if cooldown is active.
func (m *Manager) Active() bool {
	return atomic.LoadInt32(&m.active) == 1
}

// Remaining returns seconds left in cooldown (rounded down).
func (m *Manager) Remaining() int64 {
	if !m.Active() {
		return 0
	}

	now := time.Now().UnixNano()
	until := atomic.LoadInt64(&m.until)

	if now >= until {
		return 0
	}

	return int64(time.Until(time.Unix(0, until)).Seconds())
}

// Gate returns the worker-block channel.
func (m *Manager) Gate() chan struct{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.gate
}

// StartWatcher monitors cooldown expiration and releases the gate.
func (m *Manager) StartWatcher() {
	m.watchOnce.Do(func() {
		ticker := time.NewTicker(50 * time.Millisecond) // faster precision
		go func() {
			defer ticker.Stop()
			for range ticker.C {
				if !m.Active() {
					continue
				}
				if time.Now().UnixNano() < atomic.LoadInt64(&m.until) {
					continue
				}
				m.mu.Lock()
				if atomic.LoadInt32(&m.active) == 1 &&
					time.Now().UnixNano() >= atomic.LoadInt64(&m.until) {
					atomic.StoreInt32(&m.active, 0)
					close(m.gate)
				}
				m.mu.Unlock()
			}
		}()
	})
}
