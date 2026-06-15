//go:build !unix

package binfile

import "os"

// mapFile falls back to reading the whole file on platforms where we don't wire
// up mmap (e.g. Windows). The closer is a no-op.
func mapFile(path string) (data []byte, closer func() error, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	return b, func() error { return nil }, nil
}
