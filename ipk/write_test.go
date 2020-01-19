package ipk

import (
	"os"
	"strings"
	"testing"
	"time"
)

// TestWriteRead tests creating a complete IPK package, writing it out, and
// reading it back in.
func TestWriteRead(t *testing.T) {
	const (
		testFile       = "test.ipk"
		testPackage    = "foo"
		testVersion    = "1.2.3"
		testArch       = "all"
		testMaintainer = "Ford Prefect <fp@guide.etha>"
		testDesc       = "A foolish barrage"
		testConfdir    = "/etc/foo"
		testConffile   = "/etc/foo/script.sh"
	)
	defer os.Remove(testFile)
	var err error
	r := strings.NewReader(testScript)
	pkg := New()
	if err = pkg.AddField(ControlPackage, testPackage); err != nil {
		t.Errorf("Unable to add package field: %s", err)
	}
	if err = pkg.AddField(ControlVersion, testVersion); err != nil {
		t.Errorf("Unable to add version field: %s", err)
	}
	if err = pkg.AddField(ControlArch, testArch); err != nil {
		t.Errorf("Unable to add architecture field")
	}
	if err = pkg.AddField(ControlMaintainer, testMaintainer); err != nil {
		t.Errorf("Unable to add maintainer field")
	}
	if err = pkg.AddField(ControlDesc, testDesc); err != nil {
		t.Errorf("Unable to add description field: %s", err)
	}
	if err = pkg.AddConffile(testConffile); err != nil {
		t.Error("Unable to add conffile")
	}
	header := fileHeader(testConffile, int64(len(testScript)), time.Now())
	header.Mode |= 0111
	if err := pkg.AddFile(testConfdir, header.FileInfo(), r); err != nil {
		t.Error("Unable to add file")
	}
	if err = pkg.WritePath(testFile, FormatTarGZip); err != nil {
		t.Fatalf("Unable to write test package to '%s': %s", testFile, err)
	}
	if pkg, err = ReadPath(testFile); err != nil {
		t.Fatalf("Unable to read test package '%s': %s", testFile, err)
	}
	if content, ok := pkg.GetField(ControlPackage); !ok {
		t.Error("Package field not found")
	} else if content != testPackage {
		t.Errorf(
			"Expected package field to be '%s', got '%s'", testPackage, content,
		)
	}
	if content, ok := pkg.GetField(ControlVersion); !ok {
		t.Error("Version field not found")
	} else if content != testVersion {
		t.Errorf(
			"Expected version field to be '%s', got '%s'", testVersion, content,
		)
	}
	if content, ok := pkg.GetField(ControlArch); !ok {
		t.Error("Architecture field not found")
	} else if content != testArch {
		t.Errorf(
			"Expected architecture field to be '%s', got '%s'", testArch, content,
		)
	}
	if content, ok := pkg.GetField(ControlMaintainer); !ok {
		t.Error("Maintainer field not found")
	} else if content != testMaintainer {
		t.Errorf(
			"Expected maintainer field to be '%s', got '%s'", testMaintainer, content,
		)
	}
	if content, ok := pkg.GetField(ControlDesc); !ok {
		t.Error("Description field not found")
	} else if content != testDesc {
		t.Errorf(
			"Expected description field to be '%s', got '%s'", testDesc, content,
		)
	}
	if list := pkg.Conffiles(); len(list) != 1 {
		t.Error("Expected one conffile")
	} else if list[0] != testConffile {
		t.Errorf("Expected conffile '%s', got '%s'", testConffile, list[0])
	}
	if len(pkg.ScriptNames()) != 0 {
		t.Error("Expected no scripts")
	}
	list := pkg.FileNames()
	found := false
	for _, file := range list {
		if file == testConffile {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Test file '%s' not found", testConffile)
	}
}
