package ipk

// Format describes which IPK package format to use.
// The IPK specification allows .tar, .tar.gz, .ar, and .ar.gz.
// However, this implementation currently only supports .tar and .tar.gz.
type Format uint32

// Constants describing the format used for a specific IPK package.
const (
	// FormatUnknown indicates the IPK format is unknown.
	FormatUnknown Format = 0

	// formatGZip is a flag indicating the package file is gzipped.
	formatGZip Format = 1

	// FormatTar indicates the package is in POSIX .tar format.
	FormatTar = 2

	// FormatTarGZip indicates the package is in .tar.gz format.
	FormatTarGZip = FormatTar | formatGZip
)
