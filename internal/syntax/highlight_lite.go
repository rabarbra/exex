//go:build lite

package syntax

// Lite build: no Chroma. Source files are coloured by the tiny built-in
// highlighter (comments / strings / numbers / categorized keywords).
func HighlightLines(filename string, src []string, theme string) []string {
	return minimalHighlight(filename, src, theme)
}
