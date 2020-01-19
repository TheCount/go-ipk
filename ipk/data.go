package ipk

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"
)

// dataArchiveName is the name of the data archive.
const dataArchiveName = "data.tar.gz"

// readDataArchive reads in the files from the specified data archive reader.
func (t *T) readDataArchive(r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return errors.New("Unable to read compressed data archive")
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("Unable to read data tar header: %s", err)
		}
		name := path.Clean(unslash(header.Name))
		if name == "." || name == ".." || strings.HasPrefix(name, "../") {
			continue
		}
		if _, ok := t.files[name]; ok {
			return fmt.Errorf("Duplicate file data: %s", name)
		}
		if err = t.setFileData("/"+name, header, tr); err != nil {
			return fmt.Errorf("Unable to add file '%s': %s", name, err)
		}
	}
	return nil
}

// setFileData sets the file data for the specified file.
func (t *T) setFileData(
	name string, header *tar.Header, r io.Reader,
) (err error) {
	var data *bytes.Buffer
	if header.Typeflag == tar.TypeReg || header.Typeflag == tar.TypeRegA {
		data = &bytes.Buffer{}
		gzw, _ := gzip.NewWriterLevel(data, gzip.BestCompression)
		defer func() {
			if err2 := gzw.Close(); err2 != nil && err == nil {
				err = fmt.Errorf(
					"Error closing compressed stream for '%s': %s", name, err2,
				)
			}
		}()
		if _, err = io.Copy(gzw, r); err != nil {
			return fmt.Errorf("Unable to copy data for '%s': %s", name, err)
		}
	}
	t.files[name] = file{
		header: header,
		data:   data,
	}
	return nil
}

// writeDataArchive writes the data archive to the specified tar archive with
// the specified modification time. The archive will be compressed if and only
// if compress is true.
func (t *T) writeDataArchive(
	w *tar.Writer, compress bool, modTime time.Time,
) error {
	archive, size, err := t.createDataArchive(compress, modTime)
	if err != nil {
		return err
	}
	defer archive.Close()
	if err = w.WriteHeader(
		fileHeader(slash(dataArchiveName), size, modTime),
	); err != nil {
		return fmt.Errorf(
			"Unable to write tar header for '%s': %s", dataArchiveName, err,
		)
	}
	if _, err = io.Copy(w, archive); err != nil {
		return fmt.Errorf("Unable to write '%s': %s", dataArchiveName, err)
	}
	return nil
}

// createDataArchive creates the data archive and returns it, along with its
// size. The caller is responsible for closing it. The returned archive
// will be compressed if and only if compress is true.
// The modTime will be used for the root directory entry.
func (t *T) createDataArchive(compress bool, modTime time.Time) (
	r io.ReadCloser, size int64, err error,
) {
	aw, err := newTempArchive("ipkdata", compress)
	if err != nil {
		return nil, -1, fmt.Errorf(
			"Unable to create temporary data archive: %s", err,
		)
	}
	defer func() {
		var err2 error
		if r, size, err2 = aw.Finish(); err == nil {
			err = err2
		}
	}()
	err = t.fillDataArchive(aw.Writer, modTime)
	return
}

// fillDataArchive fills the data archive w. The modTime will be used for
// the root directory entry.
func (t *T) fillDataArchive(w *tar.Writer, modTime time.Time) error {
	// Add root directory
	if err := addArchiveEntry(
		w, directoryHeader("./", modTime), nil,
	); err != nil {
		return fmt.Errorf("Unable to add root entry: %s", err)
	}
	// add entries ordered by path name.
	paths := make([]string, 0, len(t.files))
	for key := range t.files {
		paths = append(paths, key)
	}
	sort.Strings(paths)
	for _, path := range paths {
		file := t.files[path]
		if err := addArchiveEntry(w, file.header, file.data); err != nil {
			return fmt.Errorf("Unable to add entry '%s': %s", path, err)
		}
	}
	return nil
}
