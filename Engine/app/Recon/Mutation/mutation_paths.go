// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package mutation

import (
	"os"
	"path/filepath"
	"strings"
)

const defaultSettingsPath = "Setting/setting.conf"

func resolveSettingsPath() string {
	if dir := executableDir(); dir != "" {
		if !isTemporaryBuildDir(dir) {
			return filepath.Join(dir, defaultSettingsPath)
		}
	}
	if root := findModuleRoot(workingDirectory()); root != "" {
		return filepath.Join(root, defaultSettingsPath)
	}
	if dir := executableDir(); dir != "" {
		return filepath.Join(dir, defaultSettingsPath)
	}
	return defaultSettingsPath
}

func workingDirectory() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return wd
}

func executableDir() string {
	exePath, err := os.Executable()
	if err != nil {
		return ""
	}
	return filepath.Dir(filepath.Clean(exePath))
}

func isTemporaryBuildDir(dir string) bool {
	tmp := filepath.Clean(os.TempDir())
	clean := filepath.Clean(dir)
	return strings.Contains(clean, filepath.Join(tmp, "go-build"))
}

func findModuleRoot(start string) string {
	dir := start
	for dir != "" {
		if hasGoMod(dir) {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
	return ""
}

func hasGoMod(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil
}
