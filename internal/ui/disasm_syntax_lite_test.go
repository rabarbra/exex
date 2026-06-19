//go:build lite

package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/rabarbra/exex/internal/config"
	"github.com/rabarbra/exex/internal/disasm"
)

func TestLiteDisasmMnemonicCategoriesUseThemeStyles(t *testing.T) {
	m := &Model{theme: NewTheme(config.Config{Colors: config.Colors{
		AsmMove:                    "#010203",
		AsmArith:                   "#040506",
		InstructionJumpConditional: "#070809",
	}})}

	move := m.renderInstTextStyled("mov %rsp,%rbp", disasm.Classify("mov %rsp,%rbp"), 0)
	if plain := ansi.Strip(move); plain != "mov %rsp,%rbp" {
		t.Fatalf("move plain text = %q", plain)
	}
	if !strings.Contains(move, m.theme.asmMoveStyle.Render("mov")) {
		t.Fatalf("move mnemonic not styled as move: %q", move)
	}

	arith := m.renderInstTextStyled("add $1,%eax", disasm.Classify("add $1,%eax"), 0)
	if !strings.Contains(arith, m.theme.asmArithStyle.Render("add")) {
		t.Fatalf("arithmetic mnemonic not styled as arithmetic: %q", arith)
	}

	jump := m.renderInstTextStyled("je 0x100", disasm.Classify("je 0x100"), 0)
	if !strings.Contains(jump, m.theme.classJumpCndStyle.Render("je")) {
		t.Fatalf("jump mnemonic not styled as jump: %q", jump)
	}
}
