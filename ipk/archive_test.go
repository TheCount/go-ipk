package ipk

import (
	"archive/tar"
	"compress/flate"
	"testing"
	"time"
)

// testHeaderCommon performs common tests on the specified header.
func testHeaderCommon(t *testing.T, header *tar.Header) {
	if header.Uid != 0 {
		t.Errorf("Invalid user ID in header: %d", header.Uid)
	}
	if header.Gid != 0 {
		t.Errorf("Invalid group ID in header: %d", header.Gid)
	}
	if header.Uname != archiveOwner {
		t.Errorf("Invalid user name in header: %s", header.Uname)
	}
	if header.Gname != archiveOwner {
		t.Errorf("Invalid group name in header: %s", header.Gname)
	}
	if header.Format != tar.FormatUSTAR {
		t.Errorf("Bad header format: %s", header.Format)
	}
}

// TestFileHeader tests the fileHeader function.
func TestFileHeader(t *testing.T) {
	now := time.Now()
	header := fileHeader("foo", 123, now)
	if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
		t.Errorf("Expected regular file header, got %d", header.Typeflag)
	}
	if header.Name != "foo" {
		t.Errorf("Bad file header name: %s", header.Name)
	}
	if header.Size != 123 {
		t.Errorf("Bad size in header: %d", header.Size)
	}
	if header.Mode != 0644 {
		t.Errorf("Bad file mode in header: %03o", header.Mode)
	}
	if !header.ModTime.Equal(now) {
		t.Errorf("Bad modification time in header: %s", header.ModTime)
	}
	testHeaderCommon(t, header)
}

// TestDirectoryHeader tests the directoryHeader function.
func TestDirectoryHeader(t *testing.T) {
	now := time.Now()
	header := directoryHeader("bar", now)
	if header.Typeflag != tar.TypeDir {
		t.Errorf("Expected directory, got %d", header.Typeflag)
	}
	if header.Name != "bar" {
		t.Errorf("Bad directory header name: %s", header.Name)
	}
	if header.Size != 0 {
		t.Errorf("Non-zero directory size: %d", header.Size)
	}
	if header.Mode != 0755 {
		t.Errorf("Bad directory mode in header: %03o", header.Mode)
	}
	if !header.ModTime.Equal(now) {
		t.Errorf("Bad modification time in header: %s", header.ModTime)
	}
	testHeaderCommon(t, header)
}

// TestCompressionLevel tests the compressionLevel function.
func TestCompressionLevel(t *testing.T) {
	if compressionLevel(false) != flate.NoCompression {
		t.Error("Expected no compression")
	}
	if compressionLevel(true) != flate.BestCompression {
		t.Error("Expected best compression")
	}
}
