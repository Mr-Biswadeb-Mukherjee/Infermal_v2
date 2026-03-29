// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package dns_test

import (
	"context"
	"errors"
	"testing"
	"time"

	dns "github.com/Mr-Biswadeb-Mukherjee/DIBs/Engine/app/DNS"
)

func TestResolveSystemFallback(t *testing.T) {
	mPrimary := &mockResolver{shouldResolve: false, err: errors.New("primary fail")}
	mBackup := &mockResolver{shouldResolve: false, err: errors.New("backup fail")}
	mSystem := &mockResolver{shouldResolve: true}

	d := &dns.DNS{}
	d.SwapPrimary(mPrimary)
	d.SwapBackup(mBackup)
	d.SwapSystem(mSystem)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ok, err := d.Resolve(ctx, "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected system resolver fallback success")
	}
	if mSystem.Calls() == 0 {
		t.Fatal("system resolver was not called")
	}
}

func TestResolveSystemOnly(t *testing.T) {
	mSystem := &mockResolver{shouldResolve: true}

	d := &dns.DNS{}
	d.SwapSystem(mSystem)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ok, err := d.Resolve(ctx, "example.org")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ok {
		t.Fatal("expected system-only resolution success")
	}
	if mSystem.Calls() == 0 {
		t.Fatal("system resolver not called")
	}
}

func TestResolveSystemFallbackReturnsJoinedError(t *testing.T) {
	errPrimary := errors.New("primary fail")
	errSystem := errors.New("system fail")

	mPrimary := &mockResolver{shouldResolve: false, err: errPrimary}
	mSystem := &mockResolver{shouldResolve: false, err: errSystem}

	d := &dns.DNS{}
	d.SwapPrimary(mPrimary)
	d.SwapSystem(mSystem)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	ok, err := d.Resolve(ctx, "example.net")
	if ok {
		t.Fatal("expected resolution failure")
	}
	if err == nil {
		t.Fatal("expected combined error")
	}
	if !errors.Is(err, errPrimary) {
		t.Fatal("expected primary error in chain")
	}
	if !errors.Is(err, errSystem) {
		t.Fatal("expected system error in chain")
	}
}

func TestInitDNSAllowsSystemFallbackWithoutUpstream(t *testing.T) {
	if _, err := dns.InitDNS(dns.Config{}); err != nil {
		t.Fatalf("InitDNS should work with system fallback only: %v", err)
	}

	if err := dns.Health(); err != nil {
		t.Fatalf("Health should pass with system fallback: %v", err)
	}
}
