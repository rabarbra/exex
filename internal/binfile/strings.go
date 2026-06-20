package binfile

import "sort"

// Printable-string extraction over the raw file, à la strings(1), annotated
// with the mapped virtual address and section when the bytes live in one.

// StringEntry is one printable run found in the file.
type StringEntry struct {
	Offset  uint64 // file offset of the first byte
	Addr    uint64 // mapped virtual address, when HasAddr
	HasAddr bool
	Section string // owning section name, when known
	Text    string
}

// minString is the shortest run of printable bytes reported as a string.
const minString = 4

// Strings scans the whole file for runs of printable ASCII at least minString
// bytes long. The result is cached. Each entry is mapped back to a virtual
// address / section when its offset falls inside a section's file bytes.
func (f *File) Strings() []StringEntry {
	if f.strings != nil {
		return f.strings
	}
	f.strings = f.extractStrings()
	return f.strings
}

// extractStrings performs the uncached printable-string scan.
func (f *File) extractStrings() []StringEntry {
	var out []StringEntry
	data := f.raw
	// Sort the file-backed sections once so each found string is mapped to its
	// section with a binary search instead of an O(sections) scan.
	secs := f.fileSectionsByOffset()
	start := -1
	flush := func(end int) {
		if start < 0 || end-start < minString {
			start = -1
			return
		}
		e := StringEntry{Offset: uint64(start), Text: string(data[start:end])}
		if sec := sectionAtSortedOffset(secs, uint64(start)); sec != nil {
			e.Section = sec.Name
			if sec.Alloc {
				e.Addr = sec.Addr + (uint64(start) - sec.Offset)
				e.HasAddr = true
			}
		}
		out = append(out, e)
		start = -1
	}
	for i := 0; i < len(data); i++ {
		if b := data[i]; b >= 0x20 && b < 0x7f {
			if start < 0 {
				start = i
			}
			continue
		}
		flush(i)
	}
	flush(len(data))
	return out
}

// fileSectionsByOffset returns the sections that occupy file bytes, sorted by
// file offset, for binary-searched offset→section lookups.
func (f *File) fileSectionsByOffset() []*Section {
	var secs []*Section
	for i := range f.Sections {
		if f.Sections[i].FileSize > 0 {
			secs = append(secs, &f.Sections[i])
		}
	}
	sort.Slice(secs, func(i, j int) bool { return secs[i].Offset < secs[j].Offset })
	return secs
}

// sectionAtSortedOffset returns the section whose file bytes cover off, from a
// slice sorted by file offset (well-formed section file ranges don't overlap).
func sectionAtSortedOffset(secs []*Section, off uint64) *Section {
	i := sort.Search(len(secs), func(i int) bool { return secs[i].Offset > off })
	if i == 0 {
		return nil
	}
	s := secs[i-1]
	if off < s.Offset+s.FileSize {
		return s
	}
	return nil
}
