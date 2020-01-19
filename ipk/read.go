package ipk

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
)

// Read reads an IPK package from the specified reader. The package must have
// the specified format.
func Read(r io.Reader, format Format) (*T, error) {
	if r == nil {
		return nil, errors.New("Nil reader supplied")
	}
	if format&^formatGZip != FormatTar {
		return nil, errors.New("Unsupported format")
	}
	var tarUnderlyingReader io.Reader
	if format&formatGZip != 0 {
		gzr, err := gzip.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("Error opening gzip stream: %s", err)
		}
		defer gzr.Close()
		tarUnderlyingReader = gzr
	} else {
		tarUnderlyingReader = r
	}
	return readTar(tar.NewReader(tarUnderlyingReader))
}

// ReadDetectFormat reads an IPK package from the specified reader/seeker.
// An attempt will be made to autodetect the format.
func ReadDetectFormat(r io.ReadSeeker) (*T, error) {
	if r == nil {
		return nil, errors.New("Nil reader/seeker supplied")
	}
	pos, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, fmt.Errorf(
			"Unable to determine current reader position: %s", err,
		)
	}
	// Try GZip.
	gzr, err := gzip.NewReader(r)
	if err != nil {
		// No GZip, assume pure tar
		if _, err = r.Seek(pos, io.SeekStart); err != nil {
			return nil, fmt.Errorf(
				"Unable to rewind stream after failed gzip detection: %s", err,
			)
		}
		return Read(r, FormatTar)
	}
	// We have GZip.
	defer gzr.Close()
	return Read(gzr, FormatTar)
}

// ReadFile reads an IPK package from the specified file, which must be open
// for reading. An attempt will be made to autodetect the format.
// ReadFile does not close the file after reading.
func ReadFile(f *os.File) (*T, error) {
	if f == nil {
		return nil, errors.New("Nil file supplied")
	}
	return ReadDetectFormat(f)
}

// ReadPath reads an IPK package from the specified path.
func ReadPath(path string) (*T, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Unable to open file '%s': %s", path, err)
	}
	defer f.Close()
	return ReadFile(f)
}

// readTar reads in an IPK from a tar reader.
func readTar(r *tar.Reader) (*T, error) {
	result := New()
	var dataRead, controlRead bool
Loop:
	for {
		header, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Error reading TAR header: %s", err)
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			continue // ignore all non-regular files
		}
		switch unslash(header.Name) {
		case controlArchiveName:
			if controlRead {
				return nil, errors.New("Control archive has already been read")
			}
			if err := result.readControlArchive(r); err != nil {
				return nil, fmt.Errorf("Error reading control archive: %s", err)
			}
			controlRead = true
		case dataArchiveName:
			if dataRead {
				return nil, errors.New("Data archive has already been read")
			}
			if err := result.readDataArchive(r); err != nil {
				return nil, fmt.Errorf("Error reading data archive: %s", err)
			}
			dataRead = true
		default:
			continue Loop
		}
	}
	if !controlRead {
		return nil, errors.New("No control archive found in package")
	}
	return result, nil
}
