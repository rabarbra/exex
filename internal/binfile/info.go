package binfile

import (
	"bytes"
	"debug/elf"
	"fmt"
	"strings"
)

// Info holds dynamic-linking and identity bits collected at Open() time.
// Everything is best-effort: missing sections or unparseable data leave the
// corresponding field zero.
type Info struct {
	Interp       string   // .interp content (program interpreter / dynamic linker)
	DynamicLibs  []string // DT_NEEDED entries
	RPath        []string // DT_RPATH entries (legacy)
	RunPath      []string // DT_RUNPATH entries
	SoName       string   // DT_SONAME if this is itself a shared object
	BuildID      string   // hex-encoded .note.gnu.build-id descriptor
	Stripped     bool     // true if no SHT_SYMTAB present
	StaticLinked bool     // true if no PT_INTERP / no dynamic libs
	Libc         LibcInfo
}

// LibcInfo identifies the C runtime the binary links against.
type LibcInfo struct {
	Kind    string // "glibc" | "musl" | "uClibc" | "bionic" | "unknown" | "none"
	Source  string // how we identified it ("interp", "needed", "symbol", "rodata-fingerprint")
	Version string // optional, e.g. "2.35"
}

func (f *File) loadInfo() {
	in := &Info{}

	// .interp
	if sec := f.ELF.Section(".interp"); sec != nil {
		if data, err := sec.Data(); err == nil {
			in.Interp = strings.TrimRight(string(data), "\x00")
		}
	}

	// Dynamic libs / rpath / runpath / soname.
	if libs, err := f.ELF.ImportedLibraries(); err == nil {
		in.DynamicLibs = libs
	}
	if v, err := f.ELF.DynString(elf.DT_RPATH); err == nil {
		in.RPath = splitColon(v)
	}
	if v, err := f.ELF.DynString(elf.DT_RUNPATH); err == nil {
		in.RunPath = splitColon(v)
	}
	if v, err := f.ELF.DynString(elf.DT_SONAME); err == nil && len(v) > 0 {
		in.SoName = v[0]
	}

	in.BuildID = readBuildID(f.ELF)

	// Stripped: SHT_SYMTAB carries the static symbol table; if absent and the
	// binary still has a DYNSYM, it has been strip(1)ped.
	hasSymtab := false
	for _, s := range f.ELF.Sections {
		if s.Type == elf.SHT_SYMTAB {
			hasSymtab = true
			break
		}
	}
	in.Stripped = !hasSymtab

	in.StaticLinked = in.Interp == "" && len(in.DynamicLibs) == 0
	in.Libc = identifyLibc(f, in)

	f.Info = in
}

// splitColon mirrors the way the loader splits DT_RPATH/DT_RUNPATH strings.
func splitColon(v []string) []string {
	var out []string
	for _, s := range v {
		for _, part := range strings.Split(s, ":") {
			part = strings.TrimSpace(part)
			if part != "" {
				out = append(out, part)
			}
		}
	}
	return out
}

// readBuildID parses .note.gnu.build-id (or a SHT_NOTE segment with the same
// name) and returns the descriptor as lower-case hex. Empty on absence.
func readBuildID(ef *elf.File) string {
	sec := ef.Section(".note.gnu.build-id")
	if sec == nil {
		return ""
	}
	data, err := sec.Data()
	if err != nil {
		return ""
	}
	order := ef.ByteOrder
	for off := 0; off+12 <= len(data); {
		nameSz := order.Uint32(data[off:])
		descSz := order.Uint32(data[off+4:])
		nType := order.Uint32(data[off+8:])
		off += 12
		name := data[off : off+int(nameSz)]
		off += align4(int(nameSz))
		desc := data[off : off+int(descSz)]
		off += align4(int(descSz))
		if nType == 3 && bytes.HasPrefix(name, []byte("GNU\x00")) {
			return fmt.Sprintf("%x", desc)
		}
	}
	return ""
}

func align4(n int) int {
	if r := n & 3; r != 0 {
		return n + (4 - r)
	}
	return n
}

