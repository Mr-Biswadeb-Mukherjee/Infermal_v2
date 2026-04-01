// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package apis

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrSessionRunning  = errors.New("a session is already running")
	ErrSessionNotFound = errors.New("session not found")
)

type SessionStatus string

const (
	StatusStarting  SessionStatus = "starting"
	StatusRunning   SessionStatus = "running"
	StatusStopping  SessionStatus = "stopping"
	StatusStopped   SessionStatus = "stopped"
	StatusCompleted SessionStatus = "completed"
	StatusFailed    SessionStatus = "failed"
)

type SessionInfo struct {
	ID        string        `json:"id"`
	Status    SessionStatus `json:"status"`
	StartedAt string        `json:"started_at_ist"`
	EndedAt   string        `json:"ended_at_ist,omitempty"`
	Error     string        `json:"error,omitempty"`
}

type Runtime interface {
	Run(ctx context.Context) error
}

type RuntimeFactory func() (Runtime, error)

type sessionRecord struct {
	runtime Runtime
	info    SessionInfo
	cancel  context.CancelFunc
	done    chan struct{}
}

type SessionManager struct {
	mu      sync.RWMutex
	current *sessionRecord
	last    SessionInfo
	hasLast bool
	factory RuntimeFactory
	now     func() time.Time
}

func NewSessionManager(factory RuntimeFactory) *SessionManager {
	return &SessionManager{
		factory: factory,
		now:     time.Now,
	}
}

func (m *SessionManager) StartSession() (SessionInfo, error) {
	if m == nil || m.factory == nil {
		return SessionInfo{}, errors.New("session manager not configured")
	}

	runtime, err := m.factory()
	if err != nil {
		return SessionInfo{}, err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if m.hasActiveLocked() {
		return SessionInfo{}, ErrSessionRunning
	}

	now := m.now().In(apiISTLocation)
	ctx, cancel := context.WithCancel(context.Background())
	rec := &sessionRecord{
		runtime: runtime,
		info: SessionInfo{
			ID:        newSessionID(now),
			Status:    StatusStarting,
			StartedAt: formatAPIISTTimestamp(now),
		},
		cancel: cancel,
		done:   make(chan struct{}),
	}

	m.current = rec
	go m.runSession(ctx, rec)
	return rec.info, nil
}

func (m *SessionManager) StopActiveSession() (SessionInfo, error) {
	m.mu.Lock()
	rec := m.current
	if rec == nil {
		if m.hasLast {
			info := m.last
			m.mu.Unlock()
			return info, nil
		}
		m.mu.Unlock()
		return SessionInfo{}, ErrSessionNotFound
	}

	if rec.info.Status == StatusStarting || rec.info.Status == StatusRunning {
		rec.info.Status = StatusStopping
		cancel := rec.cancel
		info := rec.info
		m.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		return info, nil
	}
	info := rec.info
	m.mu.Unlock()
	return info, nil
}

func (m *SessionManager) CurrentSession() (SessionInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.current != nil {
		return m.current.info, true
	}
	if !m.hasLast {
		return SessionInfo{}, false
	}
	return m.last, true
}

func (m *SessionManager) Shutdown(timeout time.Duration) {
	if m == nil {
		return
	}

	m.mu.RLock()
	rec := m.current
	m.mu.RUnlock()
	if rec == nil {
		return
	}

	done := rec.done
	cancel := rec.cancel
	m.updateCurrentStatus(StatusStopping, "")
	if cancel != nil {
		cancel()
	}
	if done == nil {
		return
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
	}
}

func (m *SessionManager) runSession(ctx context.Context, rec *sessionRecord) {
	defer close(rec.done)
	if rec == nil || rec.runtime == nil {
		m.finishSession(rec, StatusFailed, "runtime initialization failed")
		return
	}

	m.updateCurrentStatus(StatusRunning, "")
	err := rec.runtime.Run(ctx)
	if err == nil {
		m.finishSession(rec, StatusCompleted, "")
		return
	}
	if ctx.Err() != nil {
		m.finishSession(rec, StatusStopped, "")
		return
	}
	m.finishSession(rec, StatusFailed, err.Error())
}

func (m *SessionManager) finishSession(rec *sessionRecord, status SessionStatus, errText string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if rec == nil {
		return
	}

	rec.info.Status = status
	rec.info.EndedAt = formatAPIISTTimestamp(m.now())
	rec.info.Error = errText
	m.last = rec.info
	m.hasLast = true
	if m.current == rec {
		m.current = nil
	}
}

func (m *SessionManager) updateCurrentStatus(status SessionStatus, errText string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.current == nil {
		return
	}
	m.current.info.Status = status
	m.current.info.Error = errText
}

func (m *SessionManager) hasActiveLocked() bool {
	if m.current == nil {
		return false
	}
	return m.current.info.Status == StatusStarting ||
		m.current.info.Status == StatusRunning ||
		m.current.info.Status == StatusStopping
}

func (s SessionStatus) IsTerminal() bool {
	return s == StatusStopped || s == StatusCompleted || s == StatusFailed
}

func newSessionID(now time.Time) string {
	return fmt.Sprintf("sess-%d", now.UnixNano())
}
