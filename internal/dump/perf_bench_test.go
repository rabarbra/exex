package dump

import (
	"io"
	"os"
	"testing"

	"github.com/rabarbra/exex/internal/binfile"
)

// BenchmarkDisasmDump streams the disasm dump of EXEX_BENCH_BIN; set
// -benchmem (and optionally -memprofile) to attribute allocations. all=true
// covers every section (objdump -D), false only executable ones (-d).
func benchDisasm(b *testing.B, all bool) {
	path := os.Getenv("EXEX_BENCH_BIN")
	if path == "" {
		b.Skip("set EXEX_BENCH_BIN to a real binary")
	}
	f, err := binfile.Open(path)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if err := DisasmTo(io.Discard, f, all); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDisasm(b *testing.B)    { benchDisasm(b, false) }
func BenchmarkDisasmAll(b *testing.B) { benchDisasm(b, true) }
