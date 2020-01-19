package ipk

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"
)

// archiveOwner is the user the top-level archive files should be owned by.
const archiveOwner = "root"

// fileHeader returns a tar archive header for a special file entry with the
// specified name, size, and modification time. Size must not be negative.
func fileHeader(name string, size int64, modTime time.Time) *tar.Header {
	return specialHeader(name, size, modTime)
}

// directoryHeader returns a tar directory header for a directory owned by root
// with permissions 0755 and the specified modification time.
func directoryHeader(name string, modTime time.Time) *tar.Header {
	return specialHeader(name, -1, modTime)
}

// specialHeader returns a tar archive header for the "special" files or
// directories within the IPK archive or its sub-archives. If size is negative,
// a directory header is returned. Otherwise, a regular file header is returned.
func specialHeader(name string, size int64, modTime time.Time) *tar.Header {
	result := &tar.Header{
		Name:    name,
		Uname:   archiveOwner,
		Gname:   archiveOwner,
		ModTime: modTime,
		Format:  tar.FormatUSTAR,
	}
	if size >= 0 {
		result.Typeflag = tar.TypeReg
		result.Size = size
		result.Mode = 0644
	} else {
		result.Typeflag = tar.TypeDir
		result.Mode = 0755
	}
	return result
}

// compressionLevel returns the best compression level if compress is true,
// and no compression otherwise.
func compressionLevel(compress bool) int {
	if compress {
		return gzip.BestCompression
	}
	return gzip.NoCompression
}

// archiveWriter is an extension of a tar.Writer for writing temporary archives
// with a GZip layer. Once the archive is written, its Finish method returns
// a reader and the archive size.
type archiveWriter struct {
	*tar.Writer

	// gzipWriter compresses the archive (or is just an uncompressed gzip layer
	// in between).
	gzipWriter *gzip.Writer

	// file is a temporary file the archive is written to.
	file *os.File
}

// newTempArchive creates a new, temporary archive.
// The prefix is used to generate a temporary file name.
// The compress parameter controls whether the archive is compressed.
func newTempArchive(prefix string, compress bool) (*archiveWriter, error) {
	f, err := ioutil.TempFile("", prefix+"*.tar.gz")
	if err != nil {
		return nil, err
	}
	result, err := newArchiveForFile(f, compress)
	if err != nil {
		os.Remove(f.Name())
		f.Close()
	}
	return result, err
}

// newArchiveForFile creates a new archive for the specified file, which must
// be open for reading and writing.
// The compress parameter controls whether the archive is compressed.
func newArchiveForFile(f *os.File, compress bool) (*archiveWriter, error) {
	gzw, _ := gzip.NewWriterLevel(f, compressionLevel(compress))
	return &archiveWriter{
		Writer:     tar.NewWriter(gzw),
		gzipWriter: gzw,
		file:       f,
	}, nil
}

// Close should not be called directly on an archiveWriter.
// Use Finish instead.
func (aw *archiveWriter) Close() error {
	panic("internal error: should call Finish() instead of Close()")
}

// Finish finishes writing to this archive and returns the archive data for
// reading along with the archive size.
// On success, it is the responsibility of the caller
// to close the returned ReadCloser.
// On error, this archive writer will handle the cleanup.
func (aw *archiveWriter) Finish() (io.ReadCloser, int64, error) {
	var err error
	if err2 := aw.Writer.Close(); err2 != nil {
		err = fmt.Errorf(
			"Error closing tar writer to '%s': %s", aw.file.Name(), err2,
		)
	}
	if err2 := aw.gzipWriter.Close(); err2 != nil && err == nil {
		err = fmt.Errorf(
			"Error closing gzip writer to '%s': %s", aw.file.Name(), err2,
		)
	}
	size, err2 := aw.file.Seek(0, io.SeekCurrent)
	if err2 != nil && err == nil {
		err = fmt.Errorf(
			"Unable to determine file size of '%s': %s", aw.file.Name(), err2,
		)
	}
	if _, err2 = aw.file.Seek(0, io.SeekStart); err2 != nil && err == nil {
		err = fmt.Errorf("Unable to rewind '%s': %s", aw.file.Name(), err2)
	}
	if err2 = os.Remove(aw.file.Name()); err2 != nil && err == nil {
		err = fmt.Errorf("Unable to remove '%s': %s", aw.file.Name(), err2)
	}
	if err != nil {
		aw.file.Close()
	}
	return aw.file, size, err
}

// addArchiveEntry adds an entry to the specified tar archive.
// The header will be written as tar entry header as is.
// If gzdata is nil, no data will be written (e. g., for directories or
// other special files). Otherwise, gzdata will be interpreted as compressed
// file data. The gzdata buffer will not be changed (i. e., nothing will be
// directly written to or read from it).
func addArchiveEntry(
	w *tar.Writer, header *tar.Header, gzdata *bytes.Buffer,
) error {
	header.Name = slash(header.Name)
	if err := w.WriteHeader(header); err != nil {
		return fmt.Errorf(
			"Unable to write tar header for '%s': %s", header.Name, err,
		)
	}
	if gzdata == nil {
		return nil
	}
	gzr, err := gzip.NewReader(bytes.NewReader(gzdata.Bytes()))
	if err != nil {
		return fmt.Errorf(
			"Unable to create gzip reader for '%s': %s", header.Name, err,
		)
	}
	defer gzr.Close()
	if _, err = io.Copy(w, gzr); err != nil {
		return fmt.Errorf("Unable to copy tar data for '%s': %s", header.Name, err)
	}
	return nil
}
