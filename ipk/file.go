package ipk

import (
	"archive/tar"
	"bytes"
	"strings"
)

// file describes a file in the data portion of the IPK package.
type file struct {
	// header is the file header.
	header *tar.Header

	// data is the compressed file data (regular file),
	// or nil (everything else).
	data *bytes.Buffer
}

// slash prepends path with ./
func slash(path string) string {
	if strings.HasPrefix(path, "./") {
		return path
	}
	if strings.HasPrefix(path, "/") {
		return "." + path
	}
	return "./" + path
}

// unslash returns path with initial ./ or / removed.
func unslash(path string) string {
	if strings.HasPrefix(path, "./") {
		return unslash(path[2:])
	}
	if strings.HasPrefix(path, "/") {
		return unslash(path[1:])
	}
	return path
}
