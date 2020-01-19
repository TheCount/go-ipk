package ipk

import (
	"errors"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

// testScript is a script for testing purposes
const testScript = `#!/bin/sh
echo test
`

// TestNew tests the new function.
func TestNew(t *testing.T) {
	pkg := New()
	if pkg == nil {
		t.Error("New() returns nil")
	}
}

// TestAddField tests the AddField method.
func TestAddField(t *testing.T) {
	pkg := New()
	// Empty or invalid field name
	if err := pkg.AddField("", "foo"); err == nil {
		t.Error("Error expected with empty field name")
	}
	if err := pkg.AddField("1nv4l!d", "foo"); err == nil {
		t.Error("Error expected with invalid field name")
	}
	// Content with newline
	if err := pkg.AddField(ControlPackage, "\nfoo"); err != nil {
		t.Errorf("Leading whitespace unexpected error: %s", err)
	}
	if err := pkg.AddField(ControlArch, "foo\n"); err != nil {
		t.Errorf("Trailing whitespace unexpected error: %s", err)
	}
	if err := pkg.AddField(ControlDesc, "foo\nbar"); err == nil {
		t.Error("Error expected with intermittend newline")
	}
	// Duplicate field
	if err := pkg.AddField(ControlHomepage, "https://foo.bar"); err != nil {
		t.Errorf("Error adding field: %s", err)
	}
	if err := pkg.AddField(ControlHomepage, "https://bar.foo"); err == nil {
		t.Error("Error expected with duplicate field name")
	}
}

// TestGetField tests the GetField method.
func TestGetField(t *testing.T) {
	pkg := New()
	if _, ok := pkg.GetField(ControlPackage); ok {
		t.Error("Field existence not expected")
	}
	if err := pkg.AddField(ControlPackage, "\n\t foo\v\r\n"); err != nil {
		t.Fatalf("Unable to add field: %s", err)
	}
	if content, ok := pkg.GetField(ControlPackage); !ok {
		t.Error("Package field not found")
	} else if content != "foo" {
		t.Errorf("Content not properly trimmed: '%s'", content)
	}
}

// TestAddConffile tests the AddConffile method.
func TestAddConffile(t *testing.T) {
	pkg := New()
	if err := pkg.AddConffile(""); err == nil {
		t.Error("Error expected adding empty conffile")
	}
	if err := pkg.AddConffile("foo"); err == nil {
		t.Error("Error expected adding file with relative path")
	}
	if err := pkg.AddConffile("/"); err == nil {
		t.Error("Error expected adding / as conffile")
	}
	if err := pkg.AddConffile("/etc/pkg/pkg.conf"); err != nil {
		t.Errorf("Error adding configuration file: %s", err)
	}
}

// TestConffiles tests the Conffiles method.
func TestConffiles(t *testing.T) {
	pkg := New()
	if len(pkg.Conffiles()) != 0 {
		t.Error("Primordial conffiles detected")
	}
	if err := pkg.AddConffile("/etc/foo/../bar.conf"); err != nil {
		t.Fatalf("Error adding conffile: %s", err)
	}
	if list := pkg.Conffiles(); len(list) != 1 {
		t.Errorf("Expected 1 conffile, got %d", len(list))
	} else if list[0] != "/etc/bar.conf" {
		t.Errorf("Improperly stripped conffile: %s", list[0])
	}
}

// TestAddScript tests the AddScript method.
func TestAddScript(t *testing.T) {
	r := strings.NewReader(testScript)
	pkg := New()
	if err := pkg.AddScript("", r); err == nil {
		t.Error("Expected error with empty script name")
	}
	if err := pkg.AddScript("1nv4l!d", r); err == nil {
		t.Error("Expected error with empty script name")
	}
	if err := pkg.AddScript(controlName, r); err == nil {
		t.Errorf("Expected error with reserved script name '%s'", controlName)
	}
	if err := pkg.AddScript(confName, r); err == nil {
		t.Errorf("Expected error with reserved script name '%s'", confName)
	}
	if err := pkg.AddScript("foo", nil); err == nil {
		t.Errorf("Expected error with nil reader")
	}
	if err := pkg.AddScript("foo", MockReader(func([]byte) (int, error) {
		return 0, errors.New("boom")
	})); err == nil {
		t.Error("Expected error with broken reader")
	}
	if err := pkg.AddScript("foo", r); err != nil {
		t.Errorf("Unable to add script: %s", err)
	}
	if err := pkg.AddScript("foo", r); err == nil {
		t.Error("Expected error adding same script twice")
	}
}

// TestScriptNames tests the ScriptNames method.
func TestScriptNames(t *testing.T) {
	r := strings.NewReader(testScript)
	pkg := New()
	if len(pkg.ScriptNames()) != 0 {
		t.Error("Primordial scripts detected")
	}
	if err := pkg.AddScript("foo", r); err != nil {
		t.Errorf("Unable to add script: %s", err)
	}
	if list := pkg.ScriptNames(); len(list) != 1 {
		t.Error("Expected script list length 1")
	} else if list[0] != "foo" {
		t.Error("Expected script name foo")
	}
}

// TestAddRegularFile tests adding a regular file to an IPK package.
func TestAddRegularFile(t *testing.T) {
	r := strings.NewReader(testScript)
	pkg := New()
	if err := pkg.AddFile(
		"a/b",
		fileHeader("script.sh", int64(len(testScript)), time.Now()).FileInfo(), r,
	); err == nil {
		t.Error("Expected error using relative directory")
	}
	if err := pkg.AddFile("/etc/a/b/c", nil, r); err == nil {
		t.Error("Expected error on nil file info")
	}
	if err := pkg.AddFile(
		"/etc/a/b/c",
		fileHeader("script.sh", int64(len(testScript)), time.Now()).FileInfo(),
		nil,
	); err == nil {
		t.Error("Expected error using nil reader")
	}
	if err := pkg.AddFile(
		"/etc/a/b/c",
		fileHeader("script.sh", int64(len(testScript)), time.Now()).FileInfo(),
		MockReader(func([]byte) (int, error) {
			return 0, errors.New("boom")
		}),
	); err == nil {
		t.Error("Expected error using broken reader")
	}
	if err := pkg.AddFile(
		"/etc/a/b/c",
		fileHeader("script.sh", int64(len(testScript)), time.Now()).FileInfo(), r,
	); err != nil {
		t.Errorf("Unable to add regular file: %s", err)
	}
	if err := pkg.AddFile(
		"/etc/a/b/c",
		fileHeader("script.sh", int64(len(testScript)), time.Now()).FileInfo(), r,
	); err != os.ErrExist {
		t.Error("Expected file to already exist")
	}
}

// TestFileNames tests the FileNames() method.
func TestFileNames(t *testing.T) {
	r := strings.NewReader(testScript)
	pkg := New()
	if len(pkg.FileNames()) != 0 {
		t.Error("Primordial file names detected")
	}
	if err := pkg.AddFile(
		"/etc/a/b/c",
		fileHeader("script.sh", int64(len(testScript)), time.Now()).FileInfo(), r,
	); err != nil {
		t.Errorf("Unable to add regular file: %s", err)
	}
	expectedFiles := []string{
		"/etc/", "/etc/a/", "/etc/a/b/", "/etc/a/b/c/", "/etc/a/b/c/script.sh",
	}
	actualFiles := pkg.FileNames()
	sort.Strings(actualFiles)
	if len(expectedFiles) != len(actualFiles) {
		t.Fatalf("Expected %d files, got %d", len(expectedFiles), len(actualFiles))
	}
	for i := range expectedFiles {
		if expectedFiles[i] != actualFiles[i] {
			t.Errorf(
				"Expected file at index %d to be '%s', got '%s'",
				i, expectedFiles[i], actualFiles[i],
			)
		}
	}
}
