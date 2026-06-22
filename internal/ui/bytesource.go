package ui

import "github.com/rabarbra/exex/internal/binfile"

// byteSource is the random-access byte stream the hex/raw views read from. It
// abstracts over the raw file bytes (a flat slice) and the virtual-address Image
// (region-backed, zero-copy slices into the file). Bounded reads go through Bytes
// (zero-copy within one region); whole-image scans iterate Runs so bytes.Index
// runs on the real bytes at full speed, region by region.
type byteSource interface {
	Len() int
	At(i int) byte
	Bytes(start, end int) []byte
	Runs() []binfile.Run
}

// rawBytes adapts a flat byte slice (the raw-file view) to byteSource. It is a
// single run spanning the whole slice.
type rawBytes []byte

func (r rawBytes) Len() int                    { return len(r) }
func (r rawBytes) At(i int) byte               { return r[i] }
func (r rawBytes) Bytes(start, end int) []byte { return r[start:end] }
func (r rawBytes) Runs() []binfile.Run         { return []binfile.Run{{Off: 0, B: r}} }
