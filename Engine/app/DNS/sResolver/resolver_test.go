package sresolver

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/miekg/dns"
)

type fakeDNSClient struct {
	resp *dns.Msg
	err  error
}

func (c *fakeDNSClient) Exchange(*dns.Msg, string) (*dns.Msg, time.Duration, error) {
	return c.resp, 0, c.err
}

func TestStubResolverValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		r    *StubResolver
	}{
		{name: "empty domain", r: New(WithUpstream("127.0.0.1:53"), WithRetries(1), WithTimeout(time.Second))},
		{name: "missing upstream", r: New(WithRetries(1), WithTimeout(time.Second))},
		{name: "missing retries", r: New(WithUpstream("127.0.0.1:53"), WithTimeout(time.Second))},
		{name: "missing timeout", r: New(WithUpstream("127.0.0.1:53"), WithRetries(1))},
	}

	for _, tt := range tests {
		domain := "example.test"
		if tt.name == "empty domain" {
			domain = ""
		}

		ok, err := tt.r.Resolve(context.Background(), domain)
		if err == nil {
			t.Fatalf("%s: expected error", tt.name)
		}
		if ok {
			t.Fatalf("%s: expected unresolved result", tt.name)
		}
	}
}

func TestStubResolverResolveSuccessWithInjectedClient(t *testing.T) {
	oldClient := newDNSClient
	t.Cleanup(func() {
		newDNSClient = oldClient
	})

	resp := new(dns.Msg)
	resp.Answer = append(resp.Answer, &dns.A{
		Hdr: dns.RR_Header{Name: "example.test.", Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 60},
		A:   net.ParseIP("127.0.0.10"),
	})
	newDNSClient = func(time.Duration) dnsClient {
		return &fakeDNSClient{resp: resp}
	}

	resolver := New(
		WithUpstream("127.0.0.1:53"),
		WithRetries(1),
		WithTimeout(500*time.Millisecond),
	)

	ok, err := resolver.Resolve(context.Background(), "example.test")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected stub resolver to resolve domain")
	}
}

func TestSystemResolverResolveWithInjectedLookup(t *testing.T) {
	oldLookup := lookupIPAddr
	t.Cleanup(func() {
		lookupIPAddr = oldLookup
	})
	lookupIPAddr = func(_ *net.Resolver, _ context.Context, _ string) ([]net.IPAddr, error) {
		return []net.IPAddr{{IP: net.ParseIP("127.0.0.10")}}, nil
	}

	resolver := NewSystem()

	ok, err := resolver.Resolve(context.Background(), "example.test")
	if err != nil {
		t.Fatalf("Resolve returned error: %v", err)
	}
	if !ok {
		t.Fatal("expected system resolver to resolve domain")
	}

	ok, err = resolver.Resolve(context.Background(), "   ")
	if err == nil {
		t.Fatal("expected empty domain error")
	}
	if ok {
		t.Fatal("expected empty domain lookup to be false")
	}
}