// identifyLibc applies a cascade of cheap heuristics. Order matters: the
// interpreter is by far the most reliable signal when present.
func identifyLibc(f *File, in *Info) LibcInfo {
	// 1. From the program interpreter.
	if in.Interp != "" {
		low := strings.ToLower(in.Interp)
		switch {
		case strings.Contains(low, "musl"):
			return LibcInfo{Kind: "musl", Source: "interp"}
		case strings.Contains(low, "uclibc"):
			return LibcInfo{Kind: "uClibc", Source: "interp"}
		case strings.HasPrefix(low, "/system/bin/linker"):
			return LibcInfo{Kind: "bionic", Source: "interp"}
		case strings.Contains(low, "ld-linux") || strings.HasSuffix(low, "/ld.so.1") || strings.HasSuffix(low, "/ld.so.2"):
			return LibcInfo{Kind: "glibc", Source: "interp"}
		}
	}

	// 2. From DT_NEEDED.
	for _, lib := range in.DynamicLibs {
		low := strings.ToLower(lib)
		switch {
		case strings.HasPrefix(low, "libc.musl") || strings.Contains(low, "musl"):
			return LibcInfo{Kind: "musl", Source: "needed"}
		case strings.HasPrefix(low, "libuclibc") || strings.Contains(low, "uclibc"):
			return LibcInfo{Kind: "uClibc", Source: "needed"}
		case low == "libc.so.6":
			return LibcInfo{Kind: "glibc", Source: "needed"}
		case low == "libc.so" || low == "libc.so.0":
			// Often bionic on Android, but could be others. Mark soft.
			return LibcInfo{Kind: "bionic", Source: "needed"}
		}
	}

	// 3. Embedded version-string fingerprints (covers static binaries).
	if k := fingerprintLibcRodata(f); k.Kind != "" {
		return k
	}

	if in.StaticLinked {
		return LibcInfo{Kind: "unknown", Source: "static"}
	}
	if len(in.DynamicLibs) == 0 {
		return LibcInfo{Kind: "none", Source: "no-deps"}
	}
	return LibcInfo{Kind: "unknown", Source: "no-match"}
}

// fingerprintLibcRodata scans read-only data sections for distinctive vendor
// strings. We only check .rodata and .rodata.* to keep this fast.
func fingerprintLibcRodata(f *File) LibcInfo {
	for _, s := range f.ELF.Sections {
		if !strings.HasPrefix(s.Name, ".rodata") {
			continue
		}
		data, err := s.Data()
		if err != nil {
			continue
		}
		if i := bytes.Index(data, []byte("GNU C Library")); i >= 0 {
			return LibcInfo{Kind: "glibc", Source: "rodata-fingerprint", Version: extractGlibcVersion(data[i:])}
		}
		if bytes.Contains(data, []byte("musl libc")) {
			return LibcInfo{Kind: "musl", Source: "rodata-fingerprint", Version: extractMuslVersion(data)}
		}
		if bytes.Contains(data, []byte("uClibc")) {
			return LibcInfo{Kind: "uClibc", Source: "rodata-fingerprint"}
		}
		if bytes.Contains(data, []byte("Bionic")) {
			return LibcInfo{Kind: "bionic", Source: "rodata-fingerprint"}
		}
	}
	return LibcInfo{}
}

// extractGlibcVersion pulls "release version X.Y" from a glibc banner like
// "GNU C Library (Ubuntu GLIBC ...) stable release version 2.35.".
func extractGlibcVersion(s []byte) string {
	// Bound search so we don't scan an entire .rodata.
	end := len(s)
	if end > 512 {
		end = 512
	}
	chunk := string(s[:end])
	const marker = "release version "
	i := strings.Index(chunk, marker)
	if i < 0 {
		return ""
	}
	rest := chunk[i+len(marker):]
	j := strings.IndexAny(rest, ".\n,)")
	for j >= 0 && j+1 < len(rest) && rest[j] == '.' && rest[j+1] >= '0' && rest[j+1] <= '9' {
		k := strings.IndexAny(rest[j+1:], ".\n,)")
		if k < 0 {
			j = -1
			break
		}
		j = j + 1 + k
	}
	if j < 0 {
		return ""
	}
	return strings.TrimSpace(rest[:j])
}

func extractMuslVersion(data []byte) string {
	// musl embeds something like "1.2.4" near "musl libc"; pattern: digits.digits.digits.
	idx := bytes.Index(data, []byte("musl libc"))
	if idx < 0 {
		return ""
	}
	end := idx + 200
	if end > len(data) {
		end = len(data)
	}
	tail := string(data[idx:end])
	// Find a "vX.Y.Z"-ish pattern.
	for i := 0; i < len(tail)-3; i++ {
		if tail[i] >= '0' && tail[i] <= '9' && i+1 < len(tail) && tail[i+1] == '.' {
			j := i
			for j < len(tail) && (tail[j] == '.' || (tail[j] >= '0' && tail[j] <= '9')) {
				j++
			}
			return tail[i:j]
		}
	}
	return ""
}

// SectionData returns the raw bytes of a section, capped at maxBytes (0 = no
// cap). It is the source for the hex viewer.
func (f *File) SectionData(sec *elf.Section, maxBytes int) ([]byte, error) {
	if sec == nil {
		return nil, fmt.Errorf("nil section")
	}
	data, err := sec.Data()
	if err != nil {
		return nil, err
	}
	if maxBytes > 0 && len(data) > maxBytes {
		return data[:maxBytes], nil
	}
	return data, nil
}

// IsExecSection reports whether the section is executable (eligible for
// disassembly).
func IsExecSection(s *elf.Section) bool {
	return s != nil && s.Flags&elf.SHF_EXECINSTR != 0 && s.Size > 0
}

