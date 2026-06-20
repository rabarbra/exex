package ui

import (
	"reflect"
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

func TestSymbolSortAndBind(t *testing.T) {
	m := &Model{file: &binfile.File{Symbols: []binfile.Symbol{
		{Name: "a", Addr: 0x1000, Size: 10, Bind: binfile.BindLocal},
		{Name: "b", Addr: 0x3000, Size: 50, Bind: binfile.BindGlobal},
		{Name: "c", Addr: 0x2000, Size: 30, Bind: binfile.BindGlobal},
	}}}
	m.symbolsFilter = newPromptInput("", "/ ")

	addrs := func() []uint64 {
		m.recomputeSymbols()
		out := make([]uint64, 0, len(m.symbolsFiltered))
		for _, idx := range m.symbolsFiltered {
			out = append(out, m.file.Symbols[idx].Addr)
		}
		return out
	}

	m.symbolsSort = sortByAddr
	if got := addrs(); !reflect.DeepEqual(got, []uint64{0x1000, 0x2000, 0x3000}) {
		t.Fatalf("sort by addr = %#x", got)
	}
	m.symbolsSort = sortBySize // descending by size: b(50) c(30) a(10)
	if got := addrs(); !reflect.DeepEqual(got, []uint64{0x3000, 0x2000, 0x1000}) {
		t.Fatalf("sort by size = %#x", got)
	}

	m.symbolsSort = sortByName
	m.symbolsBindOn = true
	m.symbolsBind = binfile.BindGlobal
	if got := len(addrs()); got != 2 {
		t.Fatalf("bind=global count = %d, want 2", got)
	}
}
