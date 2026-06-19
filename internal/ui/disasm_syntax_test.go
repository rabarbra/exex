//go:build !lite

package ui

import (
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/alecthomas/chroma/v2"
)

func TestDisasmChromaDefaultTokenUsesSyntaxForeground(t *testing.T) {
	got := chromaStyleEntryToLipgloss(chroma.StyleEntry{}, "#586e75").Render("x")
	want := lipgloss.NewStyle().Foreground(lipgloss.Color("#586e75")).Render("x")
	if got != want {
		t.Fatalf("default disasm token style = %q, want %q", got, want)
	}
}
