package ui

import (
	"testing"

	"github.com/rabarbra/exex/internal/binfile"
	"github.com/rabarbra/exex/internal/config"
)

func TestSettingsCycleAndPersist(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)

	m := &Model{theme: DefaultTheme(), file: &binfile.File{}}

	m.openSettings()
	if !m.settingsActive {
		t.Fatal("openSettings did not activate the popup")
	}

	// Field 0: theme — right cycles off the default.
	m.updateSettings("right")
	if m.cfg.Theme == "" || m.cfg.Theme == defaultThemeName {
		t.Fatalf("theme did not cycle, got %q", m.cfg.Theme)
	}

	// Field 1: background — space toggles it on.
	m.updateSettings("down")
	m.updateSettings(" ")
	if !m.cfg.Behavior.Background {
		t.Fatal("background toggle did not turn on")
	}

	// Field 2: default wrap — space toggles it on and applies it for the session.
	m.updateSettings("down")
	m.updateSettings(" ")
	if !m.cfg.Behavior.DefaultWrap || !m.wrap {
		t.Fatal("default wrap toggle did not turn on")
	}

	// Field 3: default view — right cycles to a non-empty view name.
	m.updateSettings("down")
	m.updateSettings("right")
	if m.cfg.Behavior.DefaultView == "" {
		t.Fatal("default view did not cycle")
	}

	// Enter persists and closes.
	m.updateSettings("enter")
	if m.settingsActive {
		t.Fatal("Enter should close the popup")
	}
	c, err := config.Load()
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if c.Theme != m.cfg.Theme || !c.Behavior.Background || !c.Behavior.DefaultWrap || c.Behavior.DefaultView != m.cfg.Behavior.DefaultView {
		t.Fatalf("persisted config mismatch: %+v", c)
	}
}
