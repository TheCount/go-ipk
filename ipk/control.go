package ipk

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"
)

// Control fields
const (
	ControlPackage    = "Package"
	ControlVersion    = "Version"
	ControlArch       = "Architecture"
	ControlDep        = "Depends"
	ControlMaintainer = "Maintainer"
	ControlDesc       = "Description"
	ControlHomepage   = "Homepage"
)

const (
	// controlArchiveName is the name of the control archive.
	controlArchiveName = "control.tar.gz"

	// controlName is the name of the control file in the control archive.
	controlName = "control"

	// confName is the name of the conffiles file in the control archive.
	confName = "conffiles"
)

var (
	// allowedScriptNames is the set of allowed names for a script.
	// This set does not exclude "control" and "conffiles"; equality with these
	// special names must be checked separately.
	allowedScriptNames = regexp.MustCompile(`^[[:lower:]]+$`)

	// ErrBadScriptName is returned if a script name breaks the rules for script
	// names in a control archive.
	ErrBadScriptName = errors.New("Bad script name")
)

// readControlArchive reads in control files from the specified control
// archive reader.
func (t *T) readControlArchive(r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return errors.New("Unable to read compressed control archive")
	}
	defer gzr.Close()
	tr := tar.NewReader(gzr)
	var controlRead, conffilesRead bool
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("Unable to read control tar header: %s", err)
		}
		if header.Typeflag != tar.TypeReg && header.Typeflag != tar.TypeRegA {
			continue
		}
		name := path.Clean(unslash(header.Name))
		switch name {
		case controlName:
			if controlRead {
				return errors.New("Control file has already been read")
			}
			if err := t.readControl(tr); err != nil {
				return fmt.Errorf("Error reading control file: %s", err)
			}
			controlRead = true
		case confName:
			if conffilesRead {
				return errors.New("Conffiles have already been read")
			}
			if err := t.readConffiles(tr); err != nil {
				return fmt.Errorf("Error reading conffiles: %s", err)
			}
			conffilesRead = true
		default:
			if err := t.AddScript(name, tr); err == ErrBadScriptName {
				continue
			} else if err != nil {
				return fmt.Errorf("Error reading script '%s': %s", name, err)
			}
		}
	}
	if !controlRead {
		return errors.New("No control file found")
	}
	return nil
}

