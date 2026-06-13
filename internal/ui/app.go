// Package ui implements the Bubble Tea TUI for elf-explorer.
package ui

import (
	"debug/elf"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/psimonen/elf-explorer/internal/binfile"
	"github.com/psimonen/elf-explorer/internal/config"
	"github.com/psimonen/elf-explorer/internal/disasm"
)

type mode int

const (
	modeInfo mode = iota
	modeSections
	modeSymbols
	modeDisasm
	modeHex
	modeLibs
)

func (m mode) String() string {
	switch m {
	case modeInfo:
		return "Info"
	case modeSections:
		return "Sections"
	case modeSymbols:
		return "Symbols"
	case modeDisasm:
		return "Disasm"
	case modeHex:
		return "Hex"
	case modeLibs:
		return "Libs"
	}
	return "?"
}

// Model is the root Bubble Tea model.
type Model struct {
	file *binfile.File
	dis  disasm.Disassembler

	mode mode

	width, height int

	// Header view.
	headerVP viewport.Model

	// Sections view.
	sections    []*elf.Section
	sectionsCur int
	sectionsTop int

	// Symbols view.
	symbolsFilter   textinput.Model
	symbolsFiltered []int // indices into file.Symbols (sorted by name)
	symbolsCur      int
	symbolsTop      int

	// Disasm view.
	disasmAddr uint64
	disasmInst []disasm.Inst
	disasmCur  int
	disasmTop  int
	showSource bool
	srcVP      viewport.Model

	// Navigation history for the disasm view: the last `historyCap` jump
	// targets, with `historyPos` indicating where in that ring we are. Left
	// arrow steps back, right arrow steps forward.
	history    []uint64
	historyPos int

	// Hex view.
	hexBase    uint64
	hexData    []byte
	hexSection *elf.Section
	hexCur     int // byte offset into hexData
	hexTop     int // first row's byte offset (multiple of bytesPerHexRow)

	// Libs view.
	libsCur int
	libsTop int

	// Go-to-address modal.
	gotoInput  textinput.Model
	gotoActive bool

	// Transient status message displayed in the footer.
	status      string
	statusError bool

	// User-configurable keymap for the top-level dispatch.
	keys keyMap
}

func New(f *binfile.File) (*Model, error) {
	d, err := disasm.For(f.Machine())
	if err != nil {
		// Don't fail — the user can still browse header/sections/symbols.
		d = nil
	}

	// Load user config and overlay it before constructing styles/keymap.
	// A missing config file is fine (zero Config); a malformed one surfaces.
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	ApplyColors(cfg.Colors)
	keys := defaultKeyMap()
	keys.applyConfig(cfg.Keys)

	filter := textinput.New()
	filter.Placeholder = "type to filter…"
	filter.Prompt = "/ "
	filter.CharLimit = 256

	gotoInput := textinput.New()
	gotoInput.Placeholder = "0x401000 or symbol name"
	gotoInput.Prompt = "→ "
	gotoInput.CharLimit = 256

	m := &Model{
		file:          f,
		dis:           d,
		mode:          modeInfo,
		sections:      f.Sections,
		symbolsFilter: filter,
		gotoInput:     gotoInput,
		showSource:    true,
		keys:          keys,
	}
	m.headerVP = viewport.New(0, 0)
	m.srcVP = viewport.New(0, 0)
	m.recomputeSymbols()

	// Land the disasm cursor on the entry point by default if it's mapped.
	if d != nil && f.Entry() != 0 {
		m.loadDisasmAt(f.Entry())
	}
	return m, nil
}

func (m *Model) Init() tea.Cmd { return nil }

// recomputeSymbols rebuilds symbolsFiltered from the current filter text.
func (m *Model) recomputeSymbols() {
	needle := strings.ToLower(m.symbolsFilter.Value())
	m.symbolsFiltered = m.symbolsFiltered[:0]
	for i, s := range m.file.Symbols {
		if needle == "" || strings.Contains(strings.ToLower(s.Name), needle) {
			m.symbolsFiltered = append(m.symbolsFiltered, i)
		}
	}
	if m.symbolsCur >= len(m.symbolsFiltered) {
		m.symbolsCur = max(0, len(m.symbolsFiltered)-1)
	}
}


