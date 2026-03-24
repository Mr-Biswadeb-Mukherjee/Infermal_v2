package homograph

import (
	"reflect"
	"sort"
	"testing"
)

// helper to sort slices for deterministic comparison
func sortStrings(in []string) []string {
	out := make([]string, len(in))
	copy(out, in)
	sort.Strings(out)
	return out
}

// ---------------------------------------------------
// CORE FUNCTIONAL TESTS
// ---------------------------------------------------

func TestHomograph_SingleReplacement(t *testing.T) {
	input := "a"
	expected := []string{"а"} // Cyrillic 'a'

	result := Homograph(input)

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestHomograph_MultipleReplacements(t *testing.T) {
	input := "as"

	result := Homograph(input)

	expected := []string{
		"аs", // a replaced
		"aѕ", // s replaced
	}

	if !reflect.DeepEqual(sortStrings(result), sortStrings(expected)) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

func TestHomograph_MultipleEquivalents(t *testing.T) {
	input := "i"

	result := Homograph(input)

	expected := []string{
		"і", // Ukrainian i
		"ı", // Turkish dotless i
	}

	if !reflect.DeepEqual(sortStrings(result), sortStrings(expected)) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

// ---------------------------------------------------
// EDGE CASES
// ---------------------------------------------------

func TestHomograph_NoSubstitution(t *testing.T) {
	input := "b" // no homoglyph defined

	result := Homograph(input)

	if len(result) != 0 {
		t.Fatalf("expected empty result, got %v", result)
	}
}

func TestHomograph_EmptyInput(t *testing.T) {
	input := ""

	result := Homograph(input)

	if len(result) != 0 {
		t.Fatalf("expected empty result, got %v", result)
	}
}

func TestHomograph_CaseInsensitivity(t *testing.T) {
	input := "A" // should behave like 'a'

	result := Homograph(input)

	expected := []string{"а"}

	if !reflect.DeepEqual(result, expected) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

// ---------------------------------------------------
// POSITIONAL VALIDATION
// ---------------------------------------------------

func TestHomograph_PositionSpecificReplacement(t *testing.T) {
	input := "cat"

	result := Homograph(input)

	expected := []string{
		"ϲat", // c replaced
		"cаt", // a replaced
	}

	if !reflect.DeepEqual(sortStrings(result), sortStrings(expected)) {
		t.Fatalf("expected %v, got %v", expected, result)
	}
}

// ---------------------------------------------------
// DUPLICATE SAFETY
// ---------------------------------------------------

func TestHomograph_NoDuplicateOutputs(t *testing.T) {
	input := "aa"

	result := Homograph(input)

	seen := make(map[string]struct{})
	for _, r := range result {
		if _, exists := seen[r]; exists {
			t.Fatalf("duplicate output detected: %s", r)
		}
		seen[r] = struct{}{}
	}
}

// ---------------------------------------------------
// IMMUTABILITY / INPUT SAFETY
// ---------------------------------------------------

func TestHomograph_InputNotMutated(t *testing.T) {
	input := "test"
	original := input

	_ = Homograph(input)

	if input != original {
		t.Fatalf("input string was mutated: expected %s, got %s", original, input)
	}
}

// ---------------------------------------------------
// STABILITY TEST (ORDER NOT GUARANTEED)
// ---------------------------------------------------

func TestHomograph_DeterministicOutputSet(t *testing.T) {
	input := "sip"

	result1 := sortStrings(Homograph(input))
	result2 := sortStrings(Homograph(input))

	if !reflect.DeepEqual(result1, result2) {
		t.Fatalf("non-deterministic output: %v vs %v", result1, result2)
	}
}
