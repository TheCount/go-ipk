package ipk

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"time"
)

// Write writes this IPK package to the specified writer in the specified
// format.
func (t *T) Write(w io.Writer, format Format) (err error) {
	if w == nil {
		return errors.New("Nil writer supplied")
	}
	if format&^formatGZip != FormatTar {
		return errors.New("Unsupported format")
	}
	var tarUnderlyingWriter io.Writer
	if format&formatGZip != 0 {
		gzw, err := gzip.NewWriterLevel(w, gzip.BestCompression)
		if err != nil {
			return fmt.Errorf("Error opening gzip stream: %s", err)
		}
		defer func() {
			if err2 := gzw.Close(); err2 != nil && err == nil {
				err = fmt.Errorf("Error closing gzip stream: %s", err2)
			}
		}()
		tarUnderlyingWriter = gzw
	} else {
		tarUnderlyingWriter = w
	}
	tw := tar.NewWriter(tarUnderlyingWriter)
	defer func() {
		if err2 := tw.Close(); err2 != nil && err == nil {
			err = fmt.Errorf("Error closing TAR stream: %s", err2)
		}
	}()
	return t.writeTar(tw, format&formatGZip == 0)
}

// WriteFile writes this IPK package to the specified file in the specified
// format. The file must be open for writing or appending.
// WriteFile does not close the file after writing.
func (t *T) WriteFile(f *os.File, format Format) error {
	if f == nil {
		return errors.New("Nil file supplied")
	}
	return t.Write(f, format)
}

// WritePath writes this IPK package to the file at the specified path.
// If the file does not exist, it will be created. Otherwise, it will be
// truncated before writing the IPK data.
func (t *T) WritePath(path string, format Format) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("Unable to create '%s': %s", path, err)
	}
	defer func() {
		if err2 := f.Close(); err2 != nil && err == nil {
			err = fmt.Errorf("Error closing '%s': %s", path, err2)
		}
	}()
	return t.Write(f, format)
}

// writeTar writes this IPK package to the specified tar archive.
// The control and data archives will be compressed if and only if
// compress is true.
func (t *T) writeTar(w *tar.Writer, compress bool) error {
	now := time.Now()
	if err := writeDebianBinary(w, now); err != nil {
		return err
	}
	if err := t.writeControlArchive(w, compress, now); err != nil {
		return err
	}
	if err := t.writeDataArchive(w, compress, now); err != nil {
		return err
	}
	return nil
}

// writeDebianBinary adds the mandatory debian-binary entry to the specified
// tar archive with the specified modification time.
func writeDebianBinary(w *tar.Writer, modTime time.Time) error {
	const filename = "./debian-binary"
	const filecontent = "2.0\n"
	if err := w.WriteHeader(
		fileHeader(filename, int64(len(filecontent)), modTime),
	); err != nil {
		return fmt.Errorf("Unable to write header for '%s': %s", filename, err)
	}
	if _, err := w.Write([]byte(filecontent)); err != nil {
		return fmt.Errorf("Unable to write content for '%s': %s", filename, err)
	}
	return nil
}