func (m *Model) setStatus(s string, isError bool) {
	m.status = s
	m.statusError = isError
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		bodyH := m.bodyHeight()
		m.headerVP.Width = m.width
		m.headerVP.Height = bodyH
		m.srcVP.Width = m.width / 2
		m.srcVP.Height = bodyH
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	// Modal owns input while active.
	if m.gotoActive {
		switch key {
		case "esc":
			m.gotoActive = false
			m.gotoInput.Blur()
			m.gotoInput.SetValue("")
			return m, nil
		case "enter":
			val := strings.TrimSpace(m.gotoInput.Value())
			m.gotoActive = false
			m.gotoInput.Blur()
			m.gotoInput.SetValue("")
			m.handleGoto(val)
			return m, nil
		}
		var cmd tea.Cmd
		m.gotoInput, cmd = m.gotoInput.Update(msg)
		return m, cmd
	}

	// Filter input in symbols view captures typing keys.
	if m.mode == modeSymbols && m.symbolsFilter.Focused() {
		switch key {
		case "esc":
			m.symbolsFilter.Blur()
			return m, nil
		case "enter":
			m.symbolsFilter.Blur()
			return m, nil
		case "up", "down", "pgup", "pgdown", "home", "end":
			// Let navigation keys fall through.
		default:
			var cmd tea.Cmd
			m.symbolsFilter, cmd = m.symbolsFilter.Update(msg)
			m.recomputeSymbols()
			return m, cmd
		}
	}

	switch m.keys[key] {
	case actionQuit:
		return m, tea.Quit
	case actionViewInfo:
		m.mode = modeInfo
		return m, nil
	case actionViewSections:
		m.mode = modeSections
		return m, nil
	case actionViewSymbols:
		m.mode = modeSymbols
		return m, nil
	case actionViewDisasm:
		if m.dis == nil {
			m.setStatus("no disassembler for this architecture", true)
			return m, nil
		}
		m.mode = modeDisasm
		return m, nil
	case actionViewHex:
		m.mode = modeHex
		return m, nil
	case actionViewLibs:
		m.mode = modeLibs
		return m, nil
	case actionGoto:
		m.gotoActive = true
		m.gotoInput.Focus()
		return m, nil
	case actionToggleSource:
		if m.mode == modeDisasm {
			m.showSource = !m.showSource
		}
		return m, nil
	}

	switch m.mode {
	case modeSections:
		return m.updateSections(key)
	case modeSymbols:
		return m.updateSymbols(key)
	case modeDisasm:
		return m.updateDisasm(key)
	case modeHex:
		return m.updateHex(key)
	case modeLibs:
		return m.updateLibs(key)
	case modeInfo:
		var cmd tea.Cmd
		m.headerVP, cmd = m.headerVP.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m *Model) updateSections(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "up", "k":
		if m.sectionsCur > 0 {
			m.sectionsCur--
		}
	case "down", "j":
		if m.sectionsCur < len(m.sections)-1 {
			m.sectionsCur++
		}
	case "pgup":
		m.sectionsCur = max(0, m.sectionsCur-m.bodyHeight())
	case "pgdown":
		m.sectionsCur = min(len(m.sections)-1, m.sectionsCur+m.bodyHeight())
	case "home", "g g":
		m.sectionsCur = 0
	case "end", "G":
		m.sectionsCur = len(m.sections) - 1
	case "enter":
		sec := m.sections[m.sectionsCur]
		if binfile.IsExecSection(sec) && m.dis != nil {
			m.loadDisasmAt(sec.Addr)
		} else {
			m.openHexForSection(sec)
		}
	case "a":
		sec := m.sections[m.sectionsCur]
		m.copyToClipboard(fmt.Sprintf("0x%0*x", m.file.AddrHexWidth(), sec.Addr), "address")
	case "s":
		sec := m.sections[m.sectionsCur]
		m.copyToClipboard(sec.Name, "section name")
	}
	return m, nil
}



