package ui

// Mouse support: the wheel scrolls the active view and a left click selects
// whatever the pointer is over —
// a row in the list views, a byte in the hex/raw dumps, or an instruction in
// the disassembly.

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// doubleClickWindow is how close two clicks must be (in time, on the same row)
// to count as a double-click.
const doubleClickWindow = 350 * time.Millisecond

// wheelFlushMsg is sent by a tea.Tick to flush accumulated scroll deltas.
// Using a deferred flush instead of processing each wheel event immediately
// prevents the render loop from falling behind during momentum scrolling,
// and naturally discards stale events when the user switches views.
type wheelFlushMsg struct{ epoch uint64 }

const wheelFlushInterval = time.Second / 60
const wheelQuietInterval = 120 * time.Millisecond

func (m *Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if msg.Shift && msg.Button == tea.MouseButtonLeft {
		return m, nil
	}
	if m.searchActive && msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
		m.handleSearchPopupClick(msg.X, msg.Y)
		return m, nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		if msg.Shift && m.rightPaneActive() {
			m.scrollRightPane(-3)
			return m, nil
		}
		return m.enqueueWheel(-3)
	case tea.MouseButtonWheelDown:
		if msg.Shift && m.rightPaneActive() {
			m.scrollRightPane(3)
			return m, nil
		}
		return m.enqueueWheel(3)
	}
	if msg.Action == tea.MouseActionPress && msg.Button == tea.MouseButtonLeft {
		if msg.Y == 0 { // the tab strip
			if md, ok := m.tabHitTest(msg.X); ok {
				return m, m.switchMode(md)
			}
			return m, nil
		}
		now := time.Now()
		isDouble := m.mode == modeDisasm && msg.Y == m.lastClickY &&
			now.Sub(m.lastClickAt) < doubleClickWindow
		m.lastClickY = msg.Y
		m.lastClickAt = now

		m.handleClick(msg.X, msg.Y)
		if isDouble {
			m.followCurrentDisasm()
		}
	}
	return m, nil
}

func (m *Model) enqueueWheel(delta int) (tea.Model, tea.Cmd) {
	now := time.Now()
	if now.Before(m.wheelSuppressUntil) {
		m.resetWheelAccumulator()
		m.wheelSuppressUntil = now.Add(wheelQuietInterval)
		return m, nil
	}

	m.wheelDelta += delta
	m.wheelModeSnap = m.mode
	if !m.wheelFlushing {
		m.wheelFlushing = true
		epoch := m.wheelEpoch
		return m, tea.Tick(wheelFlushInterval, func(t time.Time) tea.Msg {
			return wheelFlushMsg{epoch: epoch}
		})
	}
	return m, nil
}

// handleWheelFlush applies the accumulated scroll delta when the mode
// hasn't changed since accumulation — if the user switched views, the
// stale delta is discarded instead of scrolling the new view.
func (m *Model) handleWheelFlush(msg wheelFlushMsg) tea.Cmd {
	if msg.epoch != m.wheelEpoch {
		return nil
	}
	m.wheelFlushing = false
	if m.wheelDelta == 0 {
		return nil
	}
	if m.mode != m.wheelModeSnap {
		m.wheelDelta = 0
		return nil
	}
	n := m.wheelDelta
	m.wheelDelta = 0
	_, cmd := m.routeScroll(n)
	return cmd
}

func (m *Model) routeScroll(delta int) (tea.Model, tea.Cmd) {
	if delta == 0 {
		return m, nil
	}
	switch m.mode {
	case modeSections:
		m.sectionsCur = scrollIndex(m.sectionsCur, len(m.sectionsFiltered), delta)
	case modeSymbols:
		m.symbolsCur = scrollIndex(m.symbolsCur, len(m.symbolsFiltered), delta)
	case modeDisasm:
		m.scrollDisasmBy(delta)
	case modeHex:
		m.ensureHex()
		m.hexCur = scrollBytes(m.hexCur, len(m.hexImg.Data), delta)
	case modeRaw:
		m.ensureRaw()
		m.rawCur = scrollBytes(m.rawCur, len(m.rawData), delta)
	case modeStrings:
		m.ensureStrings()
		m.stringsCur = scrollIndex(m.stringsCur, len(m.stringsList), delta)
	case modeSources:
		m.ensureSources()
		m.sourcesCur = scrollIndex(m.sourcesCur, len(m.sourcesFiltered), delta)
	case modeLibs:
		if m.file.Info != nil {
			m.libsCur = scrollIndex(m.libsCur, len(m.file.Info.DynamicLibs), delta)
		}
	case modeInfo:
		if delta < 0 {
			m.headerVP.LineUp(-delta)
		} else {
			m.headerVP.LineDown(delta)
		}
	}
	return m, nil
}

