package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// View implements tea.Model.
func (m *Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	if m.err != nil {
		return m.renderErrorPopup(m.err.Error())
	}

	if m.terminalWidth < 40 || m.terminalHeight < 12 {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorOrange)).
			Bold(true).
			Render(fmt.Sprintf("Terminal too small!\nMinimum: 40x12\nCurrent: %dx%d",
				m.terminalWidth, m.terminalHeight))
	}

	if m.inModeMenu {
		return m.renderModeMenuWithProxyGroups()
	}

	sections := []string{
		m.renderTopBar(),
		m.renderTabBar(),
	}

	switch m.activeTab {
	case TabLogs:
		sections = append(sections, m.renderLogPanel())
		sections = append(sections,
			FooterStyle.Render("q: quit | Tab/1/2: switch tab | d/i/w/e: filters | a: all"))
	default:
		sections = append(sections, m.renderMainPanel())
		sections = append(sections,
			FooterStyle.Render("q: quit | Tab/1/2: switch tab | m: mode"))
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// ---------- Top bar ----------

func (m *Model) renderTopBar() string {
	var status string
	if m.connected {
		status = ConnectedStyle.Render("● Connected")
	} else {
		status = DisconnectedStyle.Render("● Disconnected")
	}

	runtime := time.Since(m.startTime).Round(time.Second)
	left := "🚀 Clash Dashboard"
	right := fmt.Sprintf("%s | ⏱ %s", status, runtime)

	avail := m.terminalWidth - 2
	if avail < 40 {
		return TopBarStyle.Render("🚀 Clash")
	}

	pad := max(0, avail-lipgloss.Width(left)-lipgloss.Width(right))
	return TopBarStyle.Render(left + strings.Repeat(" ", pad) + right)
}

// ---------- Tab bar ----------

func (m *Model) renderTabBar() string {
	tabs := []struct {
		label string
		id    int
	}{
		{"Dashboard", TabDashboard},
		{"Logs", TabLogs},
	}

	var rendered []string
	for _, t := range tabs {
		if t.id == m.activeTab {
			rendered = append(rendered, ActiveTabStyle.Render(t.label))
		} else {
			rendered = append(rendered, InactiveTabStyle.Render(t.label))
		}
	}
	return TabBarStyle.Render(lipgloss.JoinHorizontal(lipgloss.Bottom, rendered...))
}

// ---------- Main panel (status cards + actions) ----------

func (m *Model) renderMainPanel() string {
	var cards string
	if m.terminalWidth < 80 {
		cards = lipgloss.JoinVertical(lipgloss.Left,
			m.renderConnectionCard(), "",
			m.renderProxyCard(), "",
			m.renderTrafficCard())
	} else {
		cards = lipgloss.JoinHorizontal(lipgloss.Top,
			m.renderConnectionCard(), "  ",
			m.renderProxyCard(), "  ",
			m.renderTrafficCard())
	}

	actions := m.renderQuickActions()
	content := lipgloss.JoinVertical(lipgloss.Left, cards, "", actions)
	return MainPanelStyle.Render(content)
}

func (m *Model) renderConnectionCard() string {
	title := cardTitle("Connection")
	lines := []string{
		title,
		statusRow("Active:", fmt.Sprintf("%d", m.activeConnections)),
		statusRow("Traffic:", formatBytes(m.totalTraffic)),
	}
	return StatusCardStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m *Model) renderProxyCard() string {
	title := cardTitle("Proxy")
	lines := []string{
		title,
		statusRow("Mode:", m.currentMode),
		statusRow("Groups:", fmt.Sprintf("%d", len(m.proxyGroupList))),
	}
	return StatusCardStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m *Model) renderTrafficCard() string {
	title := cardTitle("Traffic")
	lines := []string{
		title,
		statusRow("Upload:", formatSpeed(m.uploadSpeed)),
		statusRow("Download:", formatSpeed(m.downloadSpeed)),
		statusRow("Total:", formatBytes(m.totalTraffic)),
	}
	return StatusCardStyle.Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m *Model) renderQuickActions() string {
	actions := []string{
		QuickActionStyle.Render("Global"),
		QuickActionStyle.Render("Rule"),
		QuickActionStyle.Render("Direct"),
		QuickActionStyle.Render("Reload"),
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, actions...)
}

// ---------- Log panel ----------

func (m *Model) renderLogPanel() string {
	var filterText string
	if len(m.filters) == 0 {
		filterText = "all"
	} else {
		filterText = strings.Join(m.filters, ", ")
	}

	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorPurple)).
		Render(fmt.Sprintf("🔍 Logs (%d) [%s]", len(m.logs), filterText))

	content := lipgloss.JoinVertical(lipgloss.Left, title, m.viewport.View())
	return LogPanelStyle.Render(content)
}

