// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package dns_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	dns "github.com/Mr-Biswadeb-Mukherjee/Infermal_v2/Engine/app/DNS"
)

func TestTimeoutFallbackToBackup(t *testing.T) {
	primary := &mockResolver{delay: 80 * time.Millisecond, err: errors.New("upstream timeout")}
	backup := &mockResolver{shouldResolve: true}
	d := &dns.DNS{}
	d.SwapPrimary(primary)
	d.SwapBackup(backup)

	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
	defer cancel()

	ok, err := d.Resolve(ctx, "fallback-after-timeout.com")
	if err != nil || !ok {
		t.Fatalf("expected backup success after primary timeout, got ok=%v err=%v", ok, err)
	}
	if backup.Calls() != 1 {
		t.Fatalf("expected backup resolver to be called once, got %d", backup.Calls())
	}
}

func TestSystemFallbackAfterUnresolvedOwnResolvers(t *testing.T) {
	primary := &mockResolver{shouldResolve: false}
	backup := &mockResolver{shouldResolve: false}
	system := &mockResolver{shouldResolve: true}
	d := &dns.DNS{}
	d.SwapPrimary(primary)
	d.SwapBackup(backup)
	d.SwapSystem(system)

	ok, err := d.Resolve(context.Background(), "system-fallback.com")
	if err != nil || !ok {
		t.Fatalf("expected system fallback success, got ok=%v err=%v", ok, err)
	}
	if system.Calls() != 1 {
		t.Fatalf("expected system resolver call count 1, got %d", system.Calls())
	}
}

func TestCacheReadTimeoutDoesNotBlockResolution(t *testing.T) {
	d := &dns.DNS{}
	d.AttachCache(&slowReadCache{})
	d.SwapPrimary(&mockResolver{shouldResolve: true})
	start := time.Now()

	ok, err := d.Resolve(context.Background(), "timeout-cache.com")
	elapsed := time.Since(start)
	if err != nil || !ok {
		t.Fatalf("expected resolve success after cache timeout, got ok=%v err=%v", ok, err)
	}
	if elapsed > 350*time.Millisecond {
		t.Fatalf("cache read timeout should be bounded, got %v", elapsed)
	}
}

func TestCacheWriteFailureDoesNotFailLookup(t *testing.T) {
	d := &dns.DNS{}
	d.SwapPrimary(&mockResolver{shouldResolve: true})
	d.AttachCache(&failingWriteCache{})

	ok, err := d.Resolve(context.Background(), "cache-write-failure.com")
	if err != nil || !ok {
		t.Fatalf("cache write failure must not fail lookup, got ok=%v err=%v", ok, err)
	}
}

func TestContextCancelPropagates(t *testing.T) {
	d := &dns.DNS{}
	d.SwapPrimary(&mockResolver{blockUntilDone: true})
	d.SwapBackup(&mockResolver{blockUntilDone: true})
	d.SwapSystem(&mockResolver{blockUntilDone: true})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	ok, err := d.Resolve(ctx, "cancel.com")
	if err == nil || ok {
		t.Fatalf("expected canceled context failure, got ok=%v err=%v", ok, err)
	}
}

func TestResolveLogsFailure(t *testing.T) {
	lg := &captureLogger{}
	d := &dns.DNS{}
	d.AttachLogger(lg)
	d.SwapPrimary(&mockResolver{err: errors.New("dns broken")})
	d.SwapSystem(nil)

	_, _ = d.Resolve(context.Background(), "logging-check.com")
	if !lg.Contains("logging-check.com") {
		t.Fatal("expected warning log to include domain context")
	}
}

func TestConcurrentResolve(t *testing.T) {
	d := &dns.DNS{}
	d.SwapPrimary(&mockResolver{shouldResolve: true})
	d.AttachCache(newMockCache())
	var wg sync.WaitGroup

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ok, err := d.Resolve(context.Background(), "race.com")
			if err != nil || !ok {
				t.Errorf("resolve failed, ok=%v err=%v", ok, err)
			}
		}()
	}
	wg.Wait()
}
