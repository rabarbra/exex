package dump

import (
	"encoding/binary"
	"os"
	"testing"

	"golang.org/x/arch/arm64/arm64asm"

	"github.com/rabarbra/exex/internal/binfile"
	"github.com/rabarbra/exex/internal/cpufeat"
	"github.com/rabarbra/exex/internal/disasm"
)

func TestClassifyARM64InstMatchesTextClassifierOnSelf(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Skip("no test executable path")
	}
	f, err := binfile.Open(exe)
	if err != nil {
		t.Skipf("open self: %v", err)
	}
	defer f.Close()
	if f.Arch() != disasm.ArchARM64 {
		t.Skip("self is not arm64")
	}

	raw := f.Raw()
	decoded := 0
	for _, s := range f.Sections {
		if !s.Exec || s.FileSize == 0 {
			continue
		}
		start := int(s.Offset)
		end := int(s.Offset + s.FileSize)
		if start < 0 || start >= len(raw) {
			continue
		}
		if end > len(raw) {
			end = len(raw)
		}
		align := int((4 - s.Addr%4) % 4)
		for off := start + align; off+4 <= end; off += 4 {
			inst, err := arm64asm.Decode(raw[off:])
			if err != nil {
				continue
			}
			decoded++
			got := classifyARM64Inst(inst)
			want := cpufeat.ARM64(arm64asm.GNUSyntax(inst))
			if got != want {
				addr := s.Addr + uint64(off-start)
				word := binary.LittleEndian.Uint32(raw[off:])
				t.Fatalf("0x%x %08x %s: classifyARM64Inst=%q, text classifier=%q", addr, word, arm64asm.GNUSyntax(inst), got, want)
			}
		}
	}
	if decoded == 0 {
		t.Skip("no ARM64 instructions decoded from self")
	}
}
