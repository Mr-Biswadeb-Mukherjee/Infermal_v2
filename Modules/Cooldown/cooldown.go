package cooldown

import (
	"sync/atomic"
	"time"
)

// Manager controls cooldown state and gate blocking.
type Manager struct {
	active int32 // 0 = false, 1 = true
	until  int64 // unix timestamp

	// gate channel blocks workers while cooldown is active
	gate chan struct{}
}

// NewManager creates a new cooldown manager.
func NewManager() *Manager {
	return &Manager{
		active: 0,
		until:  0,
		gate:   make(chan struct{}),
	}
}

// Trigger activates cooldown for durSeconds.
func (m *Manager) Trigger(durSeconds int64) {
	atomic.StoreInt32(&m.active, 1)
	atomic.StoreInt64(&m.until, time.Now().Unix()+durSeconds)
	m.gate = make(chan struct{}) // recreate gate so workers block
}

// Active returns true if cooldown is active.
func (m *Manager) Active() bool {
	return atomic.LoadInt32(&m.active) == 1
}

// Remaining returns seconds left in cooldown.
func (m *Manager) Remaining() int64 {
	if !m.Active() {
		return 0
	}
	rem := atomic.LoadInt64(&m.until) - time.Now().Unix()
	if rem < 0 {
		return 0
	}
	return rem
}

// Gate returns the worker-block channel.
func (m *Manager) Gate() chan struct{} {
	return m.gate
}

// StartWatcher monitors cooldown expiration and releases the gate.
func (m *Manager) StartWatcher() {
	ticker := time.NewTicker(200 * time.Millisecond)

	go func() {
		defer ticker.Stop()
		for range ticker.C {
			if m.Active() {
				// Cooldown expired
				if time.Now().Unix() >= atomic.LoadInt64(&m.until) {
					atomic.StoreInt32(&m.active, 0)
					close(m.gate) // release any blocked workers
				}
			}
		}
	}()
}
