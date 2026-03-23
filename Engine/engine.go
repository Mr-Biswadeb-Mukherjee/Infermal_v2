// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package engine

import (
	"context"
	"fmt"
	"time"

	app "github.com/Mr-Biswadeb-Mukherjee/Infermal_v2/Engine/app"
	runtime "github.com/Mr-Biswadeb-Mukherjee/Infermal_v2/Engine/runtime"
)

var runApp = func(ctx context.Context, deps Dependencies) error {
	return runtime.Run(normalizeContext(ctx), buildRuntimeDependencies(deps))
}

var printLine = func(args ...any) {
	fmt.Println(args...)
}

func Run(deps Dependencies) {
	if err := runApp(context.Background(), deps); err != nil {
		printLine("Error:", err.Error())
	}

	printLine("Shutdown complete.")
}

func normalizeContext(ctx context.Context) context.Context {
	if ctx != nil {
		return ctx
	}
	return context.Background()
}

func buildRuntimeDependencies(deps Dependencies) runtime.Dependencies {
	return runtime.Dependencies{
		Config:      toRuntimeConfig(deps.Config),
		Paths:       toRuntimePaths(deps.Paths),
		Startup:     deps.Startup,
		Logs:        toRuntimeLogs(deps.Logs),
		Cache:       deps.Cache,
		Limiter:     deps.Limiter,
		InitLimiter: initRuntimeLimiter(deps),
		WorkerPools: workerPoolFactoryAdapter{inner: deps.WorkerPools},
		Cooldowns:   cooldownFactoryAdapter{inner: deps.Cooldowns},
		Adaptive:    adaptiveFactoryAdapter{inner: deps.Adaptive},
		Writers:     writerFactoryAdapter{inner: deps.Writers},
		Modules:     newRuntimeModuleFactory(),
		PrintLine:   printLine,
	}
}

func toRuntimeConfig(cfg Config) runtime.Config {
	return runtime.Config{
		RateLimit:      cfg.RateLimit,
		TimeoutSeconds: cfg.TimeoutSeconds,
		MaxRetries:     cfg.MaxRetries,
		AutoScale:      cfg.AutoScale,
		UpstreamDNS:    cfg.UpstreamDNS,
		BackupDNS:      cfg.BackupDNS,
		DNSRetries:     cfg.DNSRetries,
		DNSTimeoutMS:   cfg.DNSTimeoutMS,
	}
}

func toRuntimePaths(paths Paths) runtime.Paths {
	return runtime.Paths{
		KeywordsCSV:     paths.KeywordsCSV,
		DNSIntelOutput:  paths.DNSIntelOutput,
		GeneratedOutput: paths.GeneratedOutput,
	}
}

func toRuntimeLogs(logs LogSet) runtime.LogSet {
	return runtime.LogSet{
		App:         logs.App,
		DNS:         logs.DNS,
		RateLimiter: logs.RateLimiter,
	}
}

func initRuntimeLimiter(deps Dependencies) runtime.LimiterInitFunc {
	return func(_ runtime.CacheStore, window time.Duration, maxHits int64, logger runtime.ModuleLogger) {
		deps.Limiter.Init(deps.Cache, window, maxHits, toAppLogger(logger, deps.Logs.RateLimiter))
	}
}

func newRuntimeModuleFactory() runtime.ModuleFactory {
	return newModuleFactory(
		func(cfg app.Config, dnsLog app.ModuleLogger) interface {
			Resolve(ctx context.Context, domain string) (bool, error)
		} {
			return app.NewResolver(cfg, dnsLog)
		},
		app.GenerateDomains,
		app.NewDNSIntelService,
	)
}
