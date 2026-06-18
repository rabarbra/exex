package config

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestPathHonorsXDGConfigHome(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)

	want := filepath.Join(root, "exex", "config.yaml")
	if got := Path(); got != want {
		t.Fatalf("Path = %q, want %q", got, want)
	}
}

func TestPathFallsBackToHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", home)

	want := filepath.Join(home, ".config", "exex", "config.yaml")
	if got := Path(); got != want {
		t.Fatalf("Path = %q, want %q", got, want)
	}
}

func TestLoadMissingConfigReturnsDefault(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load missing config: %v", err)
	}
	if cfg == nil || cfg.Behavior.DefaultView != "" || len(cfg.Keys.Quit) != 0 {
		t.Fatalf("Load missing config = %#v, want zero defaults", cfg)
	}
}

func TestLoadParsesConfig(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	dir := filepath.Join(root, "exex")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	data := []byte(`colors:
  syntax_theme: dracula
keys:
  quit: [q, ctrl+c]
  goto: g
behavior:
  default_view: disasm
  disasm_max_bytes: 4096
`)
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Colors.SyntaxTheme != "dracula" {
		t.Fatalf("syntax theme = %q, want dracula", cfg.Colors.SyntaxTheme)
	}
	if got := []string(cfg.Keys.Quit); len(got) != 2 || got[0] != "q" || got[1] != "ctrl+c" {
		t.Fatalf("quit keys = %#v", got)
	}
	if got := []string(cfg.Keys.Goto); len(got) != 1 || got[0] != "g" {
		t.Fatalf("goto keys = %#v", got)
	}
	if cfg.Behavior.DefaultView != "disasm" || cfg.Behavior.DisasmMaxBytes != 4096 {
		t.Fatalf("behavior = %#v", cfg.Behavior)
	}
}

func TestLoadRejectsMalformedConfig(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	dir := filepath.Join(root, "exex")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("keys: ["), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("Load malformed config succeeded, want error")
	}
}

func TestStringOrSliceRejectsUnsupportedNode(t *testing.T) {
	var cfg Config
	if err := yaml.Unmarshal([]byte("keys:\n  quit:\n    nested: value\n"), &cfg); err == nil {
		t.Fatal("yaml.Unmarshal map key succeeded, want error")
	}
}
