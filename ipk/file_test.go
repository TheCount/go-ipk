package ipk

import (
	"testing"
)

// TestSlash tests the slash function.
func TestSlash(t *testing.T) {
	testcases := [][2]string{
		{"", "./"},
		{"foo", "./foo"},
		{"/bar", "./bar"},
		{"./baz", "./baz"},
	}
	for _, cse := range testcases {
		if result := slash(cse[0]); result != cse[1] {
			t.Errorf("slash(%s) expected '%s', got '%s'", cse[0], cse[1], result)
		}
	}
}

// TestUnslash tests the unslash function.
func TestUnslash(t *testing.T) {
	testcases := [][2]string{
		{"", ""},
		{".", "."},
		{"..", ".."},
		{"/", ""},
		{"/foo", "foo"},
		{"///foo", "foo"},
		{"./", ""},
		{"./foo", "foo"},
		{".///foo", "foo"},
		{"././", ""},
		{"././foo", "foo"},
		{"./foo/./", "foo/./"},
	}
	for _, cse := range testcases {
		if result := unslash(cse[0]); result != cse[1] {
			t.Errorf("unslash(%s) expected '%s', got '%s'", cse[0], cse[1], result)
		}
	}
}
