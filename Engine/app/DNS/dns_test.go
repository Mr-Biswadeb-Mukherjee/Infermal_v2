// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package dns_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	dns "github.com/Mr-Biswadeb-Mukherjee/DIBs/Engine/app/DNS"
)

type mockResolver struct {
	shouldResolve  bool
	err            error
	delay          time.Duration
	blockUntilDone bool
	called         atomic.Int32
}

func (m *mockResolver) Resolve(ctx context.Context, domain string) (bool, error) {
	m.called.Add(1)
	if m.blockUntilDone {
		<-ctx.Done()
		return false, ctx.Err()
	}
	if m.delay > 0 {
		select {
		case <-time.After(m.delay):
		case <-ctx.Done():
			return false, ctx.Err()
		}
	}
	return m.shouldResolve, m.err
}

func (m *mockResolver) Calls() int {
	return int(m.called.Load())
}

type cacheEntry struct {
	val string
	exp time.Time
}

type mockCache struct {
	mu    sync.RWMutex
	store map[string]cacheEntry
}

func newMockCache() *mockCache {
	return &mockCache{store: make(map[string]cacheEntry)}
}

func (c *mockCache) GetValue(ctx context.Context, key string) (string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	e, ok := c.store[key]
	if !ok || time.Now().After(e.exp) {
		return "", errors.New("not found")
	}
	return e.val, nil
}

func (c *mockCache) SetValue(ctx context.Context, key string, v interface{}, ttl time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.store[key] = cacheEntry{val: v.(string), exp: time.Now().Add(ttl)}
	return nil
}

type slowReadCache struct{}

func (c *slowReadCache) GetValue(ctx context.Context, key string) (string, error) {
	<-ctx.Done()
	return "", ctx.Err()
}

func (c *slowReadCache) SetValue(ctx context.Context, key string, v interface{}, ttl time.Duration) error {
	return nil
}

type failingWriteCache struct{}

func (c *failingWriteCache) GetValue(ctx context.Context, key string) (string, error) {
	return "", errors.New("not found")
}

func (c *failingWriteCache) SetValue(ctx context.Context, key string, v interface{}, ttl time.Duration) error {
	return errors.New("redis down")
}

type captureLogger struct {
	mu       sync.Mutex
	warnings []string
}

func (l *captureLogger) Warning(format string, v ...interface{}) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.warnings = append(l.warnings, fmt.Sprintf(format, v...))
}

func (l *captureLogger) Contains(sub string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, w := range l.warnings {
		if strings.Contains(w, sub) {
			return true
		}
	}
	return false
}

func waitForCacheValue(t *testing.T, cache *mockCache, key, want string) {
	t.Helper()
	deadline := time.Now().Add(250 * time.Millisecond)
	for time.Now().Before(deadline) {
		got, err := cache.GetValue(context.Background(), key)
		if err == nil && got == want {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("cache key %s did not reach %q", key, want)
}

func TestInitDNS(t *testing.T) {
	d, err := dns.InitDNS(dns.Config{Upstream: "1.1.1.1:53", Backup: "8.8.8.8:53", Retries: 2, TimeoutMS: 150, DelayMS: 10})
	if err != nil {
		t.Fatalf("InitDNS failed: %v", err)
	}
	if d == nil {
		t.Fatal("nil DNS instance")
	}
}

func TestResolvePrimarySuccess(t *testing.T) {
	primary := &mockResolver{shouldResolve: true}
	d := &dns.DNS{}
	d.SwapPrimary(primary)

	ok, err := d.Resolve(context.Background(), "example.com")
	if err != nil || !ok {
		t.Fatalf("expected success, got ok=%v err=%v", ok, err)
	}
	if primary.Calls() != 1 {
		t.Fatalf("expected primary call count 1, got %d", primary.Calls())
	}
}

func TestResolveSkipsFallbackWhenPrimarySucceeds(t *testing.T) {
	primary := &mockResolver{shouldResolve: true}
	backup := &mockResolver{shouldResolve: true}
	system := &mockResolver{shouldResolve: true}
	d := &dns.DNS{}
	d.SwapPrimary(primary)
	d.SwapBackup(backup)
	d.SwapSystem(system)

	ok, err := d.Resolve(context.Background(), "fast-path.com")
	if err != nil || !ok {
		t.Fatalf("expected primary fast-path success, got ok=%v err=%v", ok, err)
	}
	if backup.Calls() != 0 || system.Calls() != 0 {
		t.Fatal("fallback resolvers should not be called on primary success")
	}
}

func TestResolveFallbackChain(t *testing.T) {
	primary := &mockResolver{err: errors.New("primary fail")}
	backup := &mockResolver{err: errors.New("backup fail")}
	recursive := &mockResolver{shouldResolve: true}
	d := &dns.DNS{}
	d.SwapPrimary(primary)
	d.SwapBackup(backup)
	d.AttachRecursive(recursive)

	ok, err := d.Resolve(context.Background(), "fallback.com")
	if err != nil || !ok {
		t.Fatalf("expected recursive fallback success, got ok=%v err=%v", ok, err)
	}
	if recursive.Calls() != 1 {
		t.Fatalf("expected recursive resolver called once, got %d", recursive.Calls())
	}
}

func TestCacheHitPositiveAndNegative(t *testing.T) {
	cache := newMockCache()
	_ = cache.SetValue(context.Background(), "dns:ok.com", "1", time.Second)
	_ = cache.SetValue(context.Background(), "dns:bad.com", "0", time.Second)
	d := &dns.DNS{}
	d.AttachCache(cache)

	ok, err := d.Resolve(context.Background(), "ok.com")
	if err != nil || !ok {
		t.Fatalf("expected positive cache hit, got ok=%v err=%v", ok, err)
	}
	ok, err = d.Resolve(context.Background(), "bad.com")
	if err != nil || ok {
		t.Fatalf("expected negative cache hit, got ok=%v err=%v", ok, err)
	}
}

func TestNegativeCaching(t *testing.T) {
	d := &dns.DNS{}
	d.SwapPrimary(&mockResolver{err: errors.New("fail")})
	cache := newMockCache()
	d.AttachCache(cache)

	_, _ = d.Resolve(context.Background(), "fail.com")
	waitForCacheValue(t, cache, "dns:fail.com", "0")
}
