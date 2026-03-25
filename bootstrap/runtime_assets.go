// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package bootstrap

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	defaultDirMode = 0o750
)

//go:embed assets/Engine/Input/Keywords.csv
var embeddedRuntimeAssets embed.FS

type embeddedAsset struct {
	source string
	mode   fs.FileMode
	target func(runtimePaths) string
}

var defaultAssets = []embeddedAsset{
	{
		source: "assets/Engine/Input/Keywords.csv",
		mode:   0o600,
		target: func(paths runtimePaths) string { return paths.keywordsCSV },
	},
}

func ensureRuntimeAssets(paths runtimePaths) error {
	for _, asset := range defaultAssets {
		if err := ensureEmbeddedAsset(paths, asset); err != nil {
			return err
		}
	}
	return nil
}

func ensureEmbeddedAsset(paths runtimePaths, asset embeddedAsset) error {
	targetPath := filepath.Clean(asset.target(paths))
	if targetPath == "." || targetPath == "" {
		return fmt.Errorf("invalid runtime target path for asset: %s", asset.source)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), defaultDirMode); err != nil {
		return fmt.Errorf("create runtime asset directory: %w", err)
	}
	if exists, err := fileExists(targetPath); err != nil {
		return err
	} else if exists {
		return nil
	}
	content, err := embeddedRuntimeAssets.ReadFile(asset.source)
	if err != nil {
		return fmt.Errorf("read embedded asset %s: %w", asset.source, err)
	}
	if err := os.WriteFile(targetPath, content, asset.mode); err != nil {
		return fmt.Errorf("write runtime asset %s: %w", targetPath, err)
	}
	return nil
}

func fileExists(path string) (bool, error) {
	info, err := os.Stat(path)
	switch {
	case err == nil:
		if info.IsDir() {
			return false, fmt.Errorf("runtime asset path is directory: %s", path)
		}
		return true, nil
	case os.IsNotExist(err):
		return false, nil
	default:
		return false, fmt.Errorf("stat runtime asset path %s: %w", path, err)
	}
}