func (m *Model) renderLogs() string {
	var b strings.Builder
	first := true
	for _, log := range m.logs {
		if !m.isLogVisible(log) {
			continue
		}
		style, ok := TypeStyles[log.Type]
		if !ok {
			style = PayloadStyle
		}
		if !first {
			b.WriteByte('\n')
		}
		first = false
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Left,
			TimeStyle.Render(fmt.Sprintf("[%s]", log.Time)),
			" ",
			style.Render(fmt.Sprintf("%-7s", log.Type)),
			" ",
			PayloadStyle.Render(log.Payload),
		))
	}
	return b.String()
}

// ---------- Mode / Proxy group menu ----------

func (m *Model) renderModeMenuWithProxyGroups() string {
	modeCard := m.renderModeCard()
	proxyCard := m.renderProxyGroupCard()
	help := m.renderModeMenuHelp()

	top := lipgloss.JoinHorizontal(lipgloss.Top, modeCard, "  ", proxyCard)
	content := lipgloss.JoinVertical(lipgloss.Left, top, "", help)
	return lipgloss.NewStyle().Padding(2).Render(content)
}

func (m *Model) renderModeCard() string {
	var lines []string
	lines = append(lines,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorPurple)).Render("🔄 Clash Mode"),
		"")

	focused := !m.focusOnProxyGroup
	for i, mode := range m.modes {
		switch {
		case focused && i == m.selectedMode:
			lines = append(lines, SelectedModeStyle.Render("➤ "+mode))
		case mode == m.currentMode:
			lines = append(lines, CurrentModeStyle.Render("✓ "+mode))
		default:
			lines = append(lines, ModeSelectorStyle.Render("  "+mode))
		}
	}

	lines = append(lines, "",
		lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray)).
			Render(fmt.Sprintf("Current: %s", m.currentMode)))

	return CardStyle(!m.focusOnProxyGroup, 30).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m *Model) renderProxyGroupCard() string {
	var lines []string
	lines = append(lines,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorPurple)).Render("🌐 Proxy Groups"),
		"")

	if len(m.proxyGroupList) == 0 {
		lines = append(lines,
			lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray)).Render("No groups available"))
	} else {
		group := m.proxyGroupList[m.selectedGroupIndex]
		lines = append(lines,
			lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorOrange)).Render(group.Name),
			lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray)).
				Render(fmt.Sprintf("Current: %s", group.Now)),
			"")

		const maxVisible = 10
		for i, proxy := range group.All {
			if i >= maxVisible {
				lines = append(lines,
					lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray)).
						Render(fmt.Sprintf("  ... and %d more", len(group.All)-maxVisible)))
				break
			}
			isCurrent := proxy == group.Now
			switch {
			case m.focusOnProxyGroup && i == m.selectedProxyIndex:
				prefix := "➤ "
				if isCurrent {
					prefix = "➤ ✓ "
				}
				lines = append(lines, SelectedModeStyle.Render(prefix+proxy))
			case isCurrent:
				lines = append(lines, CurrentModeStyle.Render("  ✓ "+proxy))
			default:
				lines = append(lines, ModeSelectorStyle.Render("    "+proxy))
			}
		}

		if len(m.proxyGroupList) > 1 {
			lines = append(lines, "")
			var names []string
			for i, g := range m.proxyGroupList {
				if i == m.selectedGroupIndex {
					names = append(names, lipgloss.NewStyle().
						Foreground(lipgloss.Color(ColorGreen)).Bold(true).
						Render("["+g.Name+"]"))
				} else {
					names = append(names, lipgloss.NewStyle().
						Foreground(lipgloss.Color(ColorGray)).Render(g.Name))
				}
			}
			lines = append(lines, strings.Join(names, " • "))
		}
	}

	return CardStyle(m.focusOnProxyGroup, 50).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))
}

func (m *Model) renderModeMenuHelp() string {
	parts := []string{"Tab: switch focus", "↑/↓: navigate"}
	if m.focusOnProxyGroup && len(m.proxyGroupList) > 1 {
		parts = append(parts, "←/→: switch group")
	}
	parts = append(parts, "Enter: confirm", "q/Esc: cancel")
	return lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray)).
		Render(strings.Join(parts, " • "))
}

// ---------- Error popup ----------

func (m *Model) renderErrorPopup(msg string) string {
	width := lipgloss.Width(msg) + 6
	width = max(30, min(80, width))

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorRed)).
		Padding(1, 2).
		Width(width)

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(ColorRed)).Render("Error")
	body := PayloadStyle.Render(msg)
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray)).Render("Press Enter/Esc to close")
	popup := box.Render(lipgloss.JoinVertical(lipgloss.Left, title, "", body, "", hint))

	padX := max(0, (m.terminalWidth-lipgloss.Width(popup))/2)
	padY := max(0, (m.terminalHeight-lipgloss.Height(popup))/2)
	return lipgloss.NewStyle().Padding(padY, padX).Render(popup)
}

// ---------- Helpers ----------

func cardTitle(text string) string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(ColorPurple)).
		Width(15).
		Render(text)
}

func statusRow(label, value string) string {
	return fmt.Sprintf("%-10s %s",
		StatusLabelStyle.Render(label),
		StatusValueStyle.Render(value))
}

func formatBytes(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func formatSpeed(bps int64) string {
	return formatBytes(bps) + "/s"
}
