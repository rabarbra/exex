package binfile

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// buildSample compiles a small C program with -g into a temp dir and returns
// the path. Tests are skipped if no C compiler is on PATH.
func buildSample(t *testing.T) string {
	t.Helper()
	cc, err := exec.LookPath("gcc")
	if err != nil {
		cc, err = exec.LookPath("cc")
	}
	if err != nil {
		t.Skip("no C compiler available")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "sample.c")
	bin := filepath.Join(dir, "sample")
	const code = `
#include <stdio.h>
int multiply(int a, int b) {
    return a * b;
}
int main(int argc, char **argv) {
    int r = multiply(argc, 7);
    printf("r=%d\n", r);
    return r;
}
`
	if err := os.WriteFile(src, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command(cc, "-g", "-O0", "-o", bin, src).CombinedOutput()
	if err != nil {
		t.Fatalf("compile failed: %v\n%s", err, out)
	}
	return bin
}

func TestOpenAndProbeSampleBinary(t *testing.T) {
	path := buildSample(t)
	f, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if f.Entry() == 0 {
		t.Fatal("expected non-zero entry")
	}

	// Symbol names may carry a leading underscore (Mach-O / macОS).
	sym := func(name string) (Symbol, bool) {
		for _, s := range f.Symbols {
			if s.Name == name || s.Name == "_"+name {
				return s, true
			}
		}
		return Symbol{}, false
	}
	mainSym, foundMain := sym("main")
	_, foundMultiply := sym("multiply")
	if !foundMain || !foundMultiply {
		t.Fatalf("missing expected symbols: main=%v multiply=%v", foundMain, foundMultiply)
	}

	// Sanity-check section lookup over the entry address.
	if sec := f.SectionAt(f.Entry()); sec == nil {
		t.Fatalf("entry 0x%x not mapped to any section", f.Entry())
	}

	// The executable image should cover the entry point.
	if _, ok := f.ExecImage().PosForAddr(f.Entry()); !ok {
		t.Fatalf("entry 0x%x not present in the executable image", f.Entry())
	}

	// DWARF is optional: linked Mach-O executables keep their debug info in
	// separate .o/dSYM bundles. Only assert source mapping when DWARF is
	// actually embedded (e.g. ELF builds with -g).
	if f.HasDWARF() {
		file, line := f.LookupAddr(mainSym.Addr)
		if file == "" || line == 0 {
			t.Fatalf("addr→source lookup failed for main at 0x%x", mainSym.Addr)
		}
		if !strings.HasSuffix(file, "sample.c") {
			t.Fatalf("unexpected source file: %s", file)
		}
	}
}

func TestImageWindowContainingBoundsAndPreservesTarget(t *testing.T) {
	im := &Image{Data: []byte("abcdefghijklmnopqrstuvwxyz")}
	im.Regions = []Region{{Addr: 0x1000, Size: uint64(len(im.Data)), Off: 0, Name: ".text"}}

	win, ok := im.WindowContaining(0x1008, 10, 3)
	if !ok {
		t.Fatal("expected window containing target")
	}
	if win.Start != 5 || win.End != 15 {
		t.Fatalf("window bounds = [%d,%d), want [5,15)", win.Start, win.End)
	}
	if win.Addr != 0x1005 {
		t.Fatalf("window addr = 0x%x, want 0x1005", win.Addr)
	}

	win, ok = im.WindowContaining(0x1018, 10, 3)
	if !ok {
		t.Fatal("expected trailing window containing target")
	}
	if win.Start != 16 || win.End != 26 {
		t.Fatalf("trailing bounds = [%d,%d), want [16,26)", win.Start, win.End)
	}
	if win.Addr > 0x1018 || win.Addr+uint64(len(win.Data)) <= 0x1018 {
		t.Fatalf("window [0x%x,0x%x) does not contain target", win.Addr, win.Addr+uint64(len(win.Data)))
	}
}

func TestImageRegionAtRejectsGapsAndEnd(t *testing.T) {
	im := &Image{Data: []byte("abcdef")}
	im.Regions = []Region{
		{Addr: 0x1000, Size: 2, Off: 0, Name: ".text"},
		{Addr: 0x2000, Size: 2, Off: 4, Name: ".data"},
	}

	if got := im.RegionAt(1); got == nil || got.Name != ".text" {
		t.Fatalf("RegionAt(1) = %#v, want .text", got)
	}
	if got := im.RegionAt(4); got == nil || got.Name != ".data" {
		t.Fatalf("RegionAt(4) = %#v, want .data", got)
	}
	for _, pos := range []int{-1, 2, 3, 6} {
		if got := im.RegionAt(pos); got != nil {
			t.Fatalf("RegionAt(%d) = %#v, want nil", pos, got)
		}
	}
}

func TestFileAccessorsAndMappingHelpers(t *testing.T) {
	f := &File{
		raw:       []byte("abcd"),
		addrWidth: 8,
		header:    []string{"Magic: test"},
		Sections:  []Section{{Name: ".text", Addr: 0x1000, Size: 4, Alloc: true, Exec: true}},
	}
	if !f.IsMapped(0x1002) || f.IsMapped(0x2000) {
		t.Fatal("IsMapped returned unexpected values")
	}
	if !IsExecSection(&f.Sections[0]) || IsExecSection(nil) || IsExecSection(&Section{Size: 1}) {
		t.Fatal("IsExecSection returned unexpected values")
	}
	if got := f.Raw(); string(got) != "abcd" {
		t.Fatalf("Raw = %q", got)
	}
	if got := f.AddrHexWidth(); got != 8 {
		t.Fatalf("AddrHexWidth = %d, want 8", got)
	}
	if got := (&File{}).AddrHexWidth(); got != 16 {
		t.Fatalf("zero AddrHexWidth = %d, want 16", got)
	}
	if got := f.HeaderInfo(); len(got) != 1 || got[0] != "Magic: test" {
		t.Fatalf("HeaderInfo = %#v", got)
	}
}

func TestSourceLinesFindsRelativeToBinary(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "main.c")
	if err := os.WriteFile(src, []byte("int main(void) {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := &File{Path: filepath.Join(dir, "app")}
	lines := f.SourceLines("main.c")
	if len(lines) < 1 || lines[0] != "int main(void) {}" {
		t.Fatalf("SourceLines = %#v", lines)
	}
	if lines2 := f.SourceLines("main.c"); len(lines2) != len(lines) || lines2[0] != lines[0] {
		t.Fatalf("cached SourceLines = %#v", lines2)
	}
}

func TestCloseIsIdempotent(t *testing.T) {
	closed := 0
	f := &File{raw: []byte("abc"), unmap: func() error { closed++; return nil }}
	if err := f.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	if closed != 1 || f.raw != nil {
		t.Fatalf("closed = %d raw = %#v, want one close and nil raw", closed, f.raw)
	}
}
