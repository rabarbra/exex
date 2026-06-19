//go:build !lite

package syntax

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
)

func TestHighlightLines(t *testing.T) {

	src := []string{
		"#include <stdio.h>",
		"int main(void) {",
		"    printf(\"hi\\n\");",
		"    return 0;",
		"}",
	}
	hl := HighlightLines("main.c", src, defaultTheme)
	if hl == nil {
		t.Fatal("no lexer matched main.c")
	}
	if len(hl) != len(src) {
		t.Fatalf("highlighted line count = %d, want %d", len(hl), len(src))
	}
	for i := range src {
		if got := stripANSI(hl[i]); got != src[i] {
			t.Fatalf("line %d plain text = %q, want %q", i, got, src[i])
		}
	}
	if !strings.Contains(strings.Join(hl, ""), "\x1b[") {
		t.Fatal("expected ANSI colour codes in highlighted output")
	}
}

func TestHighlightUnknownExtension(t *testing.T) {
	hl := HighlightLines("data.unknownext", []string{"\x00\x01\x02"}, defaultTheme)
	for _, line := range hl {
		_ = line
	}
}

func TestHighlighterNilReceiverAndInvalidTheme(t *testing.T) {
	src := []string{"package main", "func main() {}"}
	var h *Highlighter
	if got := h.Highlight("main.go", src); len(got) != len(src) {
		t.Fatalf("nil highlighter line count = %d, want %d", len(got), len(src))
	}
	got := HighlightLines("main.go", src, "definitely-not-a-theme")
	if len(got) != len(src) {
		t.Fatalf("invalid theme line count = %d, want %d", len(got), len(src))
	}
	for i := range src {
		if plain := stripANSI(got[i]); plain != src[i] {
			t.Fatalf("line %d plain text = %q, want %q", i, plain, src[i])
		}
	}
}

func TestHighlighterCachesByFilename(t *testing.T) {
	h := NewHighlighter("")
	first := h.Highlight("main.go", []string{"package main"})
	second := h.Highlight("main.go", []string{"package changed"})
	if len(first) != len(second) || stripANSI(second[0]) != "package main" {
		t.Fatalf("cached highlight = %q, want first source", second)
	}
}

func TestChromaDefaultTokenUsesThemeForeground(t *testing.T) {
	got := chromaToLipgloss(chroma.StyleEntry{}, "#586e75").Render("x")
	want := lipgloss.NewStyle().Foreground(lipgloss.Color("#586e75")).Render("x")
	if got != want {
		t.Fatalf("default token style = %q, want %q", got, want)
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
				j++
			}
			if j < len(s) {
				j++
			}
			i = j - 1
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}
