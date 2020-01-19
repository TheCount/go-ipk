package ipk

// MockReader mocks an io.Reader.
type MockReader func([]byte) (int, error)

// Read proxies to this MockReader.
func (r MockReader) Read(p []byte) (int, error) {
	return r(p)
}
