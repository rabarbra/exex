package ui

import (
	"testing"

	"github.com/rabarbra/exex/internal/config"
)

func TestPresetColorsLookup(t *testing.T) {
	if got := presetColors(""); got.InstructionCall != "" {
		t.Fatalf("empty preset should be zero, got %q", got.InstructionCall)
	}
	if got := presetColors("nope"); got.InstructionCall != "" {
		t.Fatalf("unknown preset should be zero, got %q", got.InstructionCall)
	}
	if got := presetColors("solarized-dark"); got.InstructionCall != solBlue {
		t.Fatalf("solarized-dark call colour = %q, want %q", got.InstructionCall, solBlue)
	}
	if got := presetColors("nord"); got.InstructionCall != nord8 || len(got.HexBytePalette) != 18 {
		t.Fatalf("nord preset incomplete: call=%q palette=%d", got.InstructionCall, len(got.HexBytePalette))
	}
	// Light and dark swap the body/background tones.
	if dark, light := presetColors("solarized-dark"), presetColors("solarized-light"); dark.TableRowFG == light.TableRowFG {
		t.Fatalf("light and dark presets should differ in body text colour (both %q)", dark.TableRowFG)
	}
}

func TestNewThemePresetAndOverridePrecedence(t *testing.T) {
	base := DefaultTheme().classCallStyle.Render("x")

	preset := NewTheme(config.Config{Theme: "solarized-dark"}).classCallStyle.Render("x")
	if preset == base {
		t.Fatal("solarized-dark preset did not change the call-instruction colour")
	}

	// An explicit colour override must win over the preset.
	override := NewTheme(config.Config{
		Theme:  "solarized-dark",
		Colors: config.Colors{InstructionCall: "#ff0000"},
	}).classCallStyle.Render("x")
	if override == preset {
		t.Fatal("colors override did not win over the preset")
	}
}

func TestPathColorKeyGroupsCoarsely(t *testing.T) {
	cases := map[string]string{
		"/usr/lib/clang/foo.h":   "usr/lib",
		"/usr/lib/glib/bar.h":    "usr/lib", // sibling subtree → same key
		"/Users/x/proj/src/a.c":  "Users/x",
		"/Users/x/proj/test/b.c": "Users/x",
		"/opt/thing":             "opt",
		"libfoo.so":              "", // bare name → shared key
	}
	for in, want := range cases {
		if got := pathColorKey(in); got != want {
			t.Errorf("pathColorKey(%q) = %q, want %q", in, got, want)
		}
	}
	// Presets must supply the palettes so path and source-mapping colours follow
	// the theme rather than falling back to the hardcoded defaults.
	for _, name := range []string{"nord", "solarized-dark", "solarized-light"} {
		p := presetColors(name)
		if len(p.PathPalette) == 0 {
			t.Errorf("preset %q has no path palette", name)
		}
		if len(p.ColumnPalette) == 0 {
			t.Errorf("preset %q has no column palette", name)
		}
		if p.SourceCodeLineFG == "" {
			t.Errorf("preset %q has no source-code-line colour", name)
		}
	}
}

func TestSetBytePaletteGuards(t *testing.T) {
	// Restore the default ramp no matter what this test does.
	t.Cleanup(func() { setBytePalette(defaultBytePalette[:]) })

	before := byteHex[0x10]
	setBytePalette([]string{"#fff", "#000"}) // wrong length: ignored
	if byteHex[0x10] != before {
		t.Fatal("short palette should have been ignored")
	}

	custom := make([]string, 18)
	for i := range custom {
		custom[i] = "#abcdef"
	}
	setBytePalette(custom)
	if byteHex[0x10] == before {
		t.Fatal("valid 18-entry palette should have rebuilt the ramp")
	}
}
