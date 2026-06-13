package ui

// This file owns the dynamic-libraries view: a list of DT_NEEDED entries
// together with the linkage context (interpreter, libc kind, RPATH, RUNPATH).

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) updateLibs(key string) (tea.Model, tea.Cmd) {
	n := 0
	if m.file.Info != nil {
		n = len(m.file.Info.DynamicLibs)
	}
	if n == 0 {
		return m, nil
	}
	switch key {
	case "up", "k":
		if m.libsCur > 0 {
			m.libsCur--
		}
	case "down", "j":
		if m.libsCur < n-1 {
			m.libsCur++
		}
	case "pgup":
		m.libsCur = max(0, m.libsCur-m.bodyHeight())
	case "pgdown":
		m.libsCur = min(n-1, m.libsCur+m.bodyHeight())
	case "home":
		m.libsCur = 0
	case "end", "G":
		m.libsCur = n - 1
	case "s":
		if m.file.Info != nil && m.libsCur < len(m.file.Info.DynamicLibs) {
			m.copyToClipboard(m.file.Info.DynamicLibs[m.libsCur], "library")
		}
	}
	return m, nil
}

func (m *Model) renderLibs() string {
	bodyH := m.bodyHeight()
	info := m.file.Info
	if info == nil || len(info.DynamicLibs) == 0 {
		body := "no dynamic libraries — this binary is statically linked or has no DT_NEEDED entries\n"
		if info != nil && info.StaticLinked {
			body += "\n" + headerKey.Render("Static-linked:") + " yes\n"
			if info.Libc.Kind != "" && info.Libc.Kind != "none" {
				body += headerKey.Render("Libc:") + " " + info.Libc.Kind
				if info.Libc.Version != "" {
					body += " " + info.Libc.Version
				}
				body += "\n"
			}
		}
		return padBody(body, m.width, bodyH)
	}

	var b strings.Builder
	if info.Interp != "" {
		b.WriteString(headerKey.Render("Interpreter: "))
		b.WriteString(info.Interp + "\n")
	}
	if info.Libc.Kind != "" {
		libcLine := info.Libc.Kind
		if info.Libc.Version != "" {
			libcLine += " " + info.Libc.Version
		}
		if info.Libc.Source != "" {
			libcLine += "  " + footerStyle.Render("("+info.Libc.Source+")")
		}
		b.WriteString(headerKey.Render("Libc:        "))
		b.WriteString(libcLine + "\n")
	}
	if len(info.RPath) > 0 {
		b.WriteString(headerKey.Render("RPATH:       "))
		b.WriteString(strings.Join(info.RPath, ":") + "\n")
	}
	if len(info.RunPath) > 0 {
		b.WriteString(headerKey.Render("RUNPATH:     "))
		b.WriteString(strings.Join(info.RunPath, ":") + "\n")
	}
	b.WriteString("\n")
	b.WriteString(tableHeaderStyle.Render(padRight(fmt.Sprintf(" %3s  %s", "#", "Needed library"), m.width)))
	b.WriteString("\n")

	visible := bodyH - lipgloss.Height(b.String())
	if visible < 1 {
		visible = 1
	}
	if m.libsCur < m.libsTop {
		m.libsTop = m.libsCur
	} else if m.libsCur >= m.libsTop+visible {
		m.libsTop = m.libsCur - visible + 1
	}
	end := m.libsTop + visible
	if end > len(info.DynamicLibs) {
		end = len(info.DynamicLibs)
	}
	for i := m.libsTop; i < end; i++ {
		line := fmt.Sprintf(" %3d  %s", i, info.DynamicLibs[i])
		line = padRight(line, m.width)
		if i == m.libsCur {
			b.WriteString(tableSelStyle.Render(line))
		} else {
			b.WriteString(symbolNameStyle.Render(line))
		}
		b.WriteString("\n")
	}
	return padBody(b.String(), m.width, bodyH)
}
