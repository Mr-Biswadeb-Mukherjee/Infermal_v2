// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package apis

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
)

type sessionManagerTestRuntime struct{}

func (sessionManagerTestRuntime) Run(context.Context) error { return nil }

func TestStartSessionFailsBeforeCreatingSessionWhenFactoryErrors(t *testing.T) {
	expected := errors.New("runtime keywords file missing: Input/Keywords.csv")
	manager := NewSessionManager(func() (Runtime, error) {
		return nil, expected
	})

	_, err := manager.StartSession()
	if !errors.Is(err, expected) {
		t.Fatalf("expected %v, got %v", expected, err)
	}
	if _, ok := manager.CurrentSession(); ok {
		t.Fatal("did not expect a session to exist when runtime factory fails")
	}
}

func TestStartSessionBuildsRuntimeOnceAndStartsSession(t *testing.T) {
	var factoryCalls atomic.Int32
	manager := NewSessionManager(func() (Runtime, error) {
		factoryCalls.Add(1)
		return sessionManagerTestRuntime{}, nil
	})

	info, err := manager.StartSession()
	if err != nil {
		t.Fatalf("start session error: %v", err)
	}
	if info.ID == "" {
		t.Fatal("expected session id")
	}
	if got := factoryCalls.Load(); got != 1 {
		t.Fatalf("expected factory to be called once, got %d", got)
	}
}
