// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package engine

import (
	"context"
	"fmt"

	app "github.com/Mr-Biswadeb-Mukherjee/Infermal_v2/Engine/app"
	runtime "github.com/Mr-Biswadeb-Mukherjee/Infermal_v2/Engine/runtime"
)

var runRuntime = runtime.Run
var runApp = func(ctx context.Context, deps Dependencies) error {
	modules := newModuleFactory(
		func(cfg app.Config, dnsLog app.ModuleLogger) interface {
			Resolve(ctx context.Context, domain string) (bool, error)
		} {
			return app.NewResolver(cfg, dnsLog)
		},
		app.GenerateDomains,
		app.NewDNSIntelService,
	)
	return runRuntime(ctx, buildRuntimeDependencies(deps, modules))
}

var printLine = func(args ...any) {
	fmt.Println(args...)
}

func Run(deps Dependencies) {
	ctx := context.Background()

	if err := runApp(ctx, deps); err != nil {
		printLine("Error:", err.Error())
	}

	printLine("Shutdown complete.")
}