func (m *Model) updateSymbols(key string) (tea.Model, tea.Cmd) {
	switch key {
	case "/":
		m.symbolsFilter.Focus()
		return m, nil
	case "up", "k":
		if m.symbolsCur > 0 {
			m.symbolsCur--
		}
	case "down", "j":
		if m.symbolsCur < len(m.symbolsFiltered)-1 {
			m.symbolsCur++
		}
	case "pgup":
		m.symbolsCur = max(0, m.symbolsCur-m.bodyHeight())
	case "pgdown":
		m.symbolsCur = min(len(m.symbolsFiltered)-1, m.symbolsCur+m.bodyHeight())
	case "home":
		m.symbolsCur = 0
	case "end", "G":
		m.symbolsCur = len(m.symbolsFiltered) - 1
	case "enter":
		if len(m.symbolsFiltered) == 0 {
			return m, nil
		}
		sym := m.file.Symbols[m.symbolsFiltered[m.symbolsCur]]
		if sym.Addr == 0 {
			m.setStatus(fmt.Sprintf("symbol %s has no address", sym.Name), true)
			return m, nil
		}
		m.openSymbol(sym)
	case "a":
		if len(m.symbolsFiltered) == 0 {
			return m, nil
		}
		sym := m.file.Symbols[m.symbolsFiltered[m.symbolsCur]]
		m.copyToClipboard(fmt.Sprintf("0x%0*x", m.file.AddrHexWidth(), sym.Addr), "address")
	case "s":
		if len(m.symbolsFiltered) == 0 {
			return m, nil
		}
		sym := m.file.Symbols[m.symbolsFiltered[m.symbolsCur]]
		m.copyToClipboard(sym.Name, "symbol")
	}
	return m, nil
}

// openSymbol opens a symbol in the most appropriate view:
//   - FUNC                  → disasm
//   - OBJECT/TLS/COMMON     → hex view of the *whole containing section*,
//                             with the cursor parked on the symbol's bytes
//                             (so the user can scroll around adjacent data)
//   - SECTION               → hex view of the whole section
//   - NOTYPE                → exec section ⇒ disasm; else hex (whole section)
func (m *Model) openSymbol(sym binfile.Symbol) {
	switch sym.Type {
	case elf.STT_FUNC:
		m.loadDisasmAt(sym.Addr)
	case elf.STT_OBJECT, elf.STT_TLS, elf.STT_COMMON:
		m.openHexInSectionAt(sym.Addr, sym.Size)
	case elf.STT_SECTION:
		if sec := m.file.SectionAt(sym.Addr); sec != nil {
			m.openHexForSection(sec)
		} else {
			m.openHexAt(sym.Addr, sym.Size)
		}
	default:
		if sec := m.file.SectionAt(sym.Addr); sec != nil && binfile.IsExecSection(sec) {
			m.loadDisasmAt(sym.Addr)
		} else {
			m.openHexInSectionAt(sym.Addr, sym.Size)
		}
	}
}

// openHexInSectionAt opens the hex viewer on the section containing addr,
// then seeks the cursor to addr (and best-effort to the start of the
// symbol's row in the visible window). If addr isn't inside any allocated
// section it falls back to a windowed read of just `size` bytes.
func (m *Model) openHexInSectionAt(addr, size uint64) {
	sec := m.file.SectionAt(addr)
	if sec == nil {
		m.openHexAt(addr, size)
		return
	}
	m.openHexForSection(sec)
	off := int(addr - sec.Addr)
	if off < 0 || off >= len(m.hexData) {
		return
	}
	m.hexCur = off
	// Align the top row to the cursor's row so the symbol is immediately
	// visible after the jump.
	m.hexTop = (off / bytesPerHexRow) * bytesPerHexRow
}

// copyToClipboard puts text on the system clipboard and reports success or
// failure to the user via the status footer.
func (m *Model) copyToClipboard(text, what string) {
	if err := clipboard.WriteAll(text); err != nil {
		m.setStatus(fmt.Sprintf("clipboard: %v", err), true)
		return
	}
	m.setStatus(fmt.Sprintf("copied %s: %s", what, text), false)
}

func (m *Model) handleGoto(val string) {
	if val == "" {
		return
	}
	// Hex address first.
	parsed, err := parseAddr(val)
	if err == nil {
		if m.dis == nil {
			m.setStatus("no disassembler for this architecture", true)
			return
		}
		m.loadDisasmAt(parsed)
		return
	}
	// Else treat as symbol name (exact, then substring).
	idx := sort.Search(len(m.file.Symbols), func(i int) bool { return m.file.Symbols[i].Name >= val })
	if idx < len(m.file.Symbols) && m.file.Symbols[idx].Name == val {
		s := m.file.Symbols[idx]
		if s.Addr != 0 && m.dis != nil {
			m.loadDisasmAt(s.Addr)
			return
		}
	}
	needle := strings.ToLower(val)
	for _, s := range m.file.Symbols {
		if s.Addr != 0 && strings.Contains(strings.ToLower(s.Name), needle) {
			if m.dis != nil {
				m.loadDisasmAt(s.Addr)
				return
			}
		}
	}
	m.setStatus(fmt.Sprintf("not found: %s", val), true)
}

