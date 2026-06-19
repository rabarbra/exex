//go:build !lite

package ui

// Default build: Chroma-based assembly syntax highlighting. The `lite` build
// (disasm_syntax_lite.go) swaps in a small theme-driven token highlighter and
// drops Chroma's ~3 MB of embedded lexer/style data.

import (
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"

	"github.com/rabarbra/exex/internal/arch"
	"github.com/rabarbra/exex/internal/disasm"
)

var disasmAsmLexer chroma.Lexer = nil

func newDisasmAsmLexer(arch arch.Arch) chroma.Lexer {
	lexer_names := []string{"ArmAsm", "GAS", "asm", "NASM"}
	switch arch {
	case disasm.ArchX86, disasm.ArchAMD64, disasm.ArchRISCV64:
		lexer_names = append([]string{"GAS"}, lexer_names...)
	case disasm.ArchARM64:
		lexer_names = append([]string{"ArmAsm"}, lexer_names...)
	}
	for _, name := range lexer_names {
		if lexer := lexers.Get(name); lexer != nil {
			return chroma.Coalesce(lexer)
		}
	}
	return nil
}

// renderInstTextStyled uses Chroma for assembly syntax, overlaying semantic link
// styles on followable address literals.
func (m *Model) renderInstTextStyled(text string, class disasm.InstClass, instAddr uint64) string {
	if disasmAsmLexer == nil {
		disasmAsmLexer = newDisasmAsmLexer(m.file.Arch())
	}
	if disasmAsmLexer == nil {
		return m.renderInstTextFallback(text, class, instAddr)
	}
	tokens, err := chroma.Tokenise(disasmAsmLexer, nil, text)
	if err != nil {
		return m.renderInstTextFallback(text, class, instAddr)
	}
	spans := m.disasmAddrSpans(text, instAddr)
	pos := 0
	var b strings.Builder
	for _, tok := range tokens {
		if tok == chroma.EOF {
			break
		}
		b.WriteString(m.renderDisasmToken(tok, pos, spans))
		pos += len(tok.Value)
	}
	return b.String()
}

func (m *Model) renderDisasmToken(tok chroma.Token, pos int, spans []disasmAddrSpan) string {
	st := m.disasmTokenStyle(tok.Type)
	from := 0
	var b strings.Builder
	for _, span := range spans {
		lo := max(span.start, pos)
		hi := min(span.end, pos+len(tok.Value))
		if hi <= lo {
			continue
		}
		if rel := lo - pos; rel > from {
			b.WriteString(st.Render(tok.Value[from:rel]))
		}
		b.WriteString(span.style.Render(tok.Value[lo-pos : hi-pos]))
		from = hi - pos
	}
	if from < len(tok.Value) {
		b.WriteString(st.Render(tok.Value[from:]))
	}
	return b.String()
}

func (m *Model) disasmTokenStyle(tt chroma.TokenType) lipgloss.Style {
	if m.disasmTokenStyles == nil {
		m.disasmTokenStyles = make(map[int]lipgloss.Style)
	}
	if st, ok := m.disasmTokenStyles[int(tt)]; ok {
		return st
	}
	theme := sourceSyntaxTheme(m.cfg)
	chromaStyle := styles.Get(theme)
	if chromaStyle == nil {
		chromaStyle = styles.Fallback
	}
	st := chromaStyleEntryToLipgloss(chromaStyle.Get(tt))
	m.disasmTokenStyles[int(tt)] = st
	return st
}

func chromaStyleEntryToLipgloss(e chroma.StyleEntry) lipgloss.Style {
	st := lipgloss.NewStyle()
	if e.Colour.IsSet() {
		st = st.Foreground(lipgloss.Color(e.Colour.String()))
	}
	if e.Bold == chroma.Yes {
		st = st.Bold(true)
	}
	if e.Italic == chroma.Yes {
		st = st.Italic(true)
	}
	if e.Underline == chroma.Yes {
		st = st.Underline(true)
	}
	return st
}
