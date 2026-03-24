// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package soundsquat

import (
	"strings"
	"testing"
)

// -------------------------------------------------------------
// Helper for equality
// -------------------------------------------------------------
func assertEqual(t *testing.T, got, want string) {
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

// -------------------------------------------------------------
// Basic correctness tests (Soundex standard cases)
// -------------------------------------------------------------
func TestSoundexBasic(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Robert", "R163"},
		{"Rupert", "R163"},
		{"Rubin", "R150"},
		{"Ashcraft", "A261"},
		{"Tymczak", "T522"},
		{"Pfister", "P236"},
	}

	for _, tt := range tests {
		got := soundexInternal(tt.input)
		assertEqual(t, got, tt.want)
	}
}

// -------------------------------------------------------------
// Case insensitivity & trimming
// -------------------------------------------------------------
func TestNormalization(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"  robert  ", "R163"},
		{"RoBeRt", "R163"},
		{"\nrupert\t", "R163"},
	}

	for _, tt := range tests {
		got := soundexInternal(tt.input)
		assertEqual(t, got, tt.want)
	}
}

// -------------------------------------------------------------
// Non-alphabetic characters handling
// -------------------------------------------------------------
func TestNonAlphabeticCharacters(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"R!o@b#e$r%t", "R163"},
		{"123Robert456", "R163"},
		{"--Ashcraft--", "A261"},
	}

	for _, tt := range tests {
		got := soundexInternal(tt.input)
		assertEqual(t, got, tt.want)
	}
}

// -------------------------------------------------------------
// Short inputs & padding behavior
// -------------------------------------------------------------
func TestShortInputs(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"A", "A000"},
		{"B", "B000"},
		{"AB", "A100"},
		{"Ab", "A100"},
	}

	for _, tt := range tests {
		got := soundexInternal(tt.input)
		assertEqual(t, got, tt.want)
	}
}

// -------------------------------------------------------------
// Empty and whitespace inputs
// -------------------------------------------------------------
func TestEmptyInput(t *testing.T) {
	tests := []string{
		"",
		"   ",
		"\n\t",
	}

	for _, input := range tests {
		got := soundexInternal(input)
		if got != "" {
			t.Fatalf("expected empty string, got %q for input %q", got, input)
		}
	}
}

// -------------------------------------------------------------
// Duplicate code suppression
// -------------------------------------------------------------
func TestDuplicateCodeSuppression(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Bbbbbb", "B100"},
		{"Pfister", "P236"}, // classic tricky case
		{"Mississippi", "M210"},
	}

	for _, tt := range tests {
		got := soundexInternal(tt.input)
		assertEqual(t, got, tt.want)
	}
}

// -------------------------------------------------------------
// Unicode safety (non-ASCII letters should not panic)
// -------------------------------------------------------------
func TestUnicodeInput(t *testing.T) {
	tests := []string{
		"Ångström",
		"Éclair",
		"नमस्ते",
		"東京",
	}

	for _, input := range tests {
		_ = soundexInternal(input) // ensure no panic
	}
}

// -------------------------------------------------------------
// Stability / determinism test
// -------------------------------------------------------------
func TestDeterministicOutput(t *testing.T) {
	input := "Robert"

	first := soundexInternal(input)
	for i := 0; i < 1000; i++ {
		if soundexInternal(input) != first {
			t.Fatalf("non-deterministic output detected")
		}
	}
}

// -------------------------------------------------------------
// Public API test (contract validation)
// -------------------------------------------------------------
func TestSoundsquatAPI(t *testing.T) {
	result := Soundsquat("Robert")

	if len(result) != 1 {
		t.Fatalf("expected slice length 1, got %d", len(result))
	}

	if result[0] != "R163" {
		t.Fatalf("expected R163, got %s", result[0])
	}
}

// -------------------------------------------------------------
// Fuzz test (Go 1.18+)
// -------------------------------------------------------------
func FuzzSoundexInternal(f *testing.F) {
	seedInputs := []string{
		"Robert",
		"Ashcraft",
		"",
		"123",
		"नमस्ते",
	}

	for _, s := range seedInputs {
		f.Add(s)
	}

	f.Fuzz(func(t *testing.T, input string) {
		output := soundexInternal(input)

		// Property: length is either 0 or exactly 4
		if input == "" || len(strings.TrimSpace(input)) == 0 {
			if output != "" {
				t.Fatalf("expected empty output, got %q", output)
			}
			return
		}

		if len(output) != 4 {
			t.Fatalf("expected length 4, got %d (%q)", len(output), output)
		}

		// First character must match normalized first letter (if valid)
		if len(output) > 0 && len(input) > 0 {
			// Not strictly enforceable for all unicode, but sanity check
			if output[0] == '0' {
				t.Fatalf("invalid first character in output: %q", output)
			}
		}
	})
}
