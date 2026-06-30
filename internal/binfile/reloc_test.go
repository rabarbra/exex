package binfile

import (
	"debug/macho"
	"encoding/binary"
	"testing"
)

func TestDecodeReloc(t *testing.T) {
	bo := binary.LittleEndian

	// RELA64: r_offset(8) | r_info(8) where info = sym<<32 | type.
	rela := make([]byte, 24)
	bo.PutUint64(rela[0:], 0x1000)
	bo.PutUint64(rela[8:], (uint64(5)<<32)|7)
	if off, sym, typ := decodeReloc(rela, 0, true, bo); off != 0x1000 || sym != 5 || typ != 7 {
		t.Errorf("RELA64 = (0x%x,%d,%d), want (0x1000,5,7)", off, sym, typ)
	}

	// REL32: r_offset(4) | r_info(4) where info = sym<<8 | type.
	rel := make([]byte, 8)
	bo.PutUint32(rel[0:], 0x2000)
	bo.PutUint32(rel[4:], (uint32(3)<<8)|6)
	if off, sym, typ := decodeReloc(rel, 0, false, bo); off != 0x2000 || sym != 3 || typ != 6 {
		t.Errorf("REL32 = (0x%x,%d,%d), want (0x2000,3,6)", off, sym, typ)
	}
}

func TestIsELFMappingSymbol(t *testing.T) {
	for _, n := range []string{"$x", "$d", "$a", "$t", "$x.0", "$d.123"} {
		if !isELFMappingSymbol(n) {
			t.Errorf("%q should be a mapping symbol", n)
		}
	}
	for _, n := range []string{"main", "g", "$", "$z", "_$s123", "x$d"} {
		if isELFMappingSymbol(n) {
			t.Errorf("%q should NOT be a mapping symbol", n)
		}
	}
}

func TestHasRelocsUsesAvailabilityBeforeBuild(t *testing.T) {
	built := false
	f := &File{
		relocAvailSet: true,
		relocBuild: func() []Reloc {
			built = true
			return []Reloc{{Offset: 1}}
		},
	}
	if f.HasRelocs() {
		t.Fatal("HasRelocs should be false when the loader knows no reloc data exists")
	}
	if built {
		t.Fatal("HasRelocs should not force relocation build when availability is false")
	}

	f = &File{
		relocAvail:    true,
		relocAvailSet: true,
		relocBuild: func() []Reloc {
			built = true
			return []Reloc{{Offset: 2}}
		},
	}
	built = false
	if !f.HasRelocs() {
		t.Fatal("HasRelocs should build and report relocations when availability is true")
	}
	if !built {
		t.Fatal("HasRelocs should build relocation rows when availability is true")
	}

	f = &File{relocBuild: func() []Reloc {
		built = true
		return []Reloc{{Offset: 3}}
	}}
	built = false
	if !f.HasRelocs() {
		t.Fatal("HasRelocs should preserve lazy-build behavior when availability is unknown")
	}
	if !built {
		t.Fatal("HasRelocs should build when availability is unknown")
	}
}

func TestMachORelocsUseRetainedSymbolNames(t *testing.T) {
	mf := &macho.File{
		FileHeader: macho.FileHeader{Cpu: macho.CpuArm64},
		Sections: []*macho.Section{{
			SectionHeader: macho.SectionHeader{Name: "__text", Seg: "__TEXT", Addr: 0x100},
			Relocs:        []macho.Reloc{{Addr: 4, Value: 1, Type: 2, Extern: true}},
		}},
	}
	rs := machoRelocs(mf, 0x1000, []string{"_unused", "_target"})
	if len(rs) != 1 {
		t.Fatalf("relocs = %d, want 1", len(rs))
	}
	r := rs[0]
	if r.Offset != 0x1104 || r.Type != "ARM64_RELOC_BRANCH26" || r.Section != "__TEXT,__text" || r.Sym != "_target" {
		t.Fatalf("reloc = %+v, want offset 0x1104 type ARM64_RELOC_BRANCH26 section __TEXT,__text sym _target", r)
	}
}

func TestMachOHasRelocs(t *testing.T) {
	if machoHasRelocs(&macho.File{Sections: []*macho.Section{{}}}) {
		t.Fatal("empty Mach-O sections should not report relocs")
	}
	if !machoHasRelocs(&macho.File{Sections: []*macho.Section{{Relocs: []macho.Reloc{{Addr: 1}}}}}) {
		t.Fatal("Mach-O section relocs should be detected")
	}
}
