// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 Biswadeb Mukherjee

package bitsquatting

import (
	"reflect"
	"sort"
	"testing"
)

// helper to check uniqueness
func isUnique(slice []string) bool {
	seen := make(map[string]struct{})
	for _, v := range slice {
		if _, ok := seen[v]; ok {
			return false
		}
		seen[v] = struct{}{}
	}
	return true
}

func TestIsSafeDomainChar(t *testing.T) {
	tests := []struct {
		input byte
		want  bool
	}{
		{'a', true},
		{'Z', true},
		{'0', true},
		{'9', true},
		{'-', true},
		{'_', false},
		{'@', false},
		{' ', false},
		{0x00, false},
	}

	for _, tt := range tests {
		got := isSafeDomainChar(tt.input)
		if got != tt.want {
			t.Errorf("isSafeDomainChar(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

func TestGenerateInternal_Basic(t *testing.T) {
	base := "a"
	results := generateInternal(base)

	if len(results) == 0 {
		t.Fatalf("expected mutations for %q, got none", base)
	}

	for _, r := range results {
		if len(r) != len(base) {
			t.Errorf("invalid length mutation: %q", r)
		}
		if r == base {
			t.Errorf("base string should not appear in results")
		}
	}
}

func TestGenerateInternal_Deterministic(t *testing.T) {
	base := "test"

	r1 := generateInternal(base)
	r2 := generateInternal(base)

	if !reflect.DeepEqual(r1, r2) {
		t.Errorf("non-deterministic output:\n%v\n%v", r1, r2)
	}
}

func TestGenerateInternal_Sorted(t *testing.T) {
	base := "abc"
	results := generateInternal(base)

	sorted := make([]string, len(results))
	copy(sorted, results)
	sort.Strings(sorted)

	if !reflect.DeepEqual(results, sorted) {
		t.Errorf("results are not sorted")
	}
}

func TestGenerateInternal_Unique(t *testing.T) {
	base := "hello"
	results := generateInternal(base)

	if !isUnique(results) {
		t.Errorf("results contain duplicates")
	}
}

func TestGenerateInternal_ValidCharset(t *testing.T) {
	base := "abc123"
	results := generateInternal(base)

	for _, r := range results {
		for i := 0; i < len(r); i++ {
			if !isSafeDomainChar(r[i]) {
				t.Errorf("invalid char %q in result %q", r[i], r)
			}
		}
	}
}

func TestGenerateInternal_EmptyInput(t *testing.T) {
	base := ""
	results := generateInternal(base)

	if len(results) != 0 {
		t.Errorf("expected empty result for empty input, got %v", results)
	}
}

func TestBitsquatting_PublicAPI(t *testing.T) {
	base := "abc"

	internal := generateInternal(base)
	public := Bitsquatting(base)

	if !reflect.DeepEqual(internal, public) {
		t.Errorf("public API mismatch with internal function")
	}
}