func parseAddr(s string) (uint64, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		return strconv.ParseUint(s[2:], 16, 64)
	}
	// Heuristic: any [a-f] means hex.
	for _, r := range s {
		if r >= 'a' && r <= 'f' || r >= 'A' && r <= 'F' {
			return strconv.ParseUint(s, 16, 64)
		}
	}
	return strconv.ParseUint(s, 10, 64)
}

// View renders the screen.
func (m *Model) View() string {
	if m.width == 0 || m.height == 0 {
		return "initializing…"
	}
	parts := []string{m.renderTabs()}
	body := ""
	switch m.mode {
	case modeInfo:
		body = m.renderInfo()
	case modeSections:
		body = m.renderSections()
	case modeSymbols:
		body = m.renderSymbols()
	case modeDisasm:
		body = m.renderDisasm()
	case modeHex:
		body = m.renderHex()
	case modeLibs:
		body = m.renderLibs()
	}
	parts = append(parts, body, m.renderFooter())
	out := lipgloss.JoinVertical(lipgloss.Left, parts...)
	if m.gotoActive {
		modal := modalStyle.Render("Go to address or symbol\n\n" + m.gotoInput.View() + "\n\nEnter to jump  Esc to cancel")
		mw := lipgloss.Width(modal)
		mh := lipgloss.Height(modal)
		out = overlay(out, modal, (m.width-mw)/2, (m.height-mh)/2)
	}
	return out
}

func (m *Model) renderTabs() string {
	render := func(label string, active bool) string {
		if active {
			return activeTabStyle.Render(label)
		}
		return tabStyle.Render(label)
	}
	tabs := []string{
		titleStyle.Render(" elf-explorer "),
		render("1·Info", m.mode == modeInfo),
		render("2·Sections", m.mode == modeSections),
		render("3·Symbols", m.mode == modeSymbols),
		render("4·Disasm", m.mode == modeDisasm),
		render("5·Hex", m.mode == modeHex),
		render("6·Libs", m.mode == modeLibs),
	}
	row := lipgloss.JoinHorizontal(lipgloss.Left, tabs...)
	pad := m.width - lipgloss.Width(row)
	if pad > 0 {
		row += strings.Repeat(" ", pad)
	}
	return row
}

func (m *Model) renderFooter() string {
	var help string
	switch m.mode {
	case modeInfo:
		help = "1-6 switch view · g goto · q quit"
	case modeSections:
		help = "↑/↓ move · Enter view (disasm or hex) · g goto · q quit"
	case modeSymbols:
		help = "↑/↓ move · / filter · Enter jump · g goto · q quit"
	case modeDisasm:
		help = "↑/↓ scroll · ←/→ history · Enter follow · a copy addr · s copy sym · Tab source · g goto · q quit"
	case modeHex:
		help = "←/↓/↑/→ move · a copy addr · s copy sym · g goto · q quit"
	case modeLibs:
		help = "↑/↓ move · q quit"
	}
	left := footerStyle.Render(help)
	right := ""
	if m.status != "" {
		st := infoStyle
		if m.statusError {
			st = errorStyle
		}
		right = st.Render(m.status)
	}
	gap := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		gap = 1
	}
	return left + strings.Repeat(" ", gap) + right
}

// bodyHeight is the number of rows available between tabs and footer.
func (m *Model) bodyHeight() int {
	if m.height <= 2 {
		return 1
	}
	return m.height - 2
}

