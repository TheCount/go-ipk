package ipk

import (
	"testing"
)

// TestVersionedDependency tests the versioned dependency function.
func TestVersionedDependency(t *testing.T) {
	// Invalid package name
	if _, err := VersionedDependency("", "", ""); err == nil {
		t.Error("Error expected on empty package name")
	}
	if _, err := VersionedDependency("#$%^&*", "", ""); err == nil {
		t.Error("Error expected on invalid characters in package name")
	}
	if _, err := VersionedDependency("a", "", ""); err == nil {
		t.Error("Error expected on short package name")
	}
	// Invalid relation
	if _, err := VersionedDependency("foo", "invalid", "1.2.3"); err == nil {
		t.Error("Error expected on invalid relation")
	}
	// Valid tests
	testcases := [][4]string{
		{"foo", "", "", "foo"},
		{"foo", "invalid", "", "foo"},
		{"foo", "", "1.2.3", "foo (= 1.2.3)"},
		{"foo", RelStrictlyEarlier, "1.2.3", "foo (<< 1.2.3)"},
		{"foo", RelEarlierEqual, "1.2.3", "foo (<= 1.2.3)"},
		{"foo", RelExact, "1.2.3", "foo (= 1.2.3)"},
		{"foo", RelLaterEqual, "1.2.3", "foo (>= 1.2.3)"},
		{"foo", RelStrictlyLater, "1.2.3", "foo (>> 1.2.3)"},
	}
	for _, cse := range testcases {
		result, err := VersionedDependency(cse[0], cse[1], cse[2])
		if err != nil {
			t.Fatalf("Unexpected error on testcase %#v: %s", cse, err)
		}
		if result != cse[3] {
			t.Errorf(
				"Expected VersionedDependency(%s, %s, %s) == %s, got %s instead",
				cse[0], cse[1], cse[2], cse[3], result,
			)
		}
	}
}

// TestDisjunctiveDependency tests the DisjunctiveDepenency function.
func TestDisjunctiveDependency(t *testing.T) {
	if DisjunctiveDependency() != "" {
		t.Error("DisjunctiveDependency should be empty if called with no arguments")
	}
	if result := DisjunctiveDependency("a"); result != "a" {
		t.Errorf("DisjunctiveDependency with single argument returned %s", result)
	}
	if result := DisjunctiveDependency("a", "b", "c"); result != "a | b | c" {
		t.Errorf("DisjunctiveDependency bad result: %s", result)
	}
}

// TestConjunctiveDependency tests the ConjunctiveDependency function.
func TestConjunctiveDependency(t *testing.T) {
	type testcase struct {
		input  []string
		output string
	}
	testcases := []testcase{
		testcase{[]string{}, ""},
		testcase{[]string{""}, ""},
		testcase{[]string{"a"}, "a"},
		testcase{[]string{"a", "b", "c"}, "a, b, c"},
		testcase{[]string{"", "a", "", "b", ""}, "a, b"},
	}
	for _, cse := range testcases {
		if result := ConjunctiveDependency(cse.input...); result != cse.output {
			t.Errorf(
				"Expected ConjunctiveDependency(%#v) == %s, got %s",
				cse.input, cse.output, result,
			)
		}
	}
}