// readControl reads in the control file from the control archive, storing
// the fields. Blank and garbage lines are ignored. Duplicate fields cause an
// error.
func (t *T) readControl(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for i := 1; scanner.Scan(); i++ {
		if err := t.readControlLine(scanner.Text()); err != nil {
			return fmt.Errorf("Error reading line %d of control file: %s", i, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("Error reading control file: %s", err)
	}
	return nil
}

// readControlLine reads in a line from the control file.
func (t *T) readControlLine(line string) error {
	colonIdx := strings.IndexByte(line, ':')
	if colonIdx < 0 {
		// ignore this line
		return nil
	}
	key := line[0:colonIdx]
	value := line[colonIdx+1:]
	if err := t.AddField(key, value); err != nil {
		return fmt.Errorf("Unable to add field '%s': %s", key, err)
	}
	return nil
}

// readConffiles parses the conffiles file.
func (t *T) readConffiles(r io.Reader) error {
	scanner := bufio.NewScanner(r)
	for i := 1; scanner.Scan(); i++ {
		if err := t.readConffile(scanner.Text()); err != nil {
			return fmt.Errorf("Error reading line %d of conffiles: %s", i, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("Error reading conffiles: %s", err)
	}
	sort.Strings(t.conffiles)
	return nil
}

// readConffile reads in a line from conffiles.
// Empty lines are ignored. Otherwise, everything that is not an absolute
// pathname other than "/" is an error.
func (t *T) readConffile(line string) error {
	if line == "" {
		return nil
	}
	return t.AddConffile(line)
}

// writeControlArchive writes the control archive to the specified tar archive
// with the specified modification time. The archive will be compressed if and only
// if compress is true.
func (t *T) writeControlArchive(
	w *tar.Writer, compress bool, modTime time.Time,
) error {
	archive, size, err := t.createControlArchive(compress, modTime)
	if err != nil {
		return err
	}
	defer archive.Close()
	if err = w.WriteHeader(
		fileHeader(slash(controlArchiveName), size, modTime),
	); err != nil {
		return fmt.Errorf(
			"Unable to write tar header for '%s': %s", controlArchiveName, err,
		)
	}
	if _, err = io.Copy(w, archive); err != nil {
		return fmt.Errorf("Unable to write '%s': %s", controlArchiveName, err)
	}
	return nil
}

// createControlArchive creates the control archive and returns it,
// along with its size. The caller is responsible for closing it.
// The returned archive will be compressed if and only if compress is true.
// The modTime will be used for all the files in the archive.
func (t *T) createControlArchive(compress bool, modTime time.Time) (
	r io.ReadCloser, size int64, err error,
) {
	aw, err := newTempArchive("ipkcontrol", compress)
	if err != nil {
		return nil, -1, fmt.Errorf(
			"Unable to create temporary control archive: %s", err,
		)
	}
	defer func() {
		var err2 error
		if r, size, err2 = aw.Finish(); err == nil {
			err = err2
		}
	}()
	err = t.fillControlArchive(aw.Writer, modTime)
	return
}

// fillControlArchive filles the control archive w. The modTime will be used
// for all its entries.
func (t *T) fillControlArchive(w *tar.Writer, modTime time.Time) error {
	// Add root directory. This appears to be optional.
	if err := w.WriteHeader(directoryHeader("./", modTime)); err != nil {
		return fmt.Errorf("Unable to add root entry: %s", err)
	}
	// Add control and conffiles
	if err := t.addControl(w, modTime); err != nil {
		return err
	}
	if err := t.addConffiles(w, modTime); err != nil {
		return err
	}
	// Add scripts
	for scriptname, file := range t.scripts {
		if err := addScript(
			w, scriptname, file.header.Size, modTime, file.data,
		); err != nil {
			return fmt.Errorf("Unable to add script '%s': %s", scriptname, err)
		}
	}
	return nil
}

// addControl adds a control file to the specified tar archive, with the
// specified modification time.
func (t *T) addControl(w *tar.Writer, modTime time.Time) error {
	buf := &bytes.Buffer{}
	// Add mandatory fields first
	mandatoryFields := []string{
		ControlPackage, ControlVersion, ControlArch, ControlMaintainer, ControlDesc,
	}
	for _, field := range mandatoryFields {
		content, ok := t.fields[field]
		if !ok {
			return fmt.Errorf("Mandatory control field '%s' missing", field)
		}
		if _, err := buf.WriteString(
			fmt.Sprintf("%s: %s\n", field, content),
		); err != nil {
			return fmt.Errorf(
				"Unable to write mandatory field '%s' to buffer: %s", field, err,
			)
		}
	}
	// Add remaining fields
	sort.Strings(mandatoryFields)
	for field, content := range t.fields {
		idx := sort.SearchStrings(mandatoryFields, field)
		if idx < len(mandatoryFields) && mandatoryFields[idx] == field {
			continue
		}
		if _, err := buf.WriteString(
			fmt.Sprintf("%s: %s\n", field, content),
		); err != nil {
			return fmt.Errorf("Unable to write field '%s' to buffer: %s", field, err)
		}
	}
	// Write to tar archive
	if err := w.WriteHeader(
		fileHeader(slash(controlName), int64(buf.Len()), modTime),
	); err != nil {
		return fmt.Errorf("Unable to write control file header to archive: %s", err)
	}
	if _, err := io.Copy(w, buf); err != nil {
		return fmt.Errorf("Unable to write control file data to archive: %s", err)
	}
	return nil
}

// addConffiles adds the conffiles file to the control archive, provided there
// are any configuration files.
func (t *T) addConffiles(w *tar.Writer, modTime time.Time) error {
	if len(t.conffiles) == 0 { // nothing to do
		return nil
	}
	sort.Strings(t.conffiles)
	buf := &bytes.Buffer{}
	for _, file := range t.conffiles {
		if _, err := buf.WriteString(file + "\n"); err != nil {
			return fmt.Errorf(
				"Unable to write conffile '%s' to buffer: %s", file, err,
			)
		}
	}
	if err := w.WriteHeader(
		fileHeader(slash(confName), int64(buf.Len()), modTime),
	); err != nil {
		return fmt.Errorf("Unable to write conffiles header to archive: %s", err)
	}
	if _, err := io.Copy(w, buf); err != nil {
		return fmt.Errorf("Unable to write conffiles data to archive: %s", err)
	}
	return nil
}

// addScript adds a script with the specified name to the specified tar archive,
// with the specified size and modification time. The gzdata holds the contents
// of the script in compressed form.
func addScript(
	w *tar.Writer, name string, size int64, modTime time.Time,
	gzdata *bytes.Buffer,
) error {
	header := fileHeader(name, size, modTime)
	header.Mode |= 0111 // make script executable
	return addArchiveEntry(w, header, gzdata)
}
