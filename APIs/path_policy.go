// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package apis

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	outputDirName  = "Output"
	settingDirName = "Setting"
)

func normalizeOutputDir(path string) (string, error) {
	return normalizeNamedDir(path, outputDirName, defaultDetailsOutputDir)
}

func normalizeSettingFilePath(path string) (string, error) {
	rootDir, err := runtimeRootDir()
	if err != nil {
		return "", err
	}
	clean := strings.TrimSpace(path)
	if clean == "" {
		clean = defaultPrivateKeyPath
	}
	if !filepath.IsAbs(clean) {
		clean = filepath.Join(rootDir, clean)
	}
	absPath := filepath.Clean(clean)
	settingDir := filepath.Join(rootDir, settingDirName)
	parent := filepath.Base(filepath.Dir(absPath))
	if !strings.EqualFold(parent, settingDirName) {
		return "", fmt.Errorf("private key path must be inside %s directory next to binary", settingDirName)
	}
	if strings.EqualFold(filepath.Base(absPath), settingDirName) {
		return "", errors.New("private key path must be a file")
	}
	if filepath.Dir(absPath) != settingDir {
		return "", fmt.Errorf("private key path must resolve under %s", settingDir)
	}
	return absPath, nil
}

func normalizeNamedDir(path, expectedBase, fallback string) (string, error) {
	clean := strings.TrimSpace(path)
	if clean == "" {
		clean = fallback
	}
	absPath, err := filepath.Abs(clean)
	if err != nil {
		return "", err
	}
	base := filepath.Base(absPath)
	if !strings.EqualFold(base, expectedBase) {
		return "", fmt.Errorf("path must target %s directory", expectedBase)
	}
	return absPath, nil
}

func ensurePathWithinRoot(rootPath, targetPath string) error {
	rootAbs, err := filepath.Abs(strings.TrimSpace(rootPath))
	if err != nil {
		return err
	}
	targetAbs, err := filepath.Abs(strings.TrimSpace(targetPath))
	if err != nil {
		return err
	}
	if targetAbs == rootAbs {
		return nil
	}
	prefix := rootAbs + string(os.PathSeparator)
	if strings.HasPrefix(targetAbs, prefix) {
		return nil
	}
	return errors.New("file must be inside output directory")
}

func runtimeRootDir() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Dir(filepath.Clean(exePath)), nil
}
