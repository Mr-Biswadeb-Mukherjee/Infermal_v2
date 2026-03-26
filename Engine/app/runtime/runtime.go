// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package runtime

import (
	"context"
	"fmt"
)

func Run(parentCtx context.Context, deps Dependencies) error {
	rt, err := newAppRuntime(deps)
	if err != nil {
		return err
	}
	defer rt.Close()

	total, spool, err := loadGeneratedDomains(
		parentCtx,
		rt.paths.GeneratedOutput,
		rt.paths.KeywordsCSV,
		rt.modules,
		rt.cache,
	)
	if err != nil {
		rt.logs.app.Alert("domain generation failed: %v", err)
		return fmt.Errorf("error processing Keywords.csv: %w", err)
	}
	rt.generated = spool

	if total == 0 {
		rt.startup.Stop()
		if rt.printLine != nil {
			rt.printLine("no domains generated")
		}
		return nil
	}

	runner := newScanRunner(rt, total)
	modules, err := rt.newModules(parentCtx, runner.onIntelDone())
	if err != nil {
		rt.logs.app.Alert("intel pipeline init failed: %v", err)
		return fmt.Errorf("error starting dns intel pipeline: %w", err)
	}

	resolved := runner.run(parentCtx, modules)
	rt.finishRun(total, resolved)
	rt.logs.app.Info("run completed generated=%d resolved=%d", total, resolved)
	return nil
}
