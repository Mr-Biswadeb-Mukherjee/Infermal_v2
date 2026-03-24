// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package combosquat

import (
	"testing"
)

// -------------------------------------------------------------
// Levenshtein Distance Tests
// -------------------------------------------------------------

func TestLevenshteinBasic(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"kitten", "sitting", 3},
		{"flaw", "lawn", 2},
		{"", "", 0},
		{"a", "", 1},
		{"", "abc", 3},
		{"same", "same", 0},
	}

	for _, tt := range tests {
		got := levenshteinInternal(tt.a, tt.b)
		if got != tt.want {
			t.Fatalf("levenshteinInternal(%q, %q) = %d; want %d",
				tt.a, tt.b, got, tt.want)
		}
	}
}

func TestLevenshteinUnicode(t *testing.T) {
	a := "héllo"
	b := "hello"

	got := levenshteinInternal(a, b)
	if got != 1 {
		t.Fatalf("unicode distance mismatch: got %d, want 1", got)
	}
}

func TestLevenshteinCaseInsensitive(t *testing.T) {
	if levenshteinInternal("Test", "test") != 0 {
		t.Fatal("expected case-insensitive equality")
	}
}

// -------------------------------------------------------------
// generateEdits Tests
// -------------------------------------------------------------

func TestGenerateEditsNonEmpty(t *testing.T) {
	out := generateEdits("test")
	if len(out) == 0 {
		t.Fatal("expected edits, got empty slice")
	}
}

func TestGenerateEditsContainsDeletion(t *testing.T) {
	out := generateEdits("test")

	found := false
	for _, v := range out {
		if v == "est" {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("expected deletion variant 'est' not found")
	}
}

func TestGenerateEditsContainsInsertion(t *testing.T) {
	out := generateEdits("abc")

	// Deterministic expected insertions based on implementation
	expected := map[string]bool{
		"aabc": true, // insert at index 0
		"abbc": true, // insert at index 1
		"abcc": true, // insert at index 2
	}

	found := 0
	for _, v := range out {
		if expected[v] {
			found++
		}
	}

	if found == 0 {
		t.Fatalf("expected at least one valid insertion variant, got none")
	}
}

func TestGenerateEditsContainsSubstitution(t *testing.T) {
	out := generateEdits("abc")

	found := false
	for _, v := range out {
		if v == "bbc" || v == "acc" {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("expected substitution variant not found")
	}
}

func TestGenerateEditsContainsTransposition(t *testing.T) {
	out := generateEdits("ab")

	found := false
	for _, v := range out {
		if v == "ba" {
			found = true
			break
		}
	}

	if !found {
		t.Fatal("expected transposition 'ba' not found")
	}
}

// -------------------------------------------------------------
// Combosquat Tests (Public API)
// -------------------------------------------------------------

func TestCombosquatBasic(t *testing.T) {
	out := Combosquat("test")

	if len(out) == 0 {
		t.Fatal("expected output, got empty")
	}
}

func TestCombosquatNoSelf(t *testing.T) {
	target := "test"
	out := Combosquat(target)

	for _, v := range out {
		if v == target {
			t.Fatal("output must not contain original string")
		}
	}
}

func TestCombosquatUniqueness(t *testing.T) {
	out := Combosquat("test")

	seen := map[string]bool{}
	for _, v := range out {
		if seen[v] {
			t.Fatalf("duplicate found: %s", v)
		}
		seen[v] = true
	}
}

func TestCombosquatDistanceConstraint(t *testing.T) {
	target := "test"
	out := Combosquat(target)

	for _, v := range out {
		dist := levenshteinInternal(target, v)
		if dist > 3 {
			t.Fatalf("distance violation: %s (dist=%d)", v, dist)
		}
	}
}

func TestCombosquatEmptyInput(t *testing.T) {
	out := Combosquat("")

	if out == nil {
		t.Fatal("expected non-nil slice")
	}
}

func TestCombosquatUnicode(t *testing.T) {
	out := Combosquat("héllo")

	if len(out) == 0 {
		t.Fatal("expected unicode variants")
	}
}

func TestCombosquatDeterministic(t *testing.T) {
	a := Combosquat("test")
	b := Combosquat("test")

	if len(a) != len(b) {
		t.Fatalf("non-deterministic output size: %d vs %d", len(a), len(b))
	}
}
