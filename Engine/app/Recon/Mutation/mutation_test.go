// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package mutation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadModuleConfigCreatesSecureConfig(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Setting", "setting.conf")

	cfg, err := loadModuleConfig(path)
	if err != nil {
		t.Fatalf("loadModuleConfig error: %v", err)
	}
	if !cfg.Enabled || !cfg.Character || !cfg.Seed || !cfg.Hashchain {
		t.Fatalf("unexpected defaults: %+v", cfg)
	}
	assertNoSharedFileAccess(t, path)
	assertNoSharedDirAccess(t, filepath.Dir(path))
}

func TestLoadModuleConfigRejectsDirectoryPath(t *testing.T) {
	_, err := loadModuleConfig(t.TempDir())
	if err == nil {
		t.Fatal("expected directory path to be rejected")
	}
}

func TestLoadKeywordsRejectsNonCSVPath(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "keywords.txt")

	if err := os.WriteFile(path, []byte("domain\nexample\n"), 0o600); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := loadKeywords(path); err == nil {
		t.Fatal("expected non-csv input to be rejected")
	}
}

func TestLoadKeywordsReadsCSV(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "keywords.csv")
	content := "domain\nExample\napi-test\n"

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write csv: %v", err)
	}

	keywords, err := loadKeywords(path)
	if err != nil {
		t.Fatalf("loadKeywords error: %v", err)
	}
	if len(keywords) != 2 {
		t.Fatalf("expected 2 keywords, got %d", len(keywords))
	}
}

func assertNoSharedFileAccess(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat file: %v", err)
	}
	if info.Mode().Perm()&0o077 != 0 {
		t.Fatalf("expected private file permissions, got %o", info.Mode().Perm())
	}
}

func assertNoSharedDirAccess(t *testing.T, path string) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if info.Mode().Perm()&0o027 != 0 {
		t.Fatalf("expected directory permissions at or below 750, got %o", info.Mode().Perm())
	}
}
