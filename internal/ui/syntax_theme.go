package ui

import (
	"strings"

	"github.com/rabarbra/exex/internal/config"
)

const defaultSyntaxTheme = "catppuccin-mocha"

func sourceSyntaxTheme(cfg config.Config) string {
	if theme := strings.TrimSpace(cfg.Colors.SyntaxTheme); theme != "" {
		return theme
	}
	theme := strings.ToLower(strings.TrimSpace(cfg.Theme))
	switch theme {
	case "nord", "solarized-dark", "solarized-light":
		return theme
	}
	return defaultSyntaxTheme
}
