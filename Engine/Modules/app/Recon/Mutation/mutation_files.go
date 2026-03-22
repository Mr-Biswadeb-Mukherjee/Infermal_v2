// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package mutation

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	mutationConfigDirMode  = 0o750
	mutationConfigFileMode = 0o600
)

func sanitizeSettingsPath(path string) (string, error) {
	return sanitizeMutationPath(path, "", "settings")
}

func sanitizeMutationCSVPath(path string) (string, error) {
	return sanitizeMutationPath(path, ".csv", "csv")
}

func sanitizeMutationPath(path, ext, label string) (string, error) {
	raw := strings.TrimSpace(path)
	if raw == "" {
		return "", fmt.Errorf("%s path is empty", label)
	}

	cleanPath := filepath.Clean(raw)
	if ext != "" && !strings.EqualFold(filepath.Ext(cleanPath), ext) {
		return "", fmt.Errorf("%s path must end with %s: %s", label, ext, cleanPath)
	}
	if err := rejectDirectoryPath(cleanPath, label); err != nil {
		return "", err
	}
	return cleanPath, nil
}

func rejectDirectoryPath(path, label string) error {
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	if info.IsDir() {
		return fmt.Errorf("%s path is a directory: %s", label, path)
	}
	return nil
}

func readSettingsContent(path string) ([]byte, error) {
	cleanPath, err := sanitizeSettingsPath(path)
	if err != nil {
		return nil, err
	}
	// #nosec G304 -- cleanPath is normalized and validated by sanitizeSettingsPath.
	return os.ReadFile(cleanPath)
}

func openSettingsFile(path string) (*os.File, error) {
	cleanPath, err := sanitizeSettingsPath(path)
	if err != nil {
		return nil, err
	}
	// #nosec G304 -- cleanPath is normalized and validated by sanitizeSettingsPath.
	return os.Open(cleanPath)
}

func openSettingsAppendFile(path string) (*os.File, error) {
	cleanPath, err := sanitizeSettingsPath(path)
	if err != nil {
		return nil, err
	}
	// #nosec G304 -- cleanPath is normalized and validated by sanitizeSettingsPath.
	return os.OpenFile(cleanPath, os.O_APPEND|os.O_WRONLY, mutationConfigFileMode)
}

func createSettingsFile(path string) (*os.File, error) {
	cleanPath, err := sanitizeSettingsPath(path)
	if err != nil {
		return nil, err
	}
	if err := ensureSettingsDir(cleanPath); err != nil {
		return nil, err
	}
	// #nosec G304 -- cleanPath is normalized and validated by sanitizeSettingsPath.
	return os.OpenFile(cleanPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mutationConfigFileMode)
}

func ensureSettingsDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), mutationConfigDirMode)
}

func openMutationCSVFile(path string) (*os.File, error) {
	cleanPath, err := sanitizeMutationCSVPath(path)
	if err != nil {
		return nil, err
	}
	// #nosec G304 -- cleanPath is normalized and validated by sanitizeMutationCSVPath.
	return os.Open(cleanPath)
}
