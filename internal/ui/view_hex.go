package ui

// This file owns the hex view: opening a section into the viewer, navigating
// byte-by-byte, and rendering the offset | hex | ascii table.

import (
	"debug/elf"
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

const bytesPerHexRow = 16

// hexLoadCap caps how many bytes the hex viewer loads at once. Plenty for a
// terminal-sized window; bigger objects can be re-explored via goto.
const hexLoadCap = 64 * 1024

// openHexForSection switches into the hex viewer focused on the given section.
// Non-allocated sections still have file data; we read it via Data().
func (m *Model) openHexForSection(sec *elf.Section) {
	data, err := m.file.SectionData(sec, hexLoadCap)
	if err != nil {
		m.setStatus(err.Error(), true)
		return
	}
	if len(data) == 0 {
		m.setStatus(fmt.Sprintf("section %s has no data", sec.Name), true)
		return
	}
	m.hexBase = sec.Addr
	m.hexData = data
	m.hexSection = sec
	m.hexCur = 0
	m.hexTop = 0
	m.mode = modeHex
	m.status = ""
}

// openHexAt opens the hex viewer at a virtual address with up to size bytes
// (0 = use the hexLoadCap). Used to view data symbols (OBJECT/TLS/COMMON) at
// their actual bytes inside the containing section.
func (m *Model) openHexAt(addr, size uint64) {
	n := int(size)
	if n <= 0 || n > hexLoadCap {
		n = hexLoadCap
	}
	data, err := m.file.ReadAt(addr, n)
	if err != nil {
		m.setStatus(err.Error(), true)
		return
	}
	m.hexBase = addr
	m.hexData = data
	m.hexSection = m.file.SectionAt(addr)
	m.hexCur = 0
	m.hexTop = 0
	m.mode = modeHex
	m.status = ""
}

func (m *Model) updateHex(key string) (tea.Model, tea.Cmd) {
	if len(m.hexData) == 0 {
		return m, nil
	}
	row := bytesPerHexRow
	switch key {
	case "left", "h":
		if m.hexCur > 0 {
			m.hexCur--
		}
	case "right", "l":
		if m.hexCur < len(m.hexData)-1 {
			m.hexCur++
		}
	case "up", "k":
		if m.hexCur >= row {
			m.hexCur -= row
		}
	case "down", "j":
		if m.hexCur+row < len(m.hexData) {
			m.hexCur += row
		}
	case "pgup":
		m.hexCur = max(0, m.hexCur-row*m.bodyHeight())
	case "pgdown":
		m.hexCur = min(len(m.hexData)-1, m.hexCur+row*m.bodyHeight())
	case "home":
		m.hexCur = 0
	case "end", "G":
		m.hexCur = len(m.hexData) - 1
	case "a":
		addr := m.hexBase + uint64(m.hexCur)
		m.copyToClipboard(fmt.Sprintf("0x%0*x", m.file.AddrHexWidth(), addr), "address")
	case "s":
		addr := m.hexBase + uint64(m.hexCur)
		if sym, ok := m.file.SymbolAt(addr); ok {
			m.copyToClipboard(sym.Name, "symbol")
		} else {
			m.setStatus("no symbol at this address", true)
		}
	}
	return m, nil
}

// renderHex draws a classic offset | hex | ascii hex dump for the currently
// loaded byte window.
func (m *Model) renderHex() string {
	bodyH := m.bodyHeight()
	if len(m.hexData) == 0 {
		return padBody("press 2, pick a non-executable section, and hit Enter to view bytes\n", m.width, bodyH)
	}

	row := bytesPerHexRow
	addrW := m.file.AddrHexWidth()
	curRow := m.hexCur / row
	visible := bodyH - 1
	if visible < 1 {
		visible = 1
	}
	topRow := m.hexTop / row
	if curRow < topRow {
		topRow = curRow
	} else if curRow >= topRow+visible {
		topRow = curRow - visible + 1
	}
	m.hexTop = topRow * row

	banner := ""
	if m.hexSection != nil {
		banner = fmt.Sprintf(" %s   base 0x%0*x   shown %d / %d bytes",
			m.hexSection.Name, addrW, m.hexSection.Addr, len(m.hexData), m.hexSection.Size)
	} else {
		banner = fmt.Sprintf(" base 0x%0*x   %d bytes", addrW, m.hexBase, len(m.hexData))
	}
	out := stickySymStyle.Render(padRight(banner, m.width)) + "\n"

	end := m.hexTop + visible*row
	if end > len(m.hexData) {
		end = len(m.hexData)
	}
	for off := m.hexTop; off < end; off += row {
		out += m.renderHexRow(off, addrW) + "\n"
	}
	return padBody(out, m.width, bodyH)
}

func (m *Model) renderHexRow(off, addrW int) string {
	row := bytesPerHexRow
	end := off + row
	if end > len(m.hexData) {
		end = len(m.hexData)
	}
	addr := m.hexBase + uint64(off)
	var hexCol, asciiCol strings.Builder
	for i := off; i < off+row; i++ {
		if i > off {
			hexCol.WriteByte(' ')
			if i == off+row/2 {
				hexCol.WriteByte(' ')
			}
		}
		if i >= end {
			hexCol.WriteString("  ")
			asciiCol.WriteByte(' ')
			continue
		}
		b := m.hexData[i]
		ascii := byte('.')
		if b >= 0x20 && b < 0x7f {
			ascii = b
		}
		if i == m.hexCur {
			hexCol.WriteString(tableSelStyle.Render(hex2(b)))
			asciiCol.WriteString(tableSelStyle.Render(string(ascii)))
		} else {
			hexCol.WriteString(byteHex[b])
			asciiCol.WriteString(byteFG[b].Render(string(ascii)))
		}
	}
	line := fmt.Sprintf(" %s  %s  %s",
		addrStyle.Render(fmt.Sprintf("0x%0*x", addrW, addr)),
		hexCol.String(),
		"|"+asciiCol.String()+"|",
	)
	return padRight(line, m.width)
}