func (m *Model) renderInfo() string {
	var b strings.Builder
	kv := func(k, v string) {
		b.WriteString(headerKey.Render(padKey(k, 14)))
		b.WriteString(" ")
		b.WriteString(v)
		b.WriteString("\n")
	}

	// Core ELF header.
	for _, l := range m.file.HeaderInfo() {
		if idx := strings.IndexByte(l, ':'); idx >= 0 {
			b.WriteString(headerKey.Render(l[:idx+1]))
			b.WriteString(l[idx+1:])
		} else {
			b.WriteString(l)
		}
		b.WriteString("\n")
	}

	if m.dis != nil {
		kv("Disassembler:", m.dis.Name())
	}

	if info := m.file.Info; info != nil {
		b.WriteString("\n")
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
		kv("Stripped:", fmt.Sprintf("%v", info.Stripped))
		kv("Static-linked:", fmt.Sprintf("%v", info.StaticLinked))
		if info.Libc.Kind != "" {
			val := info.Libc.Kind
			if info.Libc.Version != "" {
				val += " " + info.Libc.Version
			}
			if info.Libc.Source != "" {
				val += "  " + footerStyle.Render("("+info.Libc.Source+")")
			}
			kv("Libc:", val)
		}
		if len(info.DynamicLibs) > 0 {
			kv("Needed libs:", fmt.Sprintf("%d (press 6 to view)", len(info.DynamicLibs)))
		}
	}

	return padBody(b.String(), m.width, m.bodyHeight())
}

// padKey right-pads a key label to a fixed column, ignoring the trailing colon
// for alignment purposes.
func padKey(s string, w int) string {
	if len(s) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-len(s))
}

func (m *Model) renderSections() string {
	bodyH := m.bodyHeight()
	// columns: idx, name, type, addr, size, flags
	addrW := m.file.AddrHexWidth()        // hex digits in an address
	addrCol := 2 + addrW                  // "0x" + digits
	hdr := fmt.Sprintf(" %3s  %-22s %-14s %-*s %-12s  %s",
		"#", "Name", "Type", addrCol, "Addr", "Size", "Flags")
	if len(hdr) > m.width {
		hdr = hdr[:m.width]
	}
	header := tableHeaderStyle.Render(padRight(hdr, m.width))

	visible := bodyH - 1
	if visible < 1 {
		visible = 1
	}
	if m.sectionsCur < m.sectionsTop {
		m.sectionsTop = m.sectionsCur
	} else if m.sectionsCur >= m.sectionsTop+visible {
		m.sectionsTop = m.sectionsCur - visible + 1
	}
	end := m.sectionsTop + visible
	if end > len(m.sections) {
		end = len(m.sections)
	}

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")
	for i := m.sectionsTop; i < end; i++ {
		s := m.sections[i]
		line := fmt.Sprintf(" %3d  %-22s %-14s 0x%0*x %-12d  %s",
			i, truncate(s.Name, 22), trimSecType(s.Type.String()), addrW, s.Addr, s.Size, sectionFlags(s.Flags))
		line = padRight(line, m.width)
		if i == m.sectionsCur {
			b.WriteString(tableSelStyle.Render(line))
		} else {
			b.WriteString(styleForSection(s).Render(line))
		}
		b.WriteString("\n")
	}
	return padBody(b.String(), m.width, bodyH)
}

func (m *Model) renderSymbols() string {
	bodyH := m.bodyHeight()
	if bodyH < 3 {
		bodyH = 3
	}

	filterRow := m.symbolsFilter.View()
	if !m.symbolsFilter.Focused() {
		filterRow = footerStyle.Render(fmt.Sprintf("/ %s   (%d / %d)", m.symbolsFilter.Value(), len(m.symbolsFiltered), len(m.file.Symbols)))
	}

	addrW := m.file.AddrHexWidth()
	addrCol := 2 + addrW
	hdr := fmt.Sprintf(" %-*s %-6s %-5s %-8s  %s", addrCol, "Address", "Size", "Bind", "Type", "Name")
	header := tableHeaderStyle.Render(padRight(hdr, m.width))

	visible := bodyH - 2 // filter row + header
	if visible < 1 {
		visible = 1
	}
	if m.symbolsCur < m.symbolsTop {
		m.symbolsTop = m.symbolsCur
	} else if m.symbolsCur >= m.symbolsTop+visible {
		m.symbolsTop = m.symbolsCur - visible + 1
	}
	end := m.symbolsTop + visible
	if end > len(m.symbolsFiltered) {
		end = len(m.symbolsFiltered)
	}

	var rows strings.Builder
	rows.WriteString(filterRow)
	rows.WriteString("\n")
	rows.WriteString(header)
	rows.WriteString("\n")
	for i := m.symbolsTop; i < end; i++ {
		s := m.file.Symbols[m.symbolsFiltered[i]]
		line := fmt.Sprintf(" 0x%0*x %-6d %-5s %-8s  %s",
			addrW, s.Addr, s.Size, trimBind(s.Bind), trimType(s.Type), s.Name)
		line = padRight(line, m.width)
		if i == m.symbolsCur {
			rows.WriteString(tableSelStyle.Render(line))
		} else {
			rows.WriteString(styleForSymbol(s.Type, s.Bind).Render(line))
		}
		rows.WriteString("\n")
	}
	return padBody(rows.String(), m.width, bodyH)
}


