package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Package struct {
	ImportPath string   `json:"ImportPath"`
	Dir        string   `json:"Dir"`
	Imports    []string `json:"Imports"`
}

const (
	penaltyCross      = 5
	penaltyHorizontal = 3
	penaltyRule3      = 2
	maxPenaltyCap     = 30.0
)

func runGoList() ([]Package, error) {
	cmd := exec.Command("go", "list", "-json", "./...")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	dec := json.NewDecoder(bytes.NewReader(out))
	var pkgs []Package

	for {
		var p Package
		if err := dec.Decode(&p); err != nil {
			break
		}
		pkgs = append(pkgs, p)
	}
	return pkgs, nil
}

// Normalize to repo-relative path
func normalizePath(dir string) string {
	cwd, _ := os.Getwd()
	rel, err := filepath.Rel(cwd, dir)
	if err != nil {
		return dir
	}
	return filepath.ToSlash(rel)
}

// Layer = first folder
func getLayer(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

// Module = second folder (if exists)
func getModule(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

// Parent = same directory
func getParent(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) <= 1 {
		return path
	}
	return strings.Join(parts[:len(parts)-1], "/")
}

func main() {
	pkgs, err := runGoList()
	if err != nil {
		fmt.Println("❌ go list failed:", err)
		os.Exit(0)
	}

	graph := make(map[string][]string)
	pathMap := make(map[string]string)
	usage := make(map[string]int)

	for _, p := range pkgs {
		rel := normalizePath(p.Dir)
		graph[p.ImportPath] = p.Imports
		pathMap[p.ImportPath] = rel

		for _, imp := range p.Imports {
			usage[imp]++
		}
	}

	var violations []string
	score := 0

	for pkg, imports := range graph {
		pathA := pathMap[pkg]
		layerA := getLayer(pathA)
		moduleA := getModule(pathA)
		parentA := getParent(pathA)

		for _, imp := range imports {
			pathB, ok := pathMap[imp]
			if !ok {
				continue // skip stdlib / external
			}

			layerB := getLayer(pathB)
			moduleB := getModule(pathB)
			parentB := getParent(pathB)

			// -----------------------------
			// RULE 1: Cross-module (same layer, different module)
			// -----------------------------
			if layerA == layerB && moduleA != "" && moduleB != "" && moduleA != moduleB {
				violations = append(violations,
					fmt.Sprintf("[CROSS] %s → %s", pathA, pathB))
				score += penaltyCross
			}

			// -----------------------------
			// RULE 2: Horizontal (same parent)
			// -----------------------------
			if parentA == parentB && pathA != pathB {
				violations = append(violations,
					fmt.Sprintf("[HORIZONTAL] %s → %s", pathA, pathB))
				score += penaltyHorizontal

				// -------------------------
				// RULE 3: Exclusive usage
				// -------------------------
				if usage[imp] > 1 {
					violations = append(violations,
						fmt.Sprintf("[RULE3] %s used by %d packages", pathB, usage[imp]))
					score += penaltyRule3
				}
			}
		}
	}

	// -----------------------------
	// OUTPUT
	// -----------------------------
	fmt.Println("\n🏛️  Architectural Analysis Report")
	fmt.Println("────────────────────────────────────────")

	if len(violations) == 0 {
		fmt.Println("✅ No architectural violations detected.")
	} else {
		for _, v := range violations {
			fmt.Println("❌", v)
		}
	}

	fmt.Println("────────────────────────────────────────")
	fmt.Printf("📉 Violation Score: %d\n", score)

	penalty := float64(score) * 0.5
	if penalty > maxPenaltyCap {
		penalty = maxPenaltyCap
	}

	fmt.Printf("📊 Suggested Coverage Penalty: -%.1f%%\n", penalty)

	// Export for CI
	f, err := os.Create("arch_score.txt")
	if err == nil {
		defer f.Close()
		fmt.Fprintf(f, "%.2f", penalty)
	}

	os.Exit(0)
}
