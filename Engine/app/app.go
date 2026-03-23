// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package app

import (
	"context"
	"time"

	dns "github.com/Mr-Biswadeb-Mukherjee/Infermal_v2/Engine/app/DNS"
	recon "github.com/Mr-Biswadeb-Mukherjee/Infermal_v2/Engine/app/Recon"
	"github.com/Mr-Biswadeb-Mukherjee/Infermal_v2/Engine/app/intel"
)

type GeneratedDomain struct {
	Domain      string
	RiskScore   float64
	Confidence  string
	GeneratedBy string
}

type IntelDomain struct {
	Name string
}

type IntelRecord struct {
	Domain    string
	A         []string
	AAAA      []string
	CNAME     []string
	NS        []string
	MX        []string
	TXT       []string
	Providers []string
}

type DNSIntelService interface {
	Run(ctx context.Context, domains []IntelDomain) ([]IntelRecord, error)
}

func DNSConfig(cfg Config) dns.Config {
	return dns.Config{
		Upstream:  cfg.UpstreamDNS,
		Backup:    cfg.BackupDNS,
		Retries:   int(cfg.DNSRetries),
		TimeoutMS: cfg.DNSTimeoutMS,
	}
}

func NewResolver(cfg Config, dnsLog ModuleLogger) recon.DNS {
	dnsCfg := DNSConfig(cfg)
	return recon.New(
		dnsCfg.Upstream,
		dnsCfg.Backup,
		dnsCfg.Retries,
		int(dnsCfg.TimeoutMS),
		dnsLog,
	)
}

func GenerateDomains(path string) ([]GeneratedDomain, error) {
	scored, err := recon.GenerateScoredDomains(path)
	if err != nil {
		return nil, err
	}
	out := make([]GeneratedDomain, 0, len(scored))
	for _, item := range scored {
		out = append(out, GeneratedDomain{
			Domain:      item.Domain,
			RiskScore:   item.RiskScore,
			Confidence:  item.Confidence,
			GeneratedBy: item.GeneratedBy,
		})
	}
	return out, nil
}

func NewDNSIntelService(dnsTimeoutMS int64) DNSIntelService {
	svc := intel.NewDefaultDNSIntelService(1, DNSIntelLookupTimeout(dnsTimeoutMS))
	return intelServiceAdapter{inner: svc}
}

func DNSIntelLookupTimeout(dnsTimeoutMS int64) time.Duration {
	if dnsTimeoutMS <= 0 {
		return 3 * time.Second
	}
	timeout := time.Duration(dnsTimeoutMS) * 6 * time.Millisecond
	if timeout < 2*time.Second {
		return 2 * time.Second
	}
	if timeout > 8*time.Second {
		return 8 * time.Second
	}
	return timeout
}

type intelServiceAdapter struct {
	inner *intel.DNSIntelService
}

func (a intelServiceAdapter) Run(ctx context.Context, domains []IntelDomain) ([]IntelRecord, error) {
	if a.inner == nil {
		return nil, nil
	}

	in := make([]intel.Domain, 0, len(domains))
	for _, domain := range domains {
		in = append(in, intel.Domain{Name: domain.Name})
	}

	records, err := a.inner.Run(ctx, in)
	if err != nil {
		return nil, err
	}
	return mapIntelRecords(records), nil
}

func mapIntelRecords(records []intel.Record) []IntelRecord {
	out := make([]IntelRecord, 0, len(records))
	for _, rec := range records {
		out = append(out, IntelRecord{
			Domain:    rec.Domain,
			A:         rec.A,
			AAAA:      rec.AAAA,
			CNAME:     rec.CNAME,
			NS:        rec.NS,
			MX:        rec.MX,
			TXT:       rec.TXT,
			Providers: rec.Providers,
		})
	}
	return out
}
