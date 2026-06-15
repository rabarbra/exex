//go:build unix

package binfile

import (
	"os"

	"golang.org/x/sys/unix"
)

// mapFile memory-maps path read-only and returns the bytes together with a
// closer that unmaps them. Mapping avoids copying the whole file into the heap
// (a 140 MB binary is otherwise a 140 MB allocation + read); pages fault in from
// the page cache only as the parser touches them. The returned slice aliases the
// mapping, so it stays valid until closer is called — keep it for the File's
// lifetime. Empty files fall back to a nil slice (mmap rejects zero length).
func mapFile(path string) (data []byte, closer func() error, err error) {
	fh, err := os.Open(path)
	if err != nil {
		return nil, nil, err
	}
	defer fh.Close()
	fi, err := fh.Stat()
	if err != nil {
		return nil, nil, err
	}
	size := fi.Size()
	if size <= 0 {
		return nil, func() error { return nil }, nil
	}
	b, err := unix.Mmap(int(fh.Fd()), 0, int(size), unix.PROT_READ, unix.MAP_SHARED)
	if err != nil {
		// Fall back to a plain read when the file can't be mapped (e.g. a
		// special file); correctness over speed.
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return nil, nil, rerr
		}
		return data, func() error { return nil }, nil
	}
	// Advise the kernel we'll read it roughly sequentially; best-effort.
	_ = unix.Madvise(b, unix.MADV_WILLNEED)
	return b, func() error { return unix.Munmap(b) }, nil
}
