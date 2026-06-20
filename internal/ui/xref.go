package ui

// Cross-references: from the disasm cursor, find every instruction in the
// executable image whose resolved target address is the one under the cursor —
// callers of a function, branches to a label, code that loads a global's
// address. The scan runs off the UI goroutine (a large image can take a moment)
// and its results open a jump-to modal. Only *direct* references are found: the
// target has to appear as a resolved literal in the instruction text, so
// indirect calls through a register or GOT slot aren't covered (the GOT slot's
// own symbol name partly bridges that gap).

import (
	"fmt"
	"sort"
	"strings"

	tea "charm.land/bubbletea/v2"
)

const (
	xrefVisible = 12  // rows shown in the modal at once
	xrefMaxHits = 500 // cap on collected references
)

// xrefHit is one referencing instruction.
type xrefHit struct {
	addr uint64 // address of the instruction making the reference
	text string // its (trimmed) assembly text
	sym  string // display name of the symbol it lives in, or ""
}

// xrefState holds the cross-reference scan + modal state.
type xrefState struct {
	xrefActive  bool // results modal open
	xrefRunning bool // background scan in flight
	xrefSeq     int  // guards against stale async results
	xrefTarget  uint64
	xrefLabel   string // display name of the target (symbol or 0x…)
	xrefResults []xrefHit
	xrefSel     int
	xrefTop     int
}

// xrefDoneMsg delivers a finished cross-reference scan.
type xrefDoneMsg struct {
	seq    int
	target uint64
	hits   []xrefHit
}

// startXrefScan launches a cross-reference scan for the address under the disasm
// cursor (a symbol start finds its callers; any other address finds branches and
// loads that target it).
func (m *Model) startXrefScan() tea.Cmd {
	if m.dis == nil || len(m.disasmInst) == 0 {
		m.setStatus("no disassembly to cross-reference", true)
		return nil
	}
	target := m.disasmInst[m.disasmCur].Addr
	label := fmt.Sprintf("0x%x", target)
	if sym, ok := m.file.SymbolAt(target); ok {
		if off := target - sym.Addr; off == 0 {
			label = sym.Display()
		} else {
			label = fmt.Sprintf("%s+0x%x", sym.Display(), off)
		}
	}
	m.xrefSeq++
	m.xrefRunning = true
	m.xrefTarget = target
	m.xrefLabel = label
	m.setStatus("finding references to "+label+" … (Esc cancels)", false)
	return m.xrefScanCmd(target, m.xrefSeq)
}

// xrefScanCmd decodes the whole executable image in chunks (reusing the decode
// cache) off the UI goroutine and collects instructions that reference target.
func (m *Model) xrefScanCmd(target uint64, seq int) tea.Cmd {
	svc := m.disasmService()
	img := m.file.ExecImage()
	file := m.file
	chunk := m.disasmSearchChunkBytes()
	return func() tea.Msg {
		var hits []xrefHit
		seen := map[uint64]bool{} // an instruction straddling a chunk edge decodes twice
		for pos := 0; pos < img.Len(); {
			win := img.Window(pos, chunk)
			if len(win.Data) == 0 || win.End <= pos {
				break
			}
			for _, inst := range svc.DecodeWindow(win) {
				if seen[inst.Addr] {
					continue
				}
				for from := 0; ; {
					addr, _, end, ok := extractTargetAt(inst.Text, from)
					if !ok {
						break
					}
					if addr == target {
						seen[inst.Addr] = true
						sym := ""
						if s, ok := file.SymbolAt(inst.Addr); ok {
							sym = s.Display()
						}
						hits = append(hits, xrefHit{addr: inst.Addr, text: strings.TrimSpace(inst.Text), sym: sym})
						break
					}
					from = end
				}
				if len(hits) >= xrefMaxHits {
					break
				}
			}
			if len(hits) >= xrefMaxHits {
				break
			}
			pos = win.End
		}
		sort.Slice(hits, func(i, j int) bool { return hits[i].addr < hits[j].addr })
		return xrefDoneMsg{seq: seq, target: target, hits: hits}
	}
}

// handleXrefDone stores a finished scan and opens the modal (or reports none).
func (m *Model) handleXrefDone(msg xrefDoneMsg) (tea.Model, tea.Cmd) {
	if !m.xrefRunning || msg.seq != m.xrefSeq {
		return m, nil // cancelled or superseded
	}
	m.xrefRunning = false
	if len(msg.hits) == 0 {
		m.setStatus("no references to "+m.xrefLabel, false)
		return m, nil
	}
	m.xrefResults = msg.hits
	m.xrefSel = 0
	m.xrefTop = 0
	m.xrefActive = true
	capped := ""
	if len(msg.hits) >= xrefMaxHits {
		capped = "+"
	}
	m.setStatus(fmt.Sprintf("%d%s references to %s", len(msg.hits), capped, m.xrefLabel), false)
	return m, nil
}

// cancelXref abandons an in-flight scan (its result is ignored by seq).
func (m *Model) cancelXref() {
	m.xrefSeq++
	m.xrefRunning = false
	m.setStatus("xref search cancelled", false)
}

// updateXrefModal drives the results list: select with up/down, Enter jumps to
// the referencing instruction, Esc closes.
func (m *Model) updateXrefModal(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "esc":
		m.xrefActive = false
	case "up", "k":
		if m.xrefSel > 0 {
			m.xrefSel--
		}
	case "down", "j":
		if m.xrefSel < len(m.xrefResults)-1 {
			m.xrefSel++
		}
	case "enter":
		if m.xrefSel >= 0 && m.xrefSel < len(m.xrefResults) {
			addr := m.xrefResults[m.xrefSel].addr
			m.xrefActive = false
			m.loadDisasmAt(addr)
		}
	}
	return m, nil
}

func (m *Model) renderXrefModal() string {
	var sb strings.Builder
	sb.WriteString(m.theme.titleStyle.Render(" References to "+m.xrefLabel+" ") + "\n\n")

	addrW := m.file.AddrHexWidth()
	rowW := min(max(72, m.width-14), 120)
	top := visualTop(m.xrefSel, m.xrefTop, len(m.xrefResults), xrefVisible, func(int) int { return 1 })
	end := min(top+xrefVisible, len(m.xrefResults))
	for i := top; i < end; i++ {
		h := m.xrefResults[i]
		loc := h.sym
		if loc == "" {
			loc = "—"
		}
		line := fmt.Sprintf(" 0x%0*x  %s  %s",
			addrW, h.addr, padVisual(truncateMiddle(loc, 22), 22), h.text)
		line = padRight(line, rowW)
		if i == m.xrefSel {
			line = m.theme.tableSelStyle.Render(line)
		}
		sb.WriteString(line + "\n")
	}

	sb.WriteString("\n" + m.theme.footerStyle.Render(
		fmt.Sprintf("↑/↓ select · Enter jump · Esc close   (%d/%d)", m.xrefSel+1, len(m.xrefResults))))
	return m.theme.modalStyle.Render(sb.String())
}
