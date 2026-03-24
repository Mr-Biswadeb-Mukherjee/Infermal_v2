// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package jarowinkler

import (
	"math"
	"testing"
)

// tolerance for float comparisons
const epsilon = 1e-9

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

// -------------------------------------------------------------
// Basic Correctness Tests
// -------------------------------------------------------------

func TestJaroDistance_Basic(t *testing.T) {
	tests := []struct {
		name string
		s1   string
		s2   string
		want float64
	}{
		{"both empty", "", "", 1.0},
		{"one empty", "abc", "", 0.0},
		{"same string", "hello", "hello", 1.0},
		{"completely different", "abc", "xyz", 0.0},

		// known values (approx)
		{"martha vs marhta", "martha", "marhta", 0.9444444444},
		{"dwayne vs duane", "dwayne", "duane", 0.8222222222},
		{"dixon vs dicksonx", "dixon", "dicksonx", 0.7666666666},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JaroDistance(tt.s1, tt.s2)
			if !almostEqual(got, tt.want) {
				t.Fatalf("JaroDistance(%q, %q) = %f, want %f",
					tt.s1, tt.s2, got, tt.want)
			}
		})
	}
}

// -------------------------------------------------------------
// Jaro-Winkler Tests
// -------------------------------------------------------------

func TestJaroWinklerDistance_Basic(t *testing.T) {
	tests := []struct {
		name string
		s1   string
		s2   string
		want float64
	}{
		{"exact match", "test", "test", 1.0},
		{"low similarity no boost", "abc", "xyz", 0.0},

		// known JW values
		{"martha vs marhta", "martha", "marhta", 0.9611111111},
		{"dwayne vs duane", "dwayne", "duane", 0.84},
		{"dixon vs dicksonx", "dixon", "dicksonx", 0.8133333333},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := JaroWinklerDistance(tt.s1, tt.s2)
			if !almostEqual(got, tt.want) {
				t.Fatalf("JaroWinklerDistance(%q, %q) = %f, want %f",
					tt.s1, tt.s2, got, tt.want)
			}
		})
	}
}

// -------------------------------------------------------------
// Symmetry Property
// -------------------------------------------------------------

func TestSymmetry(t *testing.T) {
	pairs := []struct {
		s1, s2 string
	}{
		{"hello", "hlelo"},
		{"dwayne", "duane"},
		{"abc", "xyz"},
		{"kitten", "sitting"},
	}

	for _, p := range pairs {
		j1 := JaroDistance(p.s1, p.s2)
		j2 := JaroDistance(p.s2, p.s1)

		if !almostEqual(j1, j2) {
			t.Fatalf("Jaro not symmetric: %f vs %f", j1, j2)
		}

		jw1 := JaroWinklerDistance(p.s1, p.s2)
		jw2 := JaroWinklerDistance(p.s2, p.s1)

		if !almostEqual(jw1, jw2) {
			t.Fatalf("Jaro-Winkler not symmetric: %f vs %f", jw1, jw2)
		}
	}
}

// -------------------------------------------------------------
// Unicode / Rune Safety
// -------------------------------------------------------------

func TestUnicodeSupport(t *testing.T) {
	tests := []struct {
		s1 string
		s2 string
	}{
		{"héllo", "hello"},
		{"こんにちは", "こんいちは"},
		{"😀😃😄", "😀😄😃"},
	}

	for _, tt := range tests {
		j := JaroDistance(tt.s1, tt.s2)
		jw := JaroWinklerDistance(tt.s1, tt.s2)

		if j < 0 || j > 1 {
			t.Fatalf("invalid Jaro range: %f", j)
		}
		if jw < 0 || jw > 1 {
			t.Fatalf("invalid JW range: %f", jw)
		}
	}
}

// -------------------------------------------------------------
// Prefix Boost Behavior (Winkler-specific)
// -------------------------------------------------------------

func TestPrefixBoost(t *testing.T) {
	s1 := "prefix_match_test"
	s2 := "prefix_match_best"

	j := JaroDistance(s1, s2)
	jw := JaroWinklerDistance(s1, s2)

	if jw < j {
		t.Fatalf("expected JW >= Jaro, got JW=%f J=%f", jw, j)
	}
}

// -------------------------------------------------------------
// Boundary / Small Strings
// -------------------------------------------------------------

func TestSmallStrings(t *testing.T) {
	tests := []struct {
		s1, s2 string
	}{
		{"a", "a"},
		{"a", "b"},
		{"ab", "ba"},
		{"ab", "ab"},
	}

	for _, tt := range tests {
		j := JaroDistance(tt.s1, tt.s2)

		if j < 0 || j > 1 {
			t.Fatalf("invalid range for %q,%q: %f", tt.s1, tt.s2, j)
		}
	}
}

// -------------------------------------------------------------
// Fuzz-style Random Stability (deterministic set)
// -------------------------------------------------------------

func TestStability(t *testing.T) {
	inputs := []string{
		"",
		"a",
		"random",
		"longer_random_string_here",
		"1234567890",
		"!@#$%^&*()",
	}

	for _, s1 := range inputs {
		for _, s2 := range inputs {
			j := JaroDistance(s1, s2)
			jw := JaroWinklerDistance(s1, s2)

			if j < 0 || j > 1 {
				t.Fatalf("Jaro out of bounds: %f", j)
			}
			if jw < 0 || jw > 1 {
				t.Fatalf("JW out of bounds: %f", jw)
			}
		}
	}
}
