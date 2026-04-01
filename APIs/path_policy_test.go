// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package apis

import (
	"path/filepath"
	"testing"
)

func TestNormalizeSettingFilePathDefaultUsesBinaryDirectory(t *testing.T) {
	root, err := runtimeRootDir()
	if err != nil {
		t.Fatalf("runtime root dir: %v", err)
	}
	got, err := normalizeSettingFilePath("")
	if err != nil {
		t.Fatalf("normalize setting file path: %v", err)
	}
	want := filepath.Join(root, settingDirName, "api_private.key")
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestNormalizeSettingFilePathRejectsOutsideBinarySettingDir(t *testing.T) {
	_, err := normalizeSettingFilePath("/tmp/Setting/api_private.key")
	if err == nil {
		t.Fatal("expected outside Setting path to be rejected")
	}
}
