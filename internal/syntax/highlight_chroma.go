//go:build !lite

package syntax

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// HighlightLines returns ANSI-styled source lines without using a cache. It uses
// the minimal highlighter when Chroma cannot identify or tokenise the file.
func HighlightLines(filename string, src []string, theme string) []string {
	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Analyse(strings.Join(src, "\n"))
	}
	if lexer == nil {
		// Unknown file type: fall back to the tiny built-in highlighter rather
		// than rendering plain text.
		return minimalHighlight(filename, src, theme)
	}
	lexer = chroma.Coalesce(lexer)

	st := styles.Get(theme)
	if st == nil {
		st = styles.Fallback
	}

	it, err := lexer.Tokenise(nil, strings.Join(src, "\n"))
	if err != nil {
		return minimalHighlight(filename, src, theme)
	}

	// Memoise the lipgloss style per token type: a source file has thousands of
	// tokens but only a handful of distinct types.
	styleFor := map[chroma.TokenType]lipgloss.Style{}
	lines := make([]string, 0, len(src))
	var cur strings.Builder
	for _, tok := range it.Tokens() {
		ls, ok := styleFor[tok.Type]
		if !ok {
			ls = chromaToLipgloss(st.Get(tok.Type))
			styleFor[tok.Type] = ls
		}
		parts := strings.Split(tok.Value, "\n")
		for i, p := range parts {
			if i > 0 {
				lines = append(lines, cur.String())
				cur.Reset()
			}
			if p != "" {
				cur.WriteString(ls.Render(p))
			}
		}
	}
	lines = append(lines, cur.String())
	return lines
}

// chromaToLipgloss converts the subset of Chroma style attributes used here.
func chromaToLipgloss(e chroma.StyleEntry) lipgloss.Style {
	s := lipgloss.NewStyle()
	if e.Colour.IsSet() {
		s = s.Foreground(lipgloss.Color(e.Colour.String()))
	}
	if e.Bold == chroma.Yes {
		s = s.Bold(true)
	}
	if e.Italic == chroma.Yes {
		s = s.Italic(true)
	}
	if e.Underline == chroma.Yes {
		s = s.Underline(true)
	}
	return s
}
