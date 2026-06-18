// Package syntax highlights source files for display in the TUI source pane.
package syntax

import (
	"strings"
	"sync"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

const defaultTheme = "catppuccin-mocha"

// Highlighter tokenises source files once and caches their per-line ANSI output.
type Highlighter struct {
	theme string
	mu    sync.RWMutex
	cache map[string][]string
}

// NewHighlighter creates a cached source highlighter. An empty theme selects the
// project default.
func NewHighlighter(theme string) *Highlighter {
	if theme == "" {
		theme = defaultTheme
	}
	return &Highlighter{theme: theme, cache: map[string][]string{}}
}

// Highlight returns ANSI-styled source lines for filename, using a per-filename
// cache. A nil receiver falls back to one-shot highlighting with the default
// theme.
func (h *Highlighter) Highlight(filename string, src []string) []string {
	if h == nil {
		return HighlightLines(filename, src, defaultTheme)
	}
	h.mu.RLock()
	v, ok := h.cache[filename]
	h.mu.RUnlock()
	if ok {
		return v
	}
	hl := HighlightLines(filename, src, h.theme)
	h.mu.Lock()
	h.cache[filename] = hl
	h.mu.Unlock()
	return hl
}

// HighlightLines returns ANSI-styled source lines without using a cache. It
// returns nil when Chroma cannot identify a lexer for the file/content.
func HighlightLines(filename string, src []string, theme string) []string {
	lexer := lexers.Match(filename)
	if lexer == nil {
		lexer = lexers.Analyse(strings.Join(src, "\n"))
	}
	if lexer == nil {
		return nil
	}
	lexer = chroma.Coalesce(lexer)

	st := styles.Get(theme)
	if st == nil {
		st = styles.Fallback
	}

	it, err := lexer.Tokenise(nil, strings.Join(src, "\n"))
	if err != nil {
		return nil
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
