package ui

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/guohao117/clashtui/internal/api"
)

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		cmds = append(cmds, m.handleKey(msg))

	case tea.WindowSizeMsg:
		m.handleResize(msg)

	case logMsg:
		m.logs = append(m.logs, msg.logs...)
		if len(m.logs) > maxLogs {
			m.logs = m.logs[len(m.logs)-maxLogs:]
		}
		cmds = append(cmds, m.waitForLogs())

	case statusMsg:
		if msg.err == nil {
			m.connected = true
			m.activeConnections = msg.connections.ActiveCount
			m.totalTraffic = msg.connections.TotalBytes
			m.uploadSpeed = msg.traffic.Up
			m.downloadSpeed = msg.traffic.Down
		} else {
			m.connected = false
		}
		cmds = append(cmds, m.tickStatus())
	}

	if m.ready {
		m.viewport.SetContent(m.renderLogs())
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// handleKey dispatches a key event to the appropriate handler.
func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	// Error popup: dismiss on enter/esc
	if m.err != nil {
		switch msg.String() {
		case "enter", "esc":
			m.err = nil
		}
		return nil
	}

	if m.inModeMenu {
		return m.handleModeMenuKey(msg)
	}
	return m.handleMainKey(msg)
}

// handleMainKey processes keys on the main dashboard view.
func (m *Model) handleMainKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "q", "ctrl+c":
		m.cancel()
		return tea.Quit
	case "m":
		m.inModeMenu = true
		m.focusOnProxyGroup = false
		for i, mode := range m.modes {
			if mode == m.currentMode {
				m.selectedMode = i
				break
			}
		}
		m.fetchProxyGroups()
	case "tab", "1", "2":
		switch msg.String() {
		case "tab":
			if m.activeTab == TabDashboard {
				m.activeTab = TabLogs
			} else {
				m.activeTab = TabDashboard
			}
		case "1":
			m.activeTab = TabDashboard
		case "2":
			m.activeTab = TabLogs
		}
	case "d":
		m.toggleFilter("debug")
	case "i":
		m.toggleFilter("info")
	case "w":
		m.toggleFilter("warning")
	case "e":
		m.toggleFilter("error")
	case "a":
		m.filters = nil
	}
	return nil
}

// handleModeMenuKey processes keys while the mode/proxy menu is open.
func (m *Model) handleModeMenuKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "q", "esc":
		m.inModeMenu = false
	case "tab":
		m.focusOnProxyGroup = !m.focusOnProxyGroup
		if m.focusOnProxyGroup {
			m.selectedProxyIndex = 0
		}
	case "up", "k":
		if m.focusOnProxyGroup {
			if m.selectedProxyIndex > 0 {
				m.selectedProxyIndex--
			}
		} else if m.selectedMode > 0 {
			m.selectedMode--
		}
	case "down", "j":
		if m.focusOnProxyGroup {
			if len(m.proxyGroupList) > 0 && m.selectedGroupIndex < len(m.proxyGroupList) {
				group := m.proxyGroupList[m.selectedGroupIndex]
				if m.selectedProxyIndex < len(group.All)-1 {
					m.selectedProxyIndex++
				}
			}
		} else if m.selectedMode < len(m.modes)-1 {
			m.selectedMode++
		}
	case "left", "h":
		if m.focusOnProxyGroup && m.selectedGroupIndex > 0 {
			m.selectedGroupIndex--
			m.selectedProxyIndex = 0
		}
	case "right", "l":
		if m.focusOnProxyGroup && m.selectedGroupIndex < len(m.proxyGroupList)-1 {
			m.selectedGroupIndex++
			m.selectedProxyIndex = 0
		}
	case "enter":
		if m.focusOnProxyGroup {
			m.confirmProxySelection()
		} else {
			m.confirmModeSelection()
		}
	}
	return nil
}

func (m *Model) confirmModeSelection() {
	newMode := m.modes[m.selectedMode]
	if newMode == m.currentMode {
		return
	}
	ctx := context.Background()
	if err := m.client.SwitchMode(ctx, newMode); err != nil {
		m.err = fmt.Errorf("switch mode: %w", err)
		return
	}
	m.currentMode = newMode
	m.fetchProxyGroups()
	m.selectedGroupIndex = 0
	m.selectedProxyIndex = 0
}

func (m *Model) confirmProxySelection() {
	if len(m.proxyGroupList) == 0 || m.selectedGroupIndex >= len(m.proxyGroupList) {
		return
	}
	group := m.proxyGroupList[m.selectedGroupIndex]
	if m.selectedProxyIndex >= len(group.All) {
		return
	}
	if strings.EqualFold(m.currentMode, "direct") {
		return
	}

	proxy := group.All[m.selectedProxyIndex]
	ctx := context.Background()
	if err := m.client.SwitchProxy(ctx, group.Name, proxy); err != nil {
		m.err = fmt.Errorf("switch proxy: %w", err)
		return
	}
	m.proxyGroupList[m.selectedGroupIndex].Now = proxy
}

func (m *Model) fetchProxyGroups() {
	ctx := context.Background()
	groups, err := m.client.FetchProxyGroups(ctx, m.currentMode)
	if err != nil {
		m.err = fmt.Errorf("fetch proxy groups: %w", err)
		return
	}
	m.proxyGroupList = groups
}

func (m *Model) handleResize(msg tea.WindowSizeMsg) {
	m.terminalWidth = msg.Width
	m.terminalHeight = msg.Height

	margin := m.calculateMargin()
	vpWidth := msg.Width - margin*2

	if !m.ready {
		vpHeight := m.viewportHeight(msg.Height, margin)
		m.viewport = viewport.New(vpWidth, vpHeight)
		m.viewport.Style = LogContainerStyle
		m.ready = true
	} else {
		m.viewport.Width = vpWidth
		m.viewport.Height = m.viewportHeight(msg.Height, margin)
	}
}

func (m *Model) viewportHeight(termH, margin int) int {
	// Logs tab gets most of the height: top bar (1) + tab bar (~2) + title (~2) + footer (~2)
	return max(4, termH-margin*2-7)
}

func (m *Model) toggleFilter(logType string) {
	for i, f := range m.filters {
		if f == logType {
			m.filters = append(m.filters[:i], m.filters[i+1:]...)
			return
		}
	}
	m.filters = append(m.filters, logType)
}

func (m *Model) isLogVisible(log api.ClashLog) bool {
	if len(m.filters) == 0 {
		return true
	}
	for _, f := range m.filters {
		if strings.EqualFold(log.Type, f) {
			return true
		}
	}
	return false
}

func (m *Model) calculateMargin() int {
	switch {
	case m.terminalWidth <= 60:
		return 2
	case m.terminalWidth <= 100:
		return 4
	default:
		return 6
	}
}
