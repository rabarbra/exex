package binfile

import (
	"os"
	"runtime"
	"testing"
)

// TestOpenSystemMachO exercises the Mach-O path against a real (usually fat)
// system binary. /bin/ls is Mach-O only on macOS (it's ELF on Linux CI), so the
// test is darwin-only; skipped elsewhere or when the file isn't present.
func TestOpenSystemMachO(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("system Mach-O binary only available on macOS")
	}
	const path = "/bin/ls"
	if _, err := os.Stat(path); err != nil {
		t.Skipf("%s not present", path)
	}
	f, err := Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if f.Format != FormatMachO {
		t.Fatalf("format = %q, want Mach-O", f.Format)
	}
	if f.Entry() == 0 {
		t.Fatal("entry is zero")
	}
	if len(f.Sections) == 0 {
		t.Fatal("no sections")
	}
	if len(f.Symbols) == 0 {
		t.Fatal("no symbols")
	}

	// The entry point must live inside the executable image we'll disassemble.
	if _, ok := f.ExecImage().PosForAddr(f.Entry()); !ok {
		t.Fatalf("entry 0x%x not in executable image", f.Entry())
	}
	// Every byte of the file must be addressable in the raw view.
	if got := len(f.Raw()); got == 0 {
		t.Fatal("raw file is empty")
	}
	if f.VAImage().Len() == 0 {
		t.Fatal("virtual-address image is empty")
	}

	t.Logf("format=%s arch=%d entry=0x%x sections=%d symbols=%d raw=%d va-image=%d exec-image=%d",
		f.Format, f.Arch(), f.Entry(), len(f.Sections), len(f.Symbols),
		len(f.Raw()), f.VAImage().Len(), f.ExecImage().Len())
}

// TestFatMagicVsJavaClass guards the 0xCAFEBABE ambiguity: a fat Mach-O has a
// small architecture count, while a Java .class has minor/major version (major
// >= 45) where the count would be, so it must not be detected as Mach-O.
func TestFatMagicVsJavaClass(t *testing.T) {
	fat := []byte{0xca, 0xfe, 0xba, 0xbe, 0, 0, 0, 2}      // 2 architectures
	class := []byte{0xca, 0xfe, 0xba, 0xbe, 0, 0, 0, 0x34} // Java 8 (major 52)
	if !isFatMachO(fat) || !isMachO(fat) {
		t.Fatal("a sane fat header must be detected as fat Mach-O")
	}
	if isFatMachO(class) || isMachO(class) {
		t.Fatal("a Java .class must not be detected as (fat) Mach-O")
	}
}
