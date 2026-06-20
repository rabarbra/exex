package ui

import (
	"testing"

	"github.com/rabarbra/exex/internal/binfile"
)

func TestSymbolScopeFilter(t *testing.T) {
	m := &Model{
		file: &binfile.File{Symbols: []binfile.Symbol{
			{Name: "my_func", Addr: 0x1000},                      // internal (defined here)
			{Name: "my_data", Addr: 0x2000},                      // internal
			{Name: "malloc", Addr: 0x3000, Library: "libc.so.6"}, // imported (PLT/GOT)
			{Name: "undef", Addr: 0},                             // undefined: neither internal nor imported
		}},
		symbolsState: symbolsState{},
	}
	m.symbolsFilter = newPromptInput("", "/ ")

	count := func(sc symbolScope) int {
		m.symbolsScope = sc
		m.recomputeSymbols()
		return len(m.symbolsFiltered)
	}

	if got := count(scopeAll); got != 4 {
		t.Fatalf("scope all = %d, want 4", got)
	}
	if got := count(scopeInternal); got != 2 {
		t.Fatalf("scope internal = %d, want 2 (defined here only)", got)
	}
	if got := count(scopeImported); got != 1 {
		t.Fatalf("scope imported = %d, want 1 (library-bound only)", got)
	}
}
