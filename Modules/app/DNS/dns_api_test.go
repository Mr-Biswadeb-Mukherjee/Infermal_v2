// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package dns_test

import (
	"context"
	"strings"
	"testing"
	"time"

	dns "github.com/Mr-Biswadeb-Mukherjee/Infermal_v2/Modules/app/DNS"
)

func TestStableAPI(t *testing.T) {
	cfg := dns.Config{
		Upstream:  "1.1.1.1:53",
		Retries:   2,
		TimeoutMS: 1200,
		DelayMS:   80,
	}

	if _, err := dns.InitDNS(cfg); err != nil {
		t.Fatalf("InitDNS failed: %v", err)
	}
	if err := dns.Health(); err != nil {
		t.Fatalf("Health failed: %v", err)
	}

	requireLookupOrSkip(t, "google.com", dns.ResolveDomain)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	requireLookupOrSkip(t, "cloudflare.com", func(domain string) (bool, error) {
		return dns.ResolveWithContext(ctx, domain)
	})
}

func requireLookupOrSkip(t *testing.T, domain string, lookup func(string) (bool, error)) {
	t.Helper()

	ok, err := lookup(domain)
	if err != nil && isEnvironmentDNSError(err) {
		t.Skipf("skipping external DNS integration check for %s: %v", domain, err)
	}
	if err != nil {
		t.Fatalf("lookup failed for %s: %v", domain, err)
	}
	if !ok {
		t.Fatalf("expected resolution success for %s", domain)
	}
}

func isEnvironmentDNSError(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "temporary failure in name resolution") ||
		strings.Contains(msg, "no such host") ||
		strings.Contains(msg, "i/o timeout") ||
		strings.Contains(msg, "connection refused") ||
		strings.Contains(msg, "network is unreachable")
}
