// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package bootstrap

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureRuntimeAssetsCreatesMissingFiles(t *testing.T) {
	root := t.TempDir()
	paths := testRuntimePaths(root)

	if err := ensureRuntimeAssets(paths); err != nil {
		t.Fatalf("ensureRuntimeAssets error: %v", err)
	}

	assertContains(t, paths.keywordsCSV, "Amazon")
}

func TestEnsureRuntimeAssetsPreservesExistingFiles(t *testing.T) {
	root := t.TempDir()
	paths := testRuntimePaths(root)

	custom := "my-keyword\n"
	if err := os.MkdirAll(filepath.Dir(paths.keywordsCSV), 0o750); err != nil {
		t.Fatalf("mkdir keywords dir: %v", err)
	}
	if err := os.WriteFile(paths.keywordsCSV, []byte(custom), 0o600); err != nil {
		t.Fatalf("write custom keywords: %v", err)
	}

	if err := ensureRuntimeAssets(paths); err != nil {
		t.Fatalf("ensureRuntimeAssets error: %v", err)
	}

	data, err := os.ReadFile(paths.keywordsCSV)
	if err != nil {
		t.Fatalf("read Keywords.csv: %v", err)
	}
	if string(data) != custom {
		t.Fatalf("existing Keywords.csv must not be overwritten")
	}
}

func testRuntimePaths(root string) runtimePaths {
	engine := filepath.Join(root, "Engine")
	setting := filepath.Join(root, "Setting")
	output := filepath.Join(root, "Output")
	return runtimePaths{
		repo:            root,
		engine:          engine,
		logsDir:         filepath.Join(root, "Logs"),
		keywordsCSV:     filepath.Join(engine, "Input", "Keywords.csv"),
		settingConf:     filepath.Join(setting, "setting.conf"),
		redisConf:       filepath.Join(setting, "redis.yaml"),
		dnsIntelOutput:  filepath.Join(output, "DNS_Intel.ndjson"),
		generatedOutput: filepath.Join(output, "Generated_Domain.ndjson"),
		resolvedOutput:  filepath.Join(output, "Resolved_Domain.ndjson"),
	}
}

func assertContains(t *testing.T, path, needle string) {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	content := string(data)
	if !strings.Contains(content, needle) {
		t.Fatalf("expected %s to include %q", path, needle)
	}
}
