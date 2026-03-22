// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

//go:build ignore
// +build ignore

package rresolver

import (
	"os"
	"path/filepath"
	"runtime"
)

const defaultRootHintsPath = "Setting/root.conf"

func resolveRootHintsPath() string {
	if root := findResolverModuleRoot(callerResolverDir()); root != "" {
		return filepath.Join(root, defaultRootHintsPath)
	}

	wd, err := os.Getwd()
	if err == nil {
		if root := findResolverModuleRoot(wd); root != "" {
			return filepath.Join(root, defaultRootHintsPath)
		}
	}
	return defaultRootHintsPath
}

func callerResolverDir() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return filepath.Dir(file)
}

func findResolverModuleRoot(start string) string {
	dir := start
	for dir != "" {
		if hasResolverGoMod(dir) {
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

func hasResolverGoMod(dir string) bool {
	_, err := os.Stat(filepath.Join(dir, "go.mod"))
	return err == nil
}

func rootHintsPathFromDir(dir string) string {
	root := findResolverModuleRoot(dir)
	if root == "" {
		return defaultRootHintsPath
	}
	return filepath.Join(root, defaultRootHintsPath)
}

func rootHintsPathFromWD() string {
	wd, err := os.Getwd()
	if err != nil {
		return defaultRootHintsPath
	}
	return rootHintsPathFromDir(wd)
}

func rootHintsPathFromCaller() string {
	return rootHintsPathFromDir(callerResolverDir())
}

func rootHintsDir() string {
	return filepath.Dir(resolveRootHintsPath())
}

func rootHintsFileName() string {
	return filepath.Base(resolveRootHintsPath())
}
