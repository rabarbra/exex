package ui

// This file owns the info / overview view: the file header re-aligned into one
// column, plus overview, hardening (checksec-style), dynamic-linking, and
// toolchain blocks. The Entry line is actionable — Enter follows it into the
// disassembly. The whole page scrolls through headerVP.

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/rabarbra/exex/internal/binfile"
)

func (m *Model) updateInfo(msg tea.KeyMsg, key string) (tea.Model, tea.Cmd) {
	switch key {
	case "home":
		m.headerVP.GotoTop()
		return m, nil
	case "end", "G":
		m.headerVP.GotoBottom()
		return m, nil
	case "enter":
		if m.dis != nil && m.file.Entry() != 0 {
			m.loadDisasmAt(m.file.Entry())
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.headerVP, cmd = m.headerVP.Update(msg)
	return m, cmd
}

func (m *Model) renderInfo() string {
	bodyH := m.bodyHeight()
	innerW := max(1, m.width-4) // panel border (2) + padding (2)

	var b strings.Builder
	first := true
	bodyText := func(s string) string {
		return renderStyle(s, 0, m.theme.tableRowStyle)
	}
	kv := func(k, v string) {
		b.WriteString(m.theme.headerKey.Render(padKey(k, 16)))
		b.WriteString(" ")
		b.WriteString(bodyText(v))
		b.WriteString("\n")
	}
	// head opens a labelled group with a titled separator filled to the panel
	// width; a blank line precedes every group except the first.
	head := func(title string) {
		if !first {
			b.WriteString("\n")
		}
		first = false
		label := "── " + title + " "
		if fill := innerW - lipgloss.Width(label); fill > 0 {
			label += strings.Repeat("─", fill)
		}
		b.WriteString(m.theme.helpHeadStyle.Render(label))
		b.WriteString("\n")
	}

	// Identity (from the format header). The Entry line is special: it carries
	// the entry symbol and is actionable (Enter follows it).
	head("Identity")
	for _, l := range m.file.HeaderInfo() {
		if strings.HasPrefix(l, "Entry:") {
			kv("Entry:", m.entryValue())
			continue
		}
		if idx := strings.IndexByte(l, ':'); idx >= 0 {
			kv(l[:idx+1], strings.TrimSpace(l[idx+1:]))
		} else {
			b.WriteString(bodyText(l))
			b.WriteString("\n")
		}
	}
	if m.dis != nil {
		kv("Disassembler:", m.dis.Name())
	}

	info := m.file.Info
	if info != nil {
		// Overview.
		head("Overview")
		kv("File size:", fmt.Sprintf("%s  (%d bytes)", humanBytes(info.FileSize), info.FileSize))
		if info.MappedHi > info.MappedLo {
			kv("Mapped range:", fmt.Sprintf("0x%x – 0x%x  (%s)", info.MappedLo, info.MappedHi, humanBytes(info.MappedHi-info.MappedLo)))
		}
		if info.CodeSize > 0 {
			codeLine := humanBytes(info.CodeSize)
			if info.FileSize > 0 {
				codeLine += fmt.Sprintf("  (%.0f%% of file)", 100*float64(info.CodeSize)/float64(info.FileSize))
			}
			kv("Code size:", codeLine)
		}
		if info.WordBits != 0 {
			kv("Word size:", fmt.Sprintf("%d-bit, %s", info.WordBits, info.ByteOrder))
		}
		if info.Segments > 0 {
			kv(segmentLabel(m.file.Format)+":", fmt.Sprintf("%d", info.Segments))
		}

		// Hardening — coloured by how safe each setting is (green = hardened).
		head("Hardening")
		kv("PIE:", m.triSec(info.PIE))
		kv("NX stack:", m.triSec(info.NX))
		if info.RELRO != "" {
			kv("RELRO:", m.relroSec(info.RELRO))
		}
		kv("Stack canary:", m.boolSec(info.Canary, true))
		kv("FORTIFY:", m.boolSec(info.Fortify, true))
		if m.file.Format == binfile.FormatMachO {
			kv("Code signature:", m.boolSec(info.CodeSigned, true))
			if info.Encrypted {
				kv("Encrypted:", m.theme.warnStyle.Render("yes"))
			}
		}

		// Dynamic linking.
		head("Dynamic linking")
		if info.Interp != "" {
			kv("Interpreter:", info.Interp)
		}
		if info.SoName != "" {
			kv("SONAME:", info.SoName)
		}
		if len(info.RPath) > 0 {
			kv("RPATH:", strings.Join(info.RPath, ":"))
		}
		if len(info.RunPath) > 0 {
			kv("RUNPATH:", strings.Join(info.RunPath, ":"))
		}
		if info.BuildID != "" {
			kv("Build ID:", info.BuildID)
		}
		kv("Stripped:", yesNo(info.Stripped))
		kv("Static-linked:", yesNo(info.StaticLinked))
		if info.Libc.Kind != "" {
			val := info.Libc.Kind
			if info.Libc.Version != "" {
				val += " " + info.Libc.Version
			}
			if info.Libc.Source != "" {
				val += "  " + m.theme.footerStyle.Render("("+info.Libc.Source+")")
			}
			kv("Libc:", val)
		}
		if len(info.DynamicLibs) > 0 {
			kv("Needed libs:", fmt.Sprintf("%d (press 6 to view)", len(info.DynamicLibs)))
		}

		// Toolchain / provenance.
		if info.SourceLang != "" || info.Compiler != "" || info.GoVersion != "" || info.MinOS != "" {
			head("Toolchain")
			if info.SourceLang != "" {
				kv("Language:", info.SourceLang)
			}
			// For Go binaries the toolchain is shown as "Go:" below; a stray
			// clang banner from cgo/deps would only mislead.
			if info.Compiler != "" && info.GoVersion == "" {
				kv("Compiler:", info.Compiler)
			}
			if info.GoVersion != "" {
				kv("Go:", info.GoVersion)
			}
			if info.GoModule != "" {
				kv("Go module:", info.GoModule)
			}
			if info.GoVCS != "" {
				kv("VCS:", info.GoVCS)
			}
			if info.MinOS != "" {
				v := info.MinOS
				if info.SDK != "" {
					v += "  (SDK " + info.SDK + ")"
				}
				kv("Min OS:", v)
			}
		}
	}

	// Drop the single-column content into a full-width bordered panel. A long
	// page scrolls inside the panel via the viewport; the border rows (2) leave
	// bodyH-2 rows of content. Pad every line so the panel's right edge is flush.
	lines := strings.Split(strings.TrimRight(b.String(), "\n"), "\n")
	for i := range lines {
		lines[i] = padRight(lines[i], innerW)
	}

	m.headerVP.SetWidth(innerW)
	m.headerVP.SetHeight(max(1, bodyH-2))
	m.headerVP.SetContent(strings.Join(lines, "\n"))

	panel := m.theme.panelStyle.Render(m.headerVP.View())
	return lipgloss.Place(m.width, bodyH, lipgloss.Center, lipgloss.Top, panel)
}

// entryValue renders the entry point value: its address, the entry symbol, and
// a hint that Enter follows it into the disassembly.
func (m *Model) entryValue() string {
	entry := m.file.Entry()
	val := fmt.Sprintf("0x%0*x", m.file.AddrHexWidth(), entry)
	if sym, ok := m.file.SymbolAt(entry); ok {
		name := sym.Display()
		if off := entry - sym.Addr; off != 0 {
			name = fmt.Sprintf("%s+0x%x", name, off)
		}
		val += "  " + m.theme.symbolNameStyle.Render(name)
	}
	if m.dis != nil && entry != 0 {
		val += "  " + m.theme.footerStyle.Render("↵ disassemble")
	}
	return val
}

// boolSec renders a yes/no hardening flag green when it equals the hardened
// value and red otherwise.
func (m *Model) boolSec(v, hardenedWhenYes bool) string {
	s := yesNo(v)
	if v == hardenedWhenYes {
		return m.theme.infoStyle.Render(s)
	}
	return m.theme.errorStyle.Render(s)
}

// triSec colours a tri-state hardening flag: enabled (hardened) green, disabled
// red, unknown dim.
func (m *Model) triSec(t binfile.Tristate) string {
	switch t {
	case binfile.TriYes:
		return m.theme.infoStyle.Render(t.String())
	case binfile.TriNo:
		return m.theme.errorStyle.Render(t.String())
	}
	return m.theme.srcShadowStyle.Render(t.String())
}

// relroSec colours RELRO: full = green, partial = yellow, none = red.
func (m *Model) relroSec(s string) string {
	switch s {
	case "full":
		return m.theme.infoStyle.Render(s)
	case "partial":
		return m.theme.warnStyle.Render(s)
	default:
		return m.theme.errorStyle.Render(s)
	}
}

func segmentLabel(f binfile.Format) string {
	switch f {
	case binfile.FormatMachO:
		return "Load commands"
	case binfile.FormatELF:
		return "Program headers"
	}
	return "Segments"
}

func yesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}

// humanBytes formats a byte count with a binary unit suffix.
func humanBytes(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := uint64(unit), 0
	for x := n / unit; x >= unit; x /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

// padKey right-pads a key label to a fixed column, ignoring the trailing colon
// for alignment purposes.
func padKey(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}
