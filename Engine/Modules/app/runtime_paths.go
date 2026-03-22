// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package app

import (
	"os"
	"path/filepath"
	"runtime"
	"sync"
)

type runtimeRoots struct {
	engine string
	repo   string
}

var (
	rootsOnce sync.Once
	rootsSet  runtimeRoots
)

func engineRoot() string {
	rootsOnce.Do(loadRuntimeRoots)
	return rootsSet.engine
}

func repoRoot() string {
	rootsOnce.Do(loadRuntimeRoots)
	return rootsSet.repo
}

func settingFile(name string) string {
	return filepath.Join(repoRoot(), "Setting", name)
}

func inputFile(name string) string {
	return filepath.Join(engineRoot(), "Input", name)
}

func outputDir() string {
	return filepath.Join(repoRoot(), "Output")
}

func outputFile(name string) string {
	return filepath.Join(outputDir(), name)
}

func logsDir() string {
	return filepath.Join(repoRoot(), "Logs")
}

func loadRuntimeRoots() {
	repo := detectRepoRoot()
	rootsSet = runtimeRoots{
		engine: detectEngineDir(repo),
		repo:   repo,
	}
}

func detectRepoRoot() string {
	if root := findGoModuleRoot(callerDir()); root != "" {
		return root
	}

	wd, err := os.Getwd()
	if err == nil {
		if root := findGoModuleRoot(wd); root != "" {
			return root
		}
	}
	return "."
}

func detectEngineDir(repo string) string {
	engine := filepath.Join(repo, "Engine")
	if isDir(engine) {
		return engine
	}
	return repo
}

func callerDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return filepath.Dir(file)
}

func findGoModuleRoot(start string) string {
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

func isDir(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}
