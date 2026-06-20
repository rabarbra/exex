package binfile

import "fmt"

// Option customises how Open loads a binary.
type Option func(*openOptions)

type openOptions struct {
	debugPath string
	arch      string
}

// WithDebugPath points the loader at an explicit external debug-symbols file or
// directory (an ELF .debug companion, or a .dSYM bundle / DWARF file for
// Mach-O), tried before the conventional auto-discovered locations.
func WithDebugPath(p string) Option {
	return func(o *openOptions) { o.debugPath = p }
}

// WithArch selects which slice of a universal (fat) Mach-O to load, by name
// (e.g. "x86_64", "arm64"). Empty (the default) picks the host architecture, or
// the first slice. Ignored for thin Mach-O and other formats.
func WithArch(name string) Option {
	return func(o *openOptions) { o.arch = name }
}

// Open reads path, detects its container format, and builds the neutral model.
func Open(path string, opts ...Option) (*File, error) {
	var o openOptions
	for _, opt := range opts {
		opt(&o)
	}
	// mapFile mmaps the file where that's safe (always on Linux; on macOS only
	// when the Mach-O carries no code signature, since mmap'ing a signed binary
	// gets the process SIGKILL'd), otherwise it reads the file into the heap.
	raw, closer, err := mapFile(path)
	if err != nil {
		return nil, err
	}
	f := &File{
		Path:      path,
		debugPath: o.debugPath,
		reqArch:   o.arch,
		raw:       raw,
		unmap:     closer,
		sources:   map[string][]string{},
	}
	switch {
	case len(raw) >= 4 && raw[0] == 0x7f && raw[1] == 'E' && raw[2] == 'L' && raw[3] == 'F':
		if err := f.loadELF(); err != nil {
			f.Close()
			return nil, err
		}
	case isMachO(raw):
		if err := f.loadMachO(); err != nil {
			f.Close()
			return nil, err
		}
	case len(raw) >= 2 && raw[0] == 'M' && raw[1] == 'Z':
		if err := f.loadPE(); err != nil {
			f.Close()
			return nil, err
		}
	default:
		f.Close()
		return nil, fmt.Errorf("unrecognised file format (not ELF, Mach-O, or PE)")
	}

	f.finalizeSymbols()
	f.computeOverview()
	return f, nil
}

// Close releases the file mapping. Safe to call more than once; afterwards the
// raw bytes (and anything slicing into them) must not be used.
func (f *File) Close() error {
	if f == nil || f.unmap == nil {
		return nil
	}
	err := f.unmap()
	f.unmap = nil
	f.raw = nil
	return err
}
