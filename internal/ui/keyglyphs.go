package ui

// OS-aware key labels for the footer, help overlay and in-view facet buttons.
// The key bindings themselves are platform-independent (the dispatch maps both
// the macOS Option chords and Linux Alt chords to the same actions — see
// key_dispatch.go); only the displayed glyph differs so each platform reads
// natively. exex runs on macOS and Linux alike.

import (
	"runtime"
	"strings"
)

// altKeys renders an Alt/Option chord list: the macOS Option glyph on darwin
// ("⌥t/f") and the word "Alt+" elsewhere ("Alt+t/f"). One or more keys share a
// single modifier prefix and are joined with "/".
func altKeys(keys ...string) string {
	joined := strings.Join(keys, "/")
	if runtime.GOOS == "darwin" {
		return "⌥" + joined
	}
	return "Alt+" + joined
}
