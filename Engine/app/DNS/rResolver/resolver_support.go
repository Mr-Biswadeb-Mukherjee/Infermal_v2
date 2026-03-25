// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

//go:build ignore
// +build ignore

package rresolver

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	mdns "github.com/miekg/dns"
)

// extractRecords filters answers matching qname and qtype (or all types if qtype==ANY).
func extractRecords(m *mdns.Msg, qname string, qtype uint16) []mdns.RR {
	out := make([]mdns.RR, 0)
	trimmedQName := strings.TrimSuffix(qname, ".")

	for _, rr := range m.Answer {
		name := strings.TrimSuffix(rr.Header().Name, ".")
		if !strings.EqualFold(name, trimmedQName) {
			continue
		}
		if qtype == mdns.TypeANY || rr.Header().Rrtype == qtype {
			out = append(out, rr)
		}
	}
	return out
}

// extractNameservers returns NS names from Authority section.
func extractNameservers(m *mdns.Msg) []string {
	var out []string
	for _, rr := range m.Ns {
		ns, ok := rr.(*mdns.NS)
		if !ok {
			continue
		}
		out = append(out, strings.TrimSuffix(ns.Ns, "."))
	}
	return out
}

// resolveServersFromMsg looks into Additional for A/AAAA glue records matching NS names.
func resolveServersFromMsg(m *mdns.Msg, nsNames []string) []string {
	addrs := make(map[string]struct{})
	for _, rr := range m.Extra {
		addGlueAddress(addrs, nsNames, rr)
	}
	return mapKeys(addrs)
}

func addGlueAddress(addrs map[string]struct{}, nsNames []string, rr mdns.RR) {
	switch v := rr.(type) {
	case *mdns.A:
		addIfNSMatch(addrs, nsNames, strings.TrimSuffix(v.Hdr.Name, "."), v.A.String())
	case *mdns.AAAA:
		addIfNSMatch(addrs, nsNames, strings.TrimSuffix(v.Hdr.Name, "."), v.AAAA.String())
	}
}

func addIfNSMatch(addrs map[string]struct{}, nsNames []string, rrName string, ip string) {
	for _, n := range nsNames {
		if strings.EqualFold(rrName, n) {
			addrs[net.JoinHostPort(ip, "53")] = struct{}{}
		}
	}
}

func mapKeys(values map[string]struct{}) []string {
	out := make([]string, 0, len(values))
	for k := range values {
		out = append(out, k)
	}
	return out
}

func dnsFqdn(name string) string {
	if strings.HasSuffix(name, ".") {
		return name
	}
	return name + "."
}

// LoadRootHints parses a BIND-style root hints file and extracts A/AAAA records.
// It populates r.rootServers with addresses in the form "ip:53".
func (r *Resolver) LoadRootHints(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	zp := mdns.NewZoneParser(bytes.NewReader(data), "", path)
	seen := make(map[string]struct{})

	for {
		rr, ok := zp.Next()
		if !ok {
			break
		}
		if addr, ok := rootHintAddress(rr); ok {
			seen[addr] = struct{}{}
		}
	}

	records := mapKeys(seen)
	if len(records) == 0 {
		return fmt.Errorf("no A/AAAA records found in root hints: %s", path)
	}
	r.rootServers = records
	return nil
}

func rootHintAddress(rr mdns.RR) (string, bool) {
	switch v := rr.(type) {
	case *mdns.A:
		return net.JoinHostPort(v.A.String(), "53"), true
	case *mdns.AAAA:
		return net.JoinHostPort(v.AAAA.String(), "53"), true
	}
	return "", false
}

type cacheEntry struct {
	rrs       []mdns.RR
	expiresAt time.Time
}

type cache struct {
	mu    sync.RWMutex
	store map[string]cacheEntry // key: name|type
}

func newCache() *cache {
	return &cache{store: make(map[string]cacheEntry)}
}

func keyFor(name string, qtype uint16) string {
	return strings.ToLower(name) + "|" + fmt.Sprint(qtype)
}

func (c *cache) get(name string, qtype uint16) []mdns.RR {
	k := keyFor(name, qtype)
	c.mu.RLock()
	entry, ok := c.store[k]
	c.mu.RUnlock()
	if !ok {
		return nil
	}
	if time.Now().Before(entry.expiresAt) {
		out := make([]mdns.RR, len(entry.rrs))
		copy(out, entry.rrs)
		return out
	}
	c.mu.Lock()
	delete(c.store, k)
	c.mu.Unlock()
	return nil
}

func (c *cache) set(name string, qtype uint16, rrs []mdns.RR) {
	if len(rrs) == 0 {
		return
	}
	minTTL := uint32(3600)
	for _, r := range rrs {
		if ttl := r.Header().Ttl; ttl < minTTL {
			minTTL = ttl
		}
	}
	if minTTL == 0 {
		minTTL = 30
	}

	exp := time.Now().Add(time.Duration(minTTL) * time.Second)
	copyRRs := make([]mdns.RR, len(rrs))
	copy(copyRRs, rrs)

	c.mu.Lock()
	c.store[keyFor(name, qtype)] = cacheEntry{rrs: copyRRs, expiresAt: exp}
	c.mu.Unlock()
}

func ExampleRun() {
	r := NewResolver(":5353")
	go func() {
		if err := r.Start(); err != nil {
			panic(err)
		}
	}()

	c := &mdns.Client{Net: "udp"}
	m := new(mdns.Msg)
	m.SetQuestion(dnsFqdn("example.com"), mdns.TypeA)
	in, _, err := c.Exchange(m, "127.0.0.1:5353")
	if err != nil {
		fmt.Println("query failed:", err)
	}
	fmt.Println(in)
}
