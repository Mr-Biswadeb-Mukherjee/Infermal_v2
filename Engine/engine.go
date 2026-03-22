// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package engine

import (
	"context"

	app "github.com/Mr-Biswadeb-Mukherjee/Infermal_v2/Engine/Modules/app"
)

func Run() {
	ctx := context.Background()

	if err := app.Run(ctx); err != nil {
		println("Error:", err.Error())
	}

	println("Shutdown complete.")
}