func (m *Model) scrollDisasmBy(delta int) {
	if delta == 0 {
		return
	}
	if m.sourceFirst && m.srcFile != "" {
		n := len(m.file.SourceLines(m.srcFile))
		if n == 0 {
			return
		}
		next := scrollLine(m.srcCur, n, delta)
		if next != m.srcCur {
			m.srcCur = next
			m.syncSourceAsm()
		}
		return
	}
	if len(m.disasmInst) == 0 {
		return
	}
	target := m.disasmCur + delta
	if target >= 0 && target < len(m.disasmInst) {
		m.disasmCur = target
		m.ensureDisasmViewport(m.disasmViewportHeight())
		return
	}
	forward := delta > 0
	steps := delta
	if steps < 0 {
		steps = -steps
	}
	for i := 0; i < steps; i++ {
		if !m.stepDisasm(forward) {
			return
		}
	}
}

func scrollIndex(cur, n, delta int) int {
	if n <= 0 {
		return 0
	}
	cur += delta
	if cur < 0 {
		return 0
	}
	if cur >= n {
		return n - 1
	}
	return cur
}

func scrollLine(cur, n, delta int) int {
	if n <= 0 {
		return 0
	}
	cur += delta
	if cur < 1 {
		return 1
	}
	if cur > n {
		return n
	}
	return cur
}

func scrollBytes(cur, n, delta int) int {
	if n <= 0 {
		return 0
	}
	return scrollIndex(cur, n, delta*bytesPerHexRow)
}

func (m *Model) handleSearchPopupClick(x, y int) {
	modal := m.renderSearchModal()
	mw := lipgloss.Width(modal)
	mh := lipgloss.Height(modal)
	left := (m.width - mw) / 2
	top := (m.height - mh) / 2
	// Translate to content coordinates inside modalStyle's RoundedBorder (1) +
	// Padding(1,2): x offset 3, y offset 2.
	cx := x - (left + 3)
	cy := y - (top + 2)
	if cy != searchSwitchLine || cx < 0 {
		return
	}
	pos := 0
	sepW := lipgloss.Width(searchSwitchSep)
	for _, sw := range m.searchSwitches() {
		w := lipgloss.Width(sw.label)
		if cx >= pos && cx < pos+w {
			sw.toggle()
			return
		}
		pos += w + sepW
	}
}

// followCurrentDisasm follows the first in-file address on the current disasm
// line — the mouse equivalent of pressing Enter in the disasm view.
func (m *Model) followCurrentDisasm() {
	if len(m.disasmInst) == 0 {
		return
	}
	inst := m.disasmInst[m.disasmCur]
	if target, ok := m.followableAddr(inst.Text); ok {
		m.loadDisasmAt(target)
	} else {
		m.setStatus("no in-file address to follow", true)
	}
}

// handleClick selects the item under the pointer. y == 0 is the tab row; the
// body starts at y == 1, and the footer occupies the final row.
func (m *Model) handleClick(x, y int) {
	bodyRow := y - 1 // strip the tab row
	if bodyRow < 0 || y >= m.height-1 {
		return
	}
	switch m.mode {
	case modeSections:
		// Body layout: row 0 filter, row 1 header, data follows.
		visible := max(1, m.bodyHeight()-2)
		top := visualTop(m.sectionsCur, m.sectionsTop, len(m.sectionsFiltered), visible, m.sectionRowHeight)
		if idx, ok := visualItemAtRow(top, len(m.sectionsFiltered), bodyRow-2, m.sectionRowHeight); ok {
			m.sectionsCur = idx
		}
	case modeSymbols:
		// Body layout: row 0 filter, row 1 header, data follows.
		visible := max(1, m.bodyHeight()-2)
		top := visualTop(m.symbolsCur, m.symbolsTop, len(m.symbolsFiltered), visible, m.symbolRowHeight)
		if idx, ok := visualItemAtRow(top, len(m.symbolsFiltered), bodyRow-2, m.symbolRowHeight); ok {
			m.symbolsCur = idx
		}
	case modeHex:
		m.ensureHex()
		m.hexCur = m.clickByte(modeHex, m.hexImg.Data, hexVisibleTop(m.hexCur, m.hexTop, max(1, m.bodyHeight()-1)), m.hexCur, x, bodyRow, m.hexImg.AddrAt)
	case modeRaw:
		m.ensureRaw()
		m.rawCur = m.clickByte(modeRaw, m.rawData, hexVisibleTop(m.rawCur, m.rawTop, max(1, m.bodyHeight()-1)), m.rawCur, x, bodyRow, func(pos int) uint64 { return uint64(pos) })
	case modeStrings:
		// Body layout: row 0 is the column header, data follows.
		visible := max(1, m.bodyHeight()-1)
		top := visualTop(m.stringsCur, m.stringsTop, len(m.stringsList), visible, m.stringRowHeight)
		if idx, ok := visualItemAtRow(top, len(m.stringsList), bodyRow-1, m.stringRowHeight); ok {
			m.stringsCur = idx
		}
	case modeSources:
		// File list only: row 0 is the filter, files follow.
		visible := max(1, m.bodyHeight()-1)
		top := visualTop(m.sourcesCur, m.sourcesTop, len(m.sourcesFiltered), visible, func(int) int { return 1 })
		if idx := top + bodyRow - 1; idx >= 0 && idx < len(m.sourcesFiltered) {
			m.sourcesCur = idx
		}
	case modeDisasm:
		if m.sourceFirst && m.srcFile != "" && m.clickInSourcePane(x) {
			if ln, ok := m.sourceLineAtBodyRow(bodyRow, m.sourcePaneWidth()); ok {
				m.srcCur = ln
				m.syncSourceAsm()
			}
		} else if i, ok := m.instAtBodyRow(bodyRow); ok {
			m.disasmCur = i
		}
	case modeLibs:
		headerRows := m.libsHeaderRows()
		if m.file.Info != nil {
			visible := max(1, m.bodyHeight()-headerRows)
			top := visualTop(m.libsCur, m.libsTop, len(m.file.Info.DynamicLibs), visible, m.libRowHeight)
			if idx, ok := visualItemAtRow(top, len(m.file.Info.DynamicLibs), bodyRow-headerRows, m.libRowHeight); ok {
				m.libsCur = idx
			}
		}
	}
}

