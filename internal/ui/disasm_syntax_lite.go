//go:build lite

package ui

// Lite build: a small, theme-driven assembly highlighter that replaces Chroma.
// It colours the mnemonic by instruction class, followable mapped addresses by
// their link colour, and operand registers / immediates by the theme's
// (configurable) asm token colours — so it still follows the active preset and
// any `colors:` overrides, without Chroma's embedded lexer data.

import (
	"strings"

	"github.com/rabarbra/exex/internal/disasm"
)

// asmOperandKeywords are operand-position size/scope specifiers that read better
// left uncoloured than tinted as registers.
var asmOperandKeywords = map[string]bool{
	"ptr": true, "byte": true, "word": true, "dword": true, "qword": true,
	"tword": true, "oword": true, "xmmword": true, "ymmword": true, "zmmword": true,
	"near": true, "far": true, "short": true,
}

func (m *Model) renderInstTextStyled(text string, class disasm.InstClass, instAddr uint64) string {
	spans := m.disasmAddrSpans(text, instAddr)
	var b strings.Builder
	n := len(text)

	// Mnemonic: the leading non-space run, coloured by instruction class or by
	// the built-in lite category when this is otherwise a plain instruction.
	i := 0
	for i < n && (text[i] == ' ' || text[i] == '\t') {
		b.WriteByte(text[i])
		i++
	}
	mnStart := i
	for i < n && text[i] != ' ' && text[i] != '\t' {
		i++
	}
	if mnStart < i {
		mnemonic := text[mnStart:i]
		b.WriteString(m.theme.styleForClass(class).Render(mnemonic))
	}

	for i < n {
		if sp, ok := disasmSpanAt(spans, i); ok {
			b.WriteString(sp.style.Render(text[i:sp.end]))
			i = sp.end
			continue
		}
		c := text[i]
		switch {
		case c == ' ' || c == '\t':
			j := i + 1
			for j < n && (text[j] == ' ' || text[j] == '\t') {
				j++
			}
			b.WriteString(text[i:j])
			i = j
		case c == '%': // AT&T register (%rax, %xmm0)
			j := i + 1
			for j < n && isAsmIdentChar(text[j]) {
				j++
			}
			b.WriteString(m.theme.asmRegisterStyle.Render(text[i:j]))
			i = j
		case c == '$' || c == '#': // immediate prefix (AT&T $, ARM #)
			j := i + 1
			for j < n && isAsmNumChar(text[j]) {
				j++
			}
			b.WriteString(m.theme.asmNumberStyle.Render(text[i:j]))
			i = j
		case c >= '0' && c <= '9':
			j := i + 1
			for j < n && isAsmNumChar(text[j]) {
				j++
			}
			b.WriteString(m.theme.asmNumberStyle.Render(text[i:j]))
			i = j
		case isAsmIdentStart(c):
			j := i + 1
			for j < n && isAsmIdentChar(text[j]) {
				j++
			}
			tok := text[i:j]
			if asmOperandKeywords[strings.ToLower(tok)] {
				b.WriteString(tok)
			} else {
				b.WriteString(m.theme.asmRegisterStyle.Render(tok))
			}
			i = j
		default: // punctuation: [], (), commas, +, -, *, : …
			b.WriteByte(c)
			i++
		}
	}
	return b.String()
}

// disasmSpanAt returns the address span covering byte index i, if any.
func disasmSpanAt(spans []disasmAddrSpan, i int) (disasmAddrSpan, bool) {
	for _, s := range spans {
		if i >= s.start && i < s.end {
			return s, true
		}
	}
	return disasmAddrSpan{}, false
}

func isAsmIdentStart(c byte) bool { return c == '_' || (c|0x20 >= 'a' && c|0x20 <= 'z') }
func isAsmIdentChar(c byte) bool {
	return isAsmIdentStart(c) || (c >= '0' && c <= '9') || c == '.'
}
func isAsmNumChar(c byte) bool {
	return (c >= '0' && c <= '9') || c == 'x' || c == 'X' || c == '.' || c == '_' ||
		(c|0x20 >= 'a' && c|0x20 <= 'f')
}
