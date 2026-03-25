// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

//go:build ignore
// +build ignore

package rresolver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	mdns "github.com/miekg/dns"
)

// Resolver holds server configuration and cache.
type Resolver struct {
	Addr        string // listen address, e.g. \":53\"
	Timeout     time.Duration
	client      *mdns.Client
	cache       *cache
	listenMux   sync.Mutex
	rootServers []string // populated by LoadRootHints
}

func NewResolver(listenAddr string) *Resolver {
	r := &Resolver{
		Addr:    listenAddr,
		Timeout: 3 * time.Second,
		client: &mdns.Client{
			Net:     "udp",
			Timeout: 3 * time.Second,
		},
		cache: newCache(),
	}

	// auto-load root hints
	_ = r.LoadRootHints(resolveRootHintsPath())

	return r
}

// Start starts both UDP and TCP servers and blocks.
func (r *Resolver) Start() error {
	r.listenMux.Lock()
	defer r.listenMux.Unlock()

	// UDP server
	udpServer := &mdns.Server{Addr: r.Addr, Net: "udp"}
	tcpServer := &mdns.Server{Addr: r.Addr, Net: "tcp"}

	mdns.HandleFunc(".", r.handleQuery)

	errCh := make(chan error, 2)
	go func() { errCh <- udpServer.ListenAndServe() }()
	go func() { errCh <- tcpServer.ListenAndServe() }()

	// return on first error (or nil if servers run forever)
	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			// shutdown both and return error
			udpServer.Shutdown()
			tcpServer.Shutdown()
			return err
		}
	}
	return nil
}

func (r *Resolver) handleQuery(w mdns.ResponseWriter, req *mdns.Msg) {
	ctx := context.Background()
	if len(req.Question) == 0 {
		m := new(mdns.Msg)
		m.SetRcode(req, mdns.RcodeFormatError)
		w.WriteMsg(m)
		return
	}

	q := req.Question[0]
	qtype := q.Qtype

	resp := new(mdns.Msg)
	resp.SetReply(req)

	// try cache
	if answers := r.cache.get(q.Name, qtype); answers != nil {
		resp.Answer = append(resp.Answer, answers...)
		w.WriteMsg(resp)
		return
	}

	ans, err := r.Resolve(ctx, q.Name, qtype)
	if err != nil {
		resp.SetRcode(req, mdns.RcodeServerFailure)
		w.WriteMsg(resp)
		return
	}

	resp.Answer = append(resp.Answer, ans...)
	// store to cache
	if len(ans) > 0 {
		r.cache.set(q.Name, qtype, ans)
	}

	w.WriteMsg(resp)
}

// Resolve performs an iterative (recursive) resolution for a single question.
func (r *Resolver) Resolve(ctx context.Context, qname string, qtype uint16) ([]mdns.RR, error) {
	name := dnsFqdn(qname)
	q := newIterativeQuery(name, qtype)
	servers := append([]string{}, r.rootServers...)

	for {
		resp, err := r.queryServers(ctx, servers, q)
		if err != nil {
			return nil, err
		}
		if answers, ok := r.resolveAnswerSet(ctx, resp, qname, qtype); ok {
			return answers, nil
		}
		servers, err = referralServers(ctx, resp)
		if err != nil {
			return nil, err
		}
	}
}

func newIterativeQuery(name string, qtype uint16) *mdns.Msg {
	q := new(mdns.Msg)
	q.Id = 0
	q.RecursionDesired = false
	q.Question = []mdns.Question{{Name: name, Qtype: qtype, Qclass: mdns.ClassINET}}
	return q
}

func (r *Resolver) queryServers(ctx context.Context, servers []string, q *mdns.Msg) (*mdns.Msg, error) {
	var (
		resp    *mdns.Msg
		lastErr error
	)
	for _, srv := range servers {
		resp, _, lastErr = r.tryQuery(ctx, srv, q)
		if lastErr == nil && resp != nil {
			return resp, nil
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("querying servers: %w", lastErr)
	}
	return nil, errors.New("no response from nameservers")
}

func (r *Resolver) resolveAnswerSet(
	ctx context.Context,
	resp *mdns.Msg,
	qname string,
	qtype uint16,
) ([]mdns.RR, bool) {
	if len(resp.Answer) == 0 {
		return nil, false
	}
	answers := extractRecords(resp, qname, qtype)
	final := make([]mdns.RR, 0, len(answers))
	for _, a := range answers {
		final = append(final, a)
		if a.Header().Rrtype != mdns.TypeCNAME {
			continue
		}
		cname := strings.TrimSuffix(strings.TrimSpace(a.(*mdns.CNAME).Target), ".")
		if cname == "" || strings.EqualFold(cname, qname) {
			continue
		}
		targetAnswers, err := r.Resolve(ctx, cname, qtype)
		if err == nil {
			final = append(final, targetAnswers...)
		}
	}
	return final, true
}

func referralServers(ctx context.Context, resp *mdns.Msg) ([]string, error) {
	ns := extractNameservers(resp)
	if len(ns) == 0 {
		return nil, errors.New("no answers and no referral")
	}
	servers := resolveServersFromMsg(resp, ns)
	if len(servers) > 0 {
		return servers, nil
	}
	servers = resolveServersFromSystem(ctx, ns)
	if len(servers) == 0 {
		return nil, errors.New("could not build list of nameserver addresses")
	}
	return servers, nil
}

func resolveServersFromSystem(ctx context.Context, ns []string) []string {
	servers := make([]string, 0, len(ns))
	for _, n := range ns {
		ips, err := net.DefaultResolver.LookupHost(ctx, n)
		if err != nil {
			continue
		}
		for _, ip := range ips {
			servers = append(servers, net.JoinHostPort(ip, "53"))
		}
	}
	return servers
}

func (r *Resolver) tryQuery(ctx context.Context, server string, q *mdns.Msg) (*mdns.Msg, string, error) {
	// try UDP first
	cli := &mdns.Client{Net: "udp", Timeout: r.Timeout}
	in, _, err := cli.ExchangeContext(ctx, q, server)
	if err == nil && in != nil && in.Rcode == mdns.RcodeSuccess {
		return in, server, nil
	}
	// try TCP as fallback
	cli.Net = "tcp"
	in, _, err = cli.ExchangeContext(ctx, q, server)
	if err == nil && in != nil && in.Rcode == mdns.RcodeSuccess {
		return in, server, nil
	}
	return in, server, err
}