func (m *Model) clickInSourcePane(x int) bool {
	// In the disasm view's source-first split, the source pane is on the left.
	return x < m.width/2
}

func (m *Model) sourcePaneWidth() int {
	if m.width <= 1 {
		return m.width
	}
	return m.width / 2
}

func (m *Model) sourceLineAtBodyRow(bodyRow, paneW int) (int, bool) {
	r := bodyRow - 1 // strip source header row
	if r < 0 {
		return 0, false
	}
	src := m.file.SourceLines(m.srcFile)
	contentH := max(1, m.bodyHeight()-1)
	rowHeight := func(i int) int {
		ln := i + 1
		h := m.sourceLineHeight(ln, paneW)
		if ln == m.srcCur && len(m.sourceLineColumns(m.srcFile, ln)) > 0 {
			h++
		}
		return h
	}
	idx, ok := visualItemAtRow(m.sourceTextTop(paneW, contentH), len(src), r, rowHeight)
	return idx + 1, ok
}

// clickByte maps a click at (x, bodyRow) onto a byte position in a hex dump.
// Body layout: row 0 is the banner, byte rows follow with bytesPerHexRow bytes
// each. The column→byte mapping lives in view_hex.go so it stays in sync with
// the renderer.
func (m *Model) clickByte(md mode, data []byte, top, cur, x, bodyRow int, addrAt func(pos int) uint64) int {
	r := bodyRow - 1 // strip the banner row
	if r < 0 {
		return cur
	}
	emitted := 0
	prevSec := ""
	if top >= bytesPerHexRow {
		prevSec = m.hexSectionName(md, top-bytesPerHexRow, addrAt)
	}
	for rowStart := top; rowStart < len(data); rowStart += bytesPerHexRow {
		if sec := m.hexSectionName(md, rowStart, addrAt); sec != "" && sec != prevSec {
			if emitted == r {
				return cur // clicked a section-separator row
			}
			emitted++
			prevSec = sec
		} else {
			prevSec = sec
		}
		if emitted == r {
			pos := rowStart + hexColumnToByte(m.file.AddrHexWidth(), x)
			if pos >= len(data) {
				pos = len(data) - 1
			}
			return pos
		}
		emitted++
	}
	return cur
}

// instAtBodyRow maps a click in the disasm scroller to an instruction index.
// It replays renderDisasmScroll's emit order: a symbol-start instruction is
// preceded by a "<name>:" label line, so rows aren't 1:1 with instructions.
func (m *Model) instAtBodyRow(bodyRow int) (int, bool) {
	r := bodyRow - 1 // strip the sticky-symbol row
	if r < 0 {
		return 0, false
	}
	visible := max(1, m.bodyHeight()-1)
	rowHeight := func(i int) int {
		return m.disasmInstVisualHeight(i, m.disasmRenderWidth())
	}
	top := visualTop(m.disasmCur, m.disasmTop, len(m.disasmInst), visible, rowHeight)
	return visualItemAtRow(top, len(m.disasmInst), r, rowHeight)
}
