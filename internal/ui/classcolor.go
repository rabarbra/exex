package ui

import (
	"debug/elf"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/psimonen/elf-explorer/internal/disasm"
)

// styleForClass picks the rendering style for an instruction's class. The
// default (Other) falls through to the mnemonic colour so most instructions
// look uniform and the interesting ones jump out.
func styleForClass(c disasm.InstClass) lipgloss.Style {
	switch c {
	case disasm.ClassCall:
		return classCallStyle
	case disasm.ClassRet:
		return classRetStyle
	case disasm.ClassJumpUnc:
		return classJumpUncStyle
	case disasm.ClassJumpCond:
		return classJumpCndStyle
	case disasm.ClassSyscall:
		return classSyscallStyle
	case disasm.ClassNop:
		return classNopStyle
	}
	return mnemonicStyle
}

// styleForSymbol picks the row colour for a symbol based on its ELF type.
// Bind (LOCAL/GLOBAL/WEAK) is folded in: globals are bold, weaks are italic,
// locals stay plain — so the same colour family stays consistent for the
// type while letting the eye spot scope at a glance.
//
// All colours come from the package-level palette vars so user config (loaded
// by ApplyColors) takes effect automatically.
func styleForSymbol(t elf.SymType, b elf.SymBind) lipgloss.Style {
	var base lipgloss.Style
	switch t {
	case elf.STT_FUNC:
		base = symFuncStyle
	case elf.STT_OBJECT:
		base = symObjectStyle
	case elf.STT_FILE:
		base = symFileStyle
	case elf.STT_SECTION:
		base = symSectionStyle
	case elf.STT_TLS:
		base = symTLSStyle
	case elf.STT_COMMON:
		base = symCommonStyle
	default:
		base = symOtherStyle
	}
	switch b {
	case elf.STB_GLOBAL:
		base = base.Bold(true)
	case elf.STB_WEAK:
		base = base.Italic(true)
	}
	return base
}

// styleForSection picks the row colour for a section based on its semantics.
// Reads from the package-level palette so user config overrides take effect.
func styleForSection(s *elf.Section) lipgloss.Style {
	if s == nil {
		return tableRowStyle
	}
	name := s.Name
	flags := s.Flags

	// Debug info first — typically large and noisy.
	if strings.HasPrefix(name, ".debug") || strings.HasPrefix(name, ".zdebug") {
		return secDebugStyle
	}
	if strings.HasPrefix(name, ".note") {
		return secNoteStyle
	}
	switch s.Type {
	case elf.SHT_SYMTAB, elf.SHT_DYNSYM, elf.SHT_STRTAB:
		return secSymtabStyle
	case elf.SHT_DYNAMIC, elf.SHT_HASH, elf.SHT_GNU_HASH, elf.SHT_GNU_VERSYM,
		elf.SHT_GNU_VERDEF, elf.SHT_GNU_VERNEED:
		return secDynamicStyle
	}
	if s.Type == elf.SHT_REL || s.Type == elf.SHT_RELA {
		return secRelocStyle
	}
	if flags&elf.SHF_EXECINSTR != 0 {
		return secTextStyle
	}
	if flags&elf.SHF_TLS != 0 {
		return secTLSStyle
	}
	if flags&elf.SHF_WRITE != 0 {
		return secDataStyle
	}
	if flags&elf.SHF_ALLOC != 0 {
		return secRodataStyle
	}
	return tableRowStyle
}
