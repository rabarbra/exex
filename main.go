// Command exex is a terminal UI for exploring ELF, Mach-O, and PE binaries:
// header, sections, symbols, disassembly, and DWARF-driven source mapping.
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"

	tea "charm.land/bubbletea/v2"

	"github.com/rabarbra/exex/internal/binfile"
	"github.com/rabarbra/exex/internal/config"
	"github.com/rabarbra/exex/internal/ui"
)

func main() {
	var debugPath string
	flag.StringVar(&debugPath, "debug", "", "path to an external debug-symbols file or directory (ELF .debug / Mach-O .dSYM)")
	flag.StringVar(&debugPath, "d", "", "shorthand for -debug")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [-debug PATH] <binary> [goto]\n", os.Args[0])
		fmt.Fprintln(os.Stderr, "  <binary>  path to an ELF/Mach-O/PE file, or a command name on $PATH")
		fmt.Fprintln(os.Stderr, "  goto      optional address (0x…) or symbol name to jump to on open")
		flag.PrintDefaults()
	}
	flag.Parse()

	args := flag.Args()
	if len(args) < 1 || len(args) > 2 {
		flag.Usage()
		os.Exit(2)
	}
	path := resolveTarget(args[0])
	gotoTarget := ""
	if len(args) == 2 {
		gotoTarget = args[1]
	}

	var openOpts []binfile.Option
	if debugPath != "" {
		openOpts = append(openOpts, binfile.WithDebugPath(debugPath))
	}
	f, err := binfile.Open(path, openOpts...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "exex: %v\n", err)
		os.Exit(1)
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "exex: %v\n", err)
		os.Exit(1)
	}

	m, err := ui.New(f, ui.Options{Config: cfg, Goto: gotoTarget})
	if err != nil {
		fmt.Fprintf(os.Stderr, "exex: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(m)
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "exex: %v\n", err)
		os.Exit(1)
	}
}

// resolveTarget turns the CLI argument into a file path. An existing file (or
// any argument containing a path separator) is used as-is; a bare command name
// is looked up on $PATH like a shell would, so "exex ls" opens /bin/ls. When no
// PATH entry matches, the original argument is returned so binfile.Open reports
// the usual not-found error.
func resolveTarget(arg string) string {
	if st, err := os.Stat(arg); err == nil && !st.IsDir() {
		return arg
	}
	if p, err := exec.LookPath(arg); err == nil {
		return p
	}
	return arg
}
