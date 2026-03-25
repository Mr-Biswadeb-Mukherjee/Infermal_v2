// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type runtimePaths struct {
	repo            string
	engine          string
	logsDir         string
	keywordsCSV     string
	settingConf     string
	redisConf       string
	dnsIntelOutput  string
	generatedOutput string
	resolvedOutput  string
}

var (
	pathsOnce sync.Once
	pathsSet  runtimePaths
)

func loadRuntimePaths() runtimePaths {
	pathsOnce.Do(func() {
		repo := detectRuntimeRoot()
		engineDir := filepath.Join(repo, "Engine")
		pathsSet = runtimePaths{
			repo:            repo,
			engine:          engineDir,
			logsDir:         filepath.Join(repo, "Logs"),
			keywordsCSV:     filepath.Join(engineDir, "Input", "Keywords.csv"),
			settingConf:     filepath.Join(repo, "Setting", "setting.conf"),
			redisConf:       filepath.Join(repo, "Setting", "redis.yaml"),
			dnsIntelOutput:  filepath.Join(repo, "Output", "DNS_Intel.ndjson"),
			generatedOutput: filepath.Join(repo, "Output", "Generated_Domain.ndjson"),
			resolvedOutput:  filepath.Join(repo, "Output", "Resolved_Domain.ndjson"),
		}
	})
	return pathsSet
}

func detectRuntimeRoot() string {
	if dir := executableDir(); dir != "" {
		if !isTemporaryBuildDir(dir) {
			return dir
		}
	}
	if root := moduleRootFromWorkingDir(); root != "" {
		return root
	}
	if dir := executableDir(); dir != "" {
		return dir
	}
	return "."
}

func moduleRootFromWorkingDir() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}
	return findGoModuleRoot(wd)
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

func findGoModuleRoot(start string) string {
	dir := start
	for dir != "" {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
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
