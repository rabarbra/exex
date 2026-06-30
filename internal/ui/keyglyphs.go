package ui

// Key labels for the footer, help overlay and in-view facet buttons. The facet
// filters use Ctrl chords (the same on macOS and Linux) — Alt was dropped because
// gnome-terminal binds Alt+letter to its menubar mnemonics, swallowing the keys.

import "strings"

// ctrlKeys renders a Ctrl-chord list as "^t" / "^t/^f": each key gets a caret,
// joined with "/". Compact and identical on every platform.
func ctrlKeys(keys ...string) string {
	parts := make([]string, len(keys))
	for i, k := range keys {
		parts[i] = "^" + k
	}
	return strings.Join(parts, "/")
}
