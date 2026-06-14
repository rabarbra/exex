// Command exex is a terminal UI for exploring ELF, Mach-O, and PE binaries:
// header, sections, symbols, disassembly, and DWARF-driven source mapping.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/rabarbra/exex/internal/binfile"
	"github.com/rabarbra/exex/internal/ui"
)

func main() {
	if len(os.Args) != 2 || os.Args[1] == "-h" || os.Args[1] == "--help" {
		fmt.Fprintf(os.Stderr, "usage: %s <binary>   (ELF, Mach-O, or PE)\n", os.Args[0])
		os.Exit(2)
	}
	path := os.Args[1]

	f, err := binfile.Open(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "exex: %v\n", err)
		os.Exit(1)
	}

	m, err := ui.New(f)
	if err != nil {
		fmt.Fprintf(os.Stderr, "exex: %v\n", err)
		os.Exit(1)
	}

	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "exex: %v\n", err)
		os.Exit(1)
	}
}
