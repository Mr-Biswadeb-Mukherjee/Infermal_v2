// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package engine

import (
	"context"
	"time"

	app "github.com/Mr-Biswadeb-Mukherjee/Infermal_v2/Engine/app"
	appintel "github.com/Mr-Biswadeb-Mukherjee/Infermal_v2/Engine/app/intel"
	runtime "github.com/Mr-Biswadeb-Mukherjee/Infermal_v2/Engine/runtime"
)

func buildRuntimeDependencies(deps Dependencies, modules runtime.ModuleFactory) runtime.Dependencies {
	return runtime.Dependencies{
		Config: runtime.Config{
			RateLimit:      deps.Config.RateLimit,
			TimeoutSeconds: deps.Config.TimeoutSeconds,
			MaxRetries:     deps.Config.MaxRetries,
			AutoScale:      deps.Config.AutoScale,
			UpstreamDNS:    deps.Config.UpstreamDNS,
			BackupDNS:      deps.Config.BackupDNS,
			DNSRetries:     deps.Config.DNSRetries,
			DNSTimeoutMS:   deps.Config.DNSTimeoutMS,
		},
		Paths: runtime.Paths{
			KeywordsCSV:     deps.Paths.KeywordsCSV,
			DNSIntelOutput:  deps.Paths.DNSIntelOutput,
			GeneratedOutput: deps.Paths.GeneratedOutput,
		},
		Startup: deps.Startup,
		Logs: runtime.LogSet{
			App:         deps.Logs.App,
			DNS:         deps.Logs.DNS,
			RateLimiter: deps.Logs.RateLimiter,
		},
		Cache:   deps.Cache,
		Limiter: deps.Limiter,
		InitLimiter: func(_ runtime.CacheStore, window time.Duration, maxHits int64, logger runtime.ModuleLogger) {
			deps.Limiter.Init(deps.Cache, window, maxHits, toAppLogger(logger, deps.Logs.RateLimiter))
		},
		WorkerPools: workerPoolFactoryAdapter{inner: deps.WorkerPools},
		Cooldowns:   cooldownFactoryAdapter{inner: deps.Cooldowns},
		Adaptive:    adaptiveFactoryAdapter{inner: deps.Adaptive},
		Writers:     writerFactoryAdapter{inner: deps.Writers},
		Modules:     modules,
		PrintLine:   printLine,
	}
}

type appResolverBuilder func(
	cfg app.Config,
	dnsLog app.ModuleLogger,
) interface {
	Resolve(ctx context.Context, domain string) (bool, error)
}

type appDomainGenerator func(path string) ([]app.GeneratedDomain, error)

type appIntelServiceBuilder func(dnsTimeoutMS int64) *appintel.DNSIntelService

type moduleFactory struct {
	newResolver     appResolverBuilder
	generateDomains appDomainGenerator
	newDNSIntelSVC  appIntelServiceBuilder
}

func newModuleFactory(
	resolver appResolverBuilder,
	generator appDomainGenerator,
	intelSVC appIntelServiceBuilder,
) runtime.ModuleFactory {
	return moduleFactory{
		newResolver:     resolver,
		generateDomains: generator,
		newDNSIntelSVC:  intelSVC,
	}
}

func (m moduleFactory) NewResolver(cfg runtime.Config, dnsLog runtime.ModuleLogger) runtime.DNSResolver {
	appCfg := app.Config{
		UpstreamDNS:  cfg.UpstreamDNS,
		BackupDNS:    cfg.BackupDNS,
		DNSRetries:   cfg.DNSRetries,
		DNSTimeoutMS: cfg.DNSTimeoutMS,
	}
	return m.newResolver(appCfg, toAppLogger(dnsLog, nil))
}

func (m moduleFactory) GenerateDomains(path string) ([]runtime.GeneratedDomain, error) {
	items, err := m.generateDomains(path)
	if err != nil {
		return nil, err
	}

	out := make([]runtime.GeneratedDomain, 0, len(items))
	for _, item := range items {
		out = append(out, runtime.GeneratedDomain{
			Domain:      item.Domain,
			RiskScore:   item.RiskScore,
			Confidence:  item.Confidence,
			GeneratedBy: item.GeneratedBy,
		})
	}
	return out, nil
}

func (m moduleFactory) NewDNSIntelService(dnsTimeoutMS int64) runtime.DNSIntelService {
	return intelServiceAdapter{inner: m.newDNSIntelSVC(dnsTimeoutMS)}
}

type intelServiceAdapter struct {
	inner *appintel.DNSIntelService
}

func (a intelServiceAdapter) Run(
	ctx context.Context,
	domains []runtime.IntelDomain,
) ([]runtime.IntelRecord, error) {
	if a.inner == nil {
		return nil, nil
	}

	in := make([]appintel.Domain, 0, len(domains))
	for _, d := range domains {
		in = append(in, appintel.Domain{Name: d.Name})
	}

	records, err := a.inner.Run(ctx, in)
	if err != nil {
		return nil, err
	}

	out := make([]runtime.IntelRecord, 0, len(records))
	for _, rec := range records {
		out = append(out, runtime.IntelRecord{
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
	return out, nil
}