// ---- helpers ----

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// bytesHex renders up to maxN bytes as space-separated, per-byte-coloured hex.
// The output is padded with plain spaces to a fixed visible width so columns
// line up regardless of how many bytes the instruction occupied. Uses the
// precomputed byteHex table to avoid re-rendering ANSI codes on every byte.
func bytesHex(b []byte, maxN int) string {
	if len(b) > maxN {
		b = b[:maxN]
	}
	var sb strings.Builder
	for i, x := range b {
		if i > 0 {
			sb.WriteByte(' ')
		}
		sb.WriteString(byteHex[x])
	}
	visible := len(b)*3 - 1
	if len(b) == 0 {
		visible = 0
	}
	want := maxN*3 - 1
	if visible < want {
		sb.WriteString(strings.Repeat(" ", want-visible))
	}
	return sb.String()
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

func padRight(s string, w int) string {
	plain := stripANSI(s)
	if lipgloss.Width(plain) >= w {
		return s
	}
	return s + strings.Repeat(" ", w-lipgloss.Width(plain))
}

func padBody(s string, w, h int) string {
	lines := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(lines) > h {
		lines = lines[:h]
	}
	for len(lines) < h {
		lines = append(lines, strings.Repeat(" ", w))
	}
	return strings.Join(lines, "\n")
}

func sectionFlags(f elf.SectionFlag) string {
	var b strings.Builder
	if f&elf.SHF_ALLOC != 0 {
		b.WriteByte('A')
	}
	if f&elf.SHF_WRITE != 0 {
		b.WriteByte('W')
	}
	if f&elf.SHF_EXECINSTR != 0 {
		b.WriteByte('X')
	}
	if f&elf.SHF_MERGE != 0 {
		b.WriteByte('M')
	}
	if f&elf.SHF_STRINGS != 0 {
		b.WriteByte('S')
	}
	if f&elf.SHF_TLS != 0 {
		b.WriteByte('T')
	}
	if b.Len() == 0 {
		return "-"
	}
	return b.String()
}

func trimSecType(s string) string { return strings.TrimPrefix(s, "SHT_") }
func trimBind(b elf.SymBind) string {
	return strings.TrimPrefix(b.String(), "STB_")
}
func trimType(t elf.SymType) string {
	return strings.TrimPrefix(t.String(), "STT_")
}

// overlay places fg over bg at column x, row y. Both are pre-rendered strings.
func overlay(bg, fg string, x, y int) string {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	bgLines := strings.Split(bg, "\n")
	fgLines := strings.Split(fg, "\n")
	for i, fl := range fgLines {
		row := y + i
		if row >= len(bgLines) {
			break
		}
		bgLine := bgLines[row]
		// Convert to runes for width-safe slicing — assume printable.
		plain := stripANSI(bgLine)
		if x >= lipgloss.Width(plain) {
			bgLines[row] = bgLine + strings.Repeat(" ", x-lipgloss.Width(plain)) + fl
			continue
		}
		// Best-effort overlay: just replace the whole row when fg width fits.
		fw := lipgloss.Width(stripANSI(fl))
		prefix := plain
		if x < len(prefix) {
			prefix = prefix[:x]
		}
		suffix := ""
		if x+fw < lipgloss.Width(plain) {
			suffix = plain[x+fw:]
		}
		bgLines[row] = prefix + fl + suffix
	}
	return strings.Join(bgLines, "\n")
}

// stripANSI removes ANSI escape sequences for width math. Cheap and good enough
// for our render strings, which only carry simple SGR codes from lipgloss.
func stripANSI(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] == 0x1b && i+1 < len(s) && s[i+1] == '[' {
			j := i + 2
			for j < len(s) && (s[j] < 0x40 || s[j] > 0x7e) {
				j++
			}
			if j < len(s) {
				j++
			}
			i = j - 1
			continue
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// truncateANSI naively truncates while keeping the trailing SGR reset.
func truncateANSI(s string, w int) string {
	plain := stripANSI(s)
	if lipgloss.Width(plain) <= w {
		return s
	}
	// Walk and drop characters from the end of the plain content. Cheap fallback.
	return plain[:w]
}
