// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package recon_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testRepoRootEnv = "DIBs_REPO_ROOT"

func TestMain(m *testing.M) {
	if err := setTestRepoRootEnv(); err != nil {
		fmt.Fprintln(os.Stderr, "failed to configure repo root for tests:", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func setTestRepoRootEnv() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	root, err := findRepoRootForTests(wd)
	if err != nil {
		return err
	}
	return os.Setenv(testRepoRootEnv, root)
}

func findRepoRootForTests(start string) (string, error) {
	dir := filepath.Clean(strings.TrimSpace(start))
	for dir != "" {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return "", fmt.Errorf("go.mod not found from %s", start)
}
