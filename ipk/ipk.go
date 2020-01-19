package ipk

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
)

// allowedFieldNames is the set of allowed field names.
var allowedFieldNames = regexp.MustCompile(`^[[:alpha:]-]+$`)

// T describes an IPK package.
type T struct {
	// fields are the control fields with their values.
	fields map[string]string

	// conffiles is the list of configuration files.
	// Methods may sort this list before operating on it,
	// so the order of files may change.
	conffiles []string

	// scripts are the scripts to be included in the ipk.
	// The keys are the script names.
	// The values are of type file; however, in the tar header of the values,
	// only the size field is used when writing the IPK.
	scripts map[string]file

	// files are the files in the IPK.
	// The keys are absolute path names.
	files map[string]file
}

// New returns a new, empty IPK package.
func New() *T {
	return &T{
		fields:  make(map[string]string),
		scripts: make(map[string]file),
		files:   make(map[string]file),
	}
}

// AddField adds a field to the control file.
func (t *T) AddField(name, content string) error {
	if !allowedFieldNames.MatchString(name) {
		return errors.New("Invalid field name")
	}
	content = strings.TrimSpace(content)
	if strings.ContainsRune(content, '\n') {
		return errors.New("Content contains newline character")
	}
	if _, ok := t.fields[name]; ok {
		return fmt.Errorf("Field '%s' already exists", name)
	}
	t.fields[name] = content
	return nil
}

// GetField obtains the content of the named field.
func (t *T) GetField(name string) (content string, ok bool) {
	content, ok = t.fields[name]
	return
}

// AddConffile adds the specified configuration file to conffiles.
// The path must be absolute and different from the root directory.
func (t *T) AddConffile(pathname string) error {
	cleaned := path.Clean(pathname)
	if !path.IsAbs(cleaned) {
		return fmt.Errorf("Not an absolute pathname: %s", pathname)
	}
	if cleaned == "/" {
		return errors.New("Root not allowed")
	}
	t.conffiles = append(t.conffiles, cleaned)
	return nil
}

// Conffiles returns a copy of the current configuration files.
func (t *T) Conffiles() []string {
	result := make([]string, len(t.conffiles))
	copy(result, t.conffiles)
	return result
}

// AddScript adds a script with the specified name, to be read from the
// specified reader.
func (t *T) AddScript(name string, r io.Reader) (err error) {
	if r == nil {
		return errors.New("Nil reader supplied")
	}
	if name == controlName || name == confName ||
		!allowedScriptNames.MatchString(name) {
		return ErrBadScriptName
	}
	if _, ok := t.scripts[name]; ok {
		return fmt.Errorf("Script '%s' already exists", name)
	}
	buf := &bytes.Buffer{}
	gzw, _ := gzip.NewWriterLevel(buf, gzip.BestCompression)
	defer func() {
		if err2 := gzw.Close(); err2 != nil {
			err = fmt.Errorf(
				"Error closing compressed stream for script '%s': %s", name, err2,
			)
		}
	}()
	size, err := io.Copy(gzw, r)
	if err != nil {
		return fmt.Errorf("Error copying script '%s': %s", name, err)
	}
	t.scripts[name] = file{
		header: &tar.Header{
			Name: name,
			Size: size,
		},
		data: buf,
	}
	return nil
}

// ScriptNames returns the names of all scripts in unspecified order.
func (t *T) ScriptNames() []string {
	result := make([]string, 0, len(t.scripts))
	for name := range t.scripts {
		result = append(result, name)
	}
	return result
}

// GetScript returns the named script as a ReadCloser, along with its size.
// The caller is responsible for closing the returned stream.
func (t *T) GetScript(name string) (io.ReadCloser, int64, error) {
	f, ok := t.scripts[name]
	if !ok {
		return nil, -1, fmt.Errorf("Script '%s' does not exist", name)
	}
	gzbuf := bytes.NewReader(f.data.Bytes())
	gzr, err := gzip.NewReader(gzbuf)
	if err != nil {
		return nil, -1, fmt.Errorf(
			"Unable to read compressed script '%s': %s. This should not happen",
			name, err,
		)
	}
	return gzr, f.header.Size, nil
}

// AddFile adds a file to the specified directory, which must be specified
// using an absolute path. The file itself is described
// by the specified info. If the file is not a regular file or a symlink,
// r will be ignored (can be nil). Otherwise, the file data or the linkname is
// read from r.
// The directory will be added recursively with mode 0755 if it is not present
// yet.
// If an entry for the file already exists, os.ErrExist is returned.
// The root directory is always considered to be present.
func (t *T) AddFile(dir string, info os.FileInfo, r io.Reader) (err error) {
	if info == nil {
		return errors.New("Supplied file info is nil")
	}
	if err = t.addDir(dir); err != nil && err != os.ErrExist {
		return fmt.Errorf("Unable to add directory '%s': %s", dir, err)
	}
	fullpath := path.Join(dir, info.Name())
	var f file
	f.header, err = tar.FileInfoHeader(info, "/")
	if err != nil {
		return fmt.Errorf("Unable to create tar header from file info: %s", err)
	}
	f.header.Name = fullpath
	switch f.header.Typeflag {
	case tar.TypeReg, tar.TypeRegA:
		if r == nil {
			return errors.New("Supplied reader is nil")
		}
		f.data = &bytes.Buffer{}
		gzw, _ := gzip.NewWriterLevel(f.data, gzip.BestCompression)
		defer func() {
			if err2 := gzw.Close(); err2 != nil && err == nil {
				err = fmt.Errorf("Unable to close gzip writer for '%s'", fullpath)
			}
		}()
		size, err2 := io.Copy(gzw, r)
		if err2 != nil {
			return fmt.Errorf("Unable to copy data: %s", err2)
		}
		f.header.Size = size
	case tar.TypeSymlink:
		if r == nil {
			return errors.New("Supplied reader is nil")
		}
		buf, err := ioutil.ReadAll(r)
		if err != nil {
			return fmt.Errorf("Unable to read linkname: %s", err)
		}
		f.header.Linkname = string(buf)
	case tar.TypeDir:
		f.header.Name += "/" // restore slash stripped by path.Join
	}
	if _, ok := t.files[f.header.Name]; ok {
		return os.ErrExist
	}
	t.files[f.header.Name] = f
	return nil
}

// addDir adds the specified directory.
// It must be specified as an absolute path.
// If it already exists, os.ErrExist is returned.
func (t *T) addDir(dir string) error {
	cleaned := path.Clean(dir)
	updir, basename := path.Split(cleaned)
	if updir == "" {
		return fmt.Errorf("Invalid directory: %s", dir)
	}
	if basename == "" { // cleaned == "/"
		return os.ErrExist
	}
	return t.AddFile(updir, directoryHeader(basename, time.Now()).FileInfo(), nil)
}

// FileNames returns a list of the names of all files
// (including non-regular files) in this
// IPK package, with their absolute path names. The root directory entry
// will be omitted. The list of files is in no particular order.
func (t *T) FileNames() []string {
	result := make([]string, 0, len(t.files))
	for name := range t.files {
		result = append(result, name)
	}
	return result
}
