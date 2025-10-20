package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	clashHost = "http://127.0.0.1:9090"
	authToken = "your-secret-here"
)

var (
	// 主容器样式
	mainStyle = lipgloss.NewStyle().
			Padding(1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7571F9"))

	// 标题容器样式
	headerStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7571F9")).
			Padding(0, 1).
			MarginBottom(1).
			Border(lipgloss.Border{
			Top:         "─",
			Bottom:      "─",
			Left:        "│",
			Right:       "│",
			TopLeft:     "╭",
			TopRight:    "╮",
			BottomLeft:  "╰",
			BottomRight: "╯",
		})

	// 日志容器样式
	logContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#6272A4")).
				Padding(0, 1).
				MarginTop(1)

	// 日志类型样式
	typeStyles = map[string]lipgloss.Style{
		"debug":   lipgloss.NewStyle().Foreground(lipgloss.Color("#A9A9A9")),
		"info":    lipgloss.NewStyle().Foreground(lipgloss.Color("#7571F9")),
		"warning": lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C")),
		"error":   lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")),
	}

	// 时间样式
	timeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			Italic(true)

	// 日志内容样式
	payloadStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2"))

	// 底部提示样式
	footerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			MarginTop(1).
			Align(lipgloss.Center)

	// 过滤器样式
	filterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB86C")).
			MarginBottom(1)

	// 工作模式菜单样式
	modeMenuStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7571F9")).
			Padding(1, 2).
			Margin(1)

	// 模式选择器样式
	modeSelectorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2")).
			Padding(0, 1)

	// 选中模式样式
	selectedModeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50FA7B")).
			Bold(true).
			Padding(0, 1)

	// 当前模式指示器样式
	currentModeStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB86C")).
			Bold(true)

	// 顶部状态栏样式
	topBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2")).
			Background(lipgloss.Color("#1E1E2E")).
			Padding(0, 1).
			Bold(true)

	// 连接状态样式
	connectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50FA7B")).
			Bold(true)

	disconnectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")).
			Bold(true)

	// 侧边栏样式
	sidebarStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#6272A4")).
			Padding(0, 1).
			Width(20)

	// 状态卡片样式
	statusCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7571F9")).
			Padding(0, 1).
			Margin(0, 0, 1, 0)

	// 主面板样式
	mainPanelStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#6272A4"))

	// 底部日志面板样式
	logPanelStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#6272A4"))

	// 快速操作按钮样式
	quickActionStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#F8F8F2")).
			Background(lipgloss.Color("#7571F9")).
			Padding(0, 1).
			Margin(0, 1, 0, 0)

	// 状态值样式
	statusValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#50FA7B")).
			Bold(true)

	// 状态标签样式
	statusLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4"))
)

type ClashLog struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
	Time    string `json:"time,omitempty"`
}

type ProxyGroup struct {
	Name    string   `json:"name"`
	Type    string   `json:"type"`
	Now     string   `json:"now"`
	All     []string `json:"all"`
}

type Model struct {
	logs     []ClashLog
	err      error
	client   *http.Client
	done     chan struct{}
	program  *tea.Program
	viewport viewport.Model
	filters  []string
	ready    bool

	// 工作模式相关字段
	currentMode    string
	modes          []string
	selectedMode   int
	inModeMenu     bool

	// Proxy Group相关字段
	proxyGroupList     []ProxyGroup
	selectedGroupIndex int
	selectedProxyIndex int
	inProxyGroupMenu   bool
	focusOnProxyGroup  bool // 焦点是否在proxy group卡片上

	// Dashboard相关字段
	connected      bool
	startTime      time.Time
	showSidebar    bool
	showLogsPanel  bool

	// 状态信息字段
	activeConnections int
	totalConnections  int
	memoryUsage       string
	proxyGroups       int
	proxyNodes        int
	uploadSpeed       string
	downloadSpeed     string

	// 终端尺寸字段
	terminalWidth  int
	terminalHeight int
	totalTraffic      string
}

type LogMsg struct {
	logs []ClashLog
}

func initialModel() *Model {
	return &Model{
		logs:    make([]ClashLog, 0, 100),
		done:    make(chan struct{}),
		filters: []string{},
		client: &http.Client{
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout: 5 * time.Second,
				}).DialContext,
			},
		},
		// 初始化工作模式相关字段
		currentMode:  "Rule",
		modes:        []string{"Global", "Rule", "Direct"},
		selectedMode: 1, // 默认选中Rule模式
		inModeMenu:   false,
		// 初始化Proxy Group相关字段
		proxyGroupList:     make([]ProxyGroup, 0),
		selectedGroupIndex: 0,
		selectedProxyIndex: 0,
		inProxyGroupMenu:   false,
		focusOnProxyGroup:  false,
		// 初始化Dashboard相关字段
		connected:     true, // 假设初始连接成功
		startTime:     time.Now(),
		showSidebar:   false,
		showLogsPanel: true, // 默认显示日志面板
		// 初始化状态信息字段（模拟数据）
		activeConnections: 5,
		totalConnections:  42,
		memoryUsage:       "64MB",
		proxyGroups:       3,
		proxyNodes:        12,
		uploadSpeed:       "1.2MB/s",
		downloadSpeed:     "2.8MB/s",
		totalTraffic:      "15.6GB",
	}
}

func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

// 切换Clash工作模式
func (m *Model) switchClashMode(mode string) error {
	// Clash的配置API端点
	configURL := clashHost + "/configs"

	// 构建请求体
	configBody := fmt.Sprintf(`{"mode": "%s"}`, strings.ToLower(mode))

	req, err := http.NewRequest("PATCH", configURL, strings.NewReader(configBody))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Add("Authorization", "Bearer "+authToken)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("API返回错误状态码: %d", resp.StatusCode)
	}

	return nil
}

// 获取当前mode对应的proxy groups
func (m *Model) fetchProxyGroupsForMode() error {
	req, err := http.NewRequest("GET", clashHost+"/proxies", nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	if authToken != "" {
		req.Header.Add("Authorization", "Bearer "+authToken)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API返回错误状态码: %d", resp.StatusCode)
	}


	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}

	proxies, ok := result["proxies"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("无法解析proxies数据")
	}

	m.proxyGroupList = make([]ProxyGroup, 0)
	
	// 根据不同mode显示不同的proxy groups
	switch m.currentMode {
	case "Global":
		// Global模式：只显示GLOBAL组
		for name, data := range proxies {
			if strings.ToUpper(name) != "GLOBAL" {
				continue
			}
			proxyData, ok := data.(map[string]interface{})
			if !ok {
				continue
			}

			proxyType, _ := proxyData["type"].(string)
			if proxyType != "Selector" {
				continue
			}

			group := ProxyGroup{
				Name: name,
				Type: proxyType,
			}

			if now, ok := proxyData["now"].(string); ok {
				group.Now = now
			}

			if all, ok := proxyData["all"].([]interface{}); ok {
				group.All = make([]string, 0, len(all))
				for _, item := range all {
					if str, ok := item.(string); ok {
						group.All = append(group.All, str)
					}
				}
			}

			m.proxyGroupList = append(m.proxyGroupList, group)
		}
	case "Rule":
		// Rule模式：显示除GLOBAL以外的所有Selector组
		for name, data := range proxies {
			if strings.ToUpper(name) == "GLOBAL" {
				continue
			}
			
			proxyData, ok := data.(map[string]interface{})
			if !ok {
				continue
			}

			proxyType, _ := proxyData["type"].(string)
			if proxyType != "Selector" {
				continue
			}

			group := ProxyGroup{
				Name: name,
				Type: proxyType,
			}

			if now, ok := proxyData["now"].(string); ok {
				group.Now = now
			}

			if all, ok := proxyData["all"].([]interface{}); ok {
				group.All = make([]string, 0, len(all))
				for _, item := range all {
					if str, ok := item.(string); ok {
						group.All = append(group.All, str)
					}
				}
			}

			m.proxyGroupList = append(m.proxyGroupList, group)
		}
	case "Direct":
		// Direct模式：显示空的Direct组
		m.proxyGroupList = []ProxyGroup{
			{
				Name: "Direct",
				Type: "Direct",
				Now:  "DIRECT",
				All:  []string{"DIRECT"},
			},
		}
	}

	return nil
}

// 切换proxy group的选择
func (m *Model) switchProxyGroupSelection(groupName, proxyName string) error {
	url := fmt.Sprintf("%s/proxies/%s", clashHost, groupName)
	body := fmt.Sprintf(`{"name": "%s"}`, proxyName)

	req, err := http.NewRequest("PUT", url, strings.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %v", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if authToken != "" {
		req.Header.Add("Authorization", "Bearer "+authToken)
	}

	resp, err := m.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API返回错误状态码: %d", resp.StatusCode)
	}

	return nil
}

func (m *Model) fetchLogsRoutine() {
	var batch []ClashLog
	const batchSize = 10
	const batchTimeout = 500 * time.Millisecond

	for {
		select {
		case <-m.done:
			return
		default:
			req, err := http.NewRequest("GET", clashHost+"/logs", nil)
			if err != nil {
				time.Sleep(time.Second)
				continue
			}

			if authToken != "" {
				req.Header.Add("Authorization", "Bearer "+authToken)
			}
			resp, err := m.client.Do(req)
			if err != nil {
				time.Sleep(time.Second)
				continue
			}

			decoder := json.NewDecoder(resp.Body)
			batchTimer := time.NewTimer(batchTimeout)

			for {
				select {
				case <-batchTimer.C:
					if len(batch) > 0 {
						m.program.Send(LogMsg{logs: batch})
						batch = nil
					}
					batchTimer.Reset(batchTimeout)
				default:
					var log ClashLog
					if err := decoder.Decode(&log); err != nil {
						resp.Body.Close()
						if err != io.EOF {
							time.Sleep(time.Second)
						}
						goto NEXT_REQUEST
					}

					batch = append(batch, log)
					if len(batch) >= batchSize {
						m.program.Send(LogMsg{logs: batch})
						batch = nil
						batchTimer.Reset(batchTimeout)
					}
				}
			}
		NEXT_REQUEST:
			if len(batch) > 0 {
				m.program.Send(LogMsg{logs: batch})
				batch = nil
			}

			if !batchTimer.Stop() {
				<-batchTimer.C
			}
		}
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.inModeMenu {
			// Mode菜单中的键盘处理
			switch msg.String() {
			case "q", "esc":
				m.inModeMenu = false
			case "tab":
				// Tab切换焦点：mode列表 <-> proxy group列表
				m.focusOnProxyGroup = !m.focusOnProxyGroup
				if m.focusOnProxyGroup {
					m.selectedProxyIndex = 0
				}
			case "up", "k":
				if m.focusOnProxyGroup {
					// 在proxy列表中向上
					if m.selectedProxyIndex > 0 {
						m.selectedProxyIndex--
					}
				} else {
					// 在mode列表中向上
					if m.selectedMode > 0 {
						m.selectedMode--
					}
				}
			case "down", "j":
				if m.focusOnProxyGroup {
					// 在proxy列表中向下
					if len(m.proxyGroupList) > 0 && m.selectedGroupIndex < len(m.proxyGroupList) {
						currentGroup := m.proxyGroupList[m.selectedGroupIndex]
						if m.selectedProxyIndex < len(currentGroup.All)-1 {
							m.selectedProxyIndex++
						}
					}
				} else {
					// 在mode列表中向下
					if m.selectedMode < len(m.modes)-1 {
						m.selectedMode++
					}
				}
			case "left", "h":
				if m.focusOnProxyGroup {
					// 在proxy group之间向左切换
					if m.selectedGroupIndex > 0 {
						m.selectedGroupIndex--
						m.selectedProxyIndex = 0
					}
				}
			case "right", "l":
				if m.focusOnProxyGroup {
					// 在proxy group之间向右切换
					if m.selectedGroupIndex < len(m.proxyGroupList)-1 {
						m.selectedGroupIndex++
						m.selectedProxyIndex = 0
					}
				}
			case "enter":
				if m.focusOnProxyGroup {
					// 确认选择proxy
					if len(m.proxyGroupList) > 0 && m.selectedGroupIndex < len(m.proxyGroupList) {
						currentGroup := m.proxyGroupList[m.selectedGroupIndex]
						if m.selectedProxyIndex < len(currentGroup.All) {
							selectedProxy := currentGroup.All[m.selectedProxyIndex]
							// Direct模式不需要切换
							if m.currentMode != "Direct" {
								err := m.switchProxyGroupSelection(currentGroup.Name, selectedProxy)
								if err != nil {
									m.err = fmt.Errorf("切换代理失败: %v", err)
								} else {
									// 更新本地状态
									m.proxyGroupList[m.selectedGroupIndex].Now = selectedProxy
								}
							}
						}
					}
				} else {
					// 确认选择mode
					newMode := m.modes[m.selectedMode]
					if newMode != m.currentMode {
						// 调用Clash API切换模式
						err := m.switchClashMode(newMode)
						if err != nil {
							m.err = fmt.Errorf("切换模式失败: %v", err)
						} else {
							m.currentMode = newMode
							// 切换mode后，重新获取对应的proxy groups
							err := m.fetchProxyGroupsForMode()
							if err != nil {
								m.err = fmt.Errorf("获取代理组失败: %v", err)
							}
							m.selectedGroupIndex = 0
							m.selectedProxyIndex = 0
						}
					}
				}
			}
		} else {
			// 主界面中的键盘处理
			switch msg.String() {
			case "q":
				close(m.done)
				return m, tea.Quit
			case "m":
				m.inModeMenu = true
				m.focusOnProxyGroup = false
				// 设置选中模式为当前模式
				for i, mode := range m.modes {
					if mode == m.currentMode {
						m.selectedMode = i
						break
					}
				}
				// 获取当前mode对应的proxy groups
				err := m.fetchProxyGroupsForMode()
				if err != nil {
					m.err = fmt.Errorf("获取代理组失败: %v", err)
				}
			case "l":
				m.showLogsPanel = !m.showLogsPanel
				// 视口大小会在下一次窗口大小事件时自动调整
			case "s":
				m.showSidebar = !m.showSidebar
			case "r":
				// 重新加载状态（可以添加实际的重载逻辑）
				m.connected = true // 模拟重新连接
			case "d":
				m.toggleFilter("debug")
			case "i":
				m.toggleFilter("info")
			case "w":
				m.toggleFilter("warning")
			case "e":
				m.toggleFilter("error")
			case "a":
				m.filters = []string{}
			}
		}
	case tea.WindowSizeMsg:
		// 保存终端尺寸用于自适应布局
		m.terminalWidth = msg.Width
		m.terminalHeight = msg.Height

		// 计算自适应边距和尺寸
		margin := m.calculateMargin()
		logPanelRatio := m.calculateLogPanelRatio()

		if !m.ready {
			// 根据是否显示日志面板调整视口大小
			if m.showLogsPanel {
				m.viewport = viewport.New(msg.Width-margin*2, int(float64(msg.Height)*logPanelRatio))
			} else {
				m.viewport = viewport.New(msg.Width-margin*2, msg.Height-margin*3)
			}
			m.viewport.Style = logContainerStyle
			m.ready = true
		} else {
			// 动态调整视口大小
			if m.showLogsPanel {
				m.viewport.Width = msg.Width - margin*2
				m.viewport.Height = int(float64(msg.Height) * logPanelRatio)
			} else {
				m.viewport.Width = msg.Width - margin*2
				m.viewport.Height = msg.Height - margin*3
			}
		}
	case LogMsg:
		m.logs = append(m.logs, msg.logs...)
		if len(m.logs) > 100 {
			m.logs = m.logs[len(m.logs)-100:]
		}
	}

	if m.ready {
		m.viewport.SetContent(m.renderLogs())
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
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

func (m Model) renderLogs() string {
	var logLines []string
	for _, log := range m.logs {
		if !m.isLogVisible(log) {
			continue
		}

		typeStyle, ok := typeStyles[log.Type]
		if !ok {
			typeStyle = payloadStyle
		}

		logLine := lipgloss.JoinHorizontal(
			lipgloss.Left,
			timeStyle.Render(fmt.Sprintf("[%s]", log.Time)),
			" ",
			typeStyle.Render(fmt.Sprintf("%-7s", log.Type)),

			" ",
			payloadStyle.Render(log.Payload),
		)
		logLines = append(logLines, logLine)
	}
	return lipgloss.JoinVertical(lipgloss.Left, logLines...)
}

func (m Model) View() string {
	if !m.ready {
		return "Initializing..."
	}

	// 错误弹窗优先渲染，按 Esc/Enter 关闭
	if m.err != nil {
		popup := m.renderErrorPopup(fmt.Sprintf("%v", m.err))
		return popup
	}

	// 检查终端尺寸是否太小
	if m.isTerminalTooSmall() {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FFB86C")).
			Bold(true).
			Render(fmt.Sprintf("Terminal too small!\nMinimum size: 40x12\nCurrent: %dx%d",
				m.terminalWidth, m.terminalHeight))
	}

	if m.inModeMenu {
		return m.renderModeMenuWithProxyGroups()
	}

	// 顶部状态栏
	topBar := m.renderTopBar()

	// 主面板
	mainPanel := m.renderMainPanel()

	// 底部日志面板（可选）
	var logPanel string
	if m.showLogsPanel {
		logPanel = m.renderLogPanel()
	}

	// 底部提示栏
	footer := footerStyle.Render("q: quit • m: mode • l: logs • d/i/w/e: filters • a: all • r: reload")

	// 整体布局
	var content []string
	content = append(content, topBar)
	content = append(content, mainPanel)
	if m.showLogsPanel {
		content = append(content, logPanel)
	}
	content = append(content, footer)

	return lipgloss.JoinVertical(lipgloss.Left, content...)
}

func (m Model) getFilterStatus() string {
	if len(m.filters) == 0 {
		return "all"
	}
	return strings.Join(m.filters, ", ")
}

func (m Model) isLogVisible(log ClashLog) bool {
	if len(m.filters) == 0 {
		return true
	}
	for _, filter := range m.filters {
		if strings.EqualFold(log.Type, filter) {
			return true
		}
	}
	return false
}

// 辅助函数：返回两个整数中的最大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// 渲染错误弹窗（居中显示）
func (m Model) renderErrorPopup(msg string) string {
	// 计算内容宽度，限定范围
	contentWidth := lipgloss.Width(msg) + 6
	if contentWidth < 30 { contentWidth = 30 }
	if contentWidth > 80 { contentWidth = 80 }

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FF5555")).
		Padding(1, 2).
		Width(contentWidth)

	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FF5555")).Render("Error")
	body := payloadStyle.Render(msg)
	footer := lipgloss.NewStyle().Foreground(lipgloss.Color("#6272A4")).Render("Press Enter/Esc to close")
	content := lipgloss.JoinVertical(lipgloss.Left, title, "", body, "", footer)

	popup := box.Render(content)
	padX := 0
	padY := 0
	if m.terminalWidth > 0 {
		padX = max(0, (m.terminalWidth - lipgloss.Width(popup)) / 2)
	}
	if m.terminalHeight > 0 {
		padY = max(0, (m.terminalHeight - lipgloss.Height(popup)) / 2)
	}
	return lipgloss.NewStyle().Padding(padY, padX).Render(popup)
}

// 计算自适应边距
func (m Model) calculateMargin() int {
	// 根据终端宽度动态计算边距
	// 小终端：2字符边距
	// 中等终端：4字符边距
	// 大终端：6字符边距
	if m.terminalWidth <= 60 {
		return 2
	} else if m.terminalWidth <= 100 {
		return 4
	} else {
		return 6
	}
}

// 计算日志面板比例
func (m Model) calculateLogPanelRatio() float64 {
	// 根据终端高度动态计算日志面板比例
	// 小终端：40%高度
	// 中等终端：33%高度
	// 大终端：25%高度
	if m.terminalHeight <= 20 {
		return 0.4
	} else if m.terminalHeight <= 30 {
		return 0.33
	} else {
		return 0.25
	}
}

// 检查终端是否太小
func (m Model) isTerminalTooSmall() bool {
	// 最小终端尺寸要求
	return m.terminalWidth < 40 || m.terminalHeight < 12
}

// 渲染顶部状态栏
func (m Model) renderTopBar() string {
	// 连接状态
	var statusText string
	if m.connected {
		statusText = connectedStyle.Render("● Connected")
	} else {
		statusText = disconnectedStyle.Render("● Disconnected")
	}

	// 运行时间
	runtime := time.Since(m.startTime).Round(time.Second)
	runtimeText := fmt.Sprintf("⏱ %s", runtime)

	// 构建状态栏内容
	leftContent := "🚀 Clash Dashboard"
	rightContent := fmt.Sprintf("%s | %s", statusText, runtimeText)

	// 自适应对齐显示
	availableWidth := m.terminalWidth - 2 // 减去边距
	if availableWidth < 40 {
		// 终端太小，只显示关键信息
		return topBarStyle.Render("🚀 Clash")
	}

	// 计算合适的填充
	padding := max(0, availableWidth-lipgloss.Width(leftContent)-lipgloss.Width(rightContent))
	spacer := strings.Repeat(" ", padding)

	return topBarStyle.Render(leftContent + spacer + rightContent)
}

// 渲染主面板
func (m Model) renderMainPanel() string {
	var mainContent []string

	// 自适应状态卡片布局
	var statusCards string
	if m.terminalWidth < 80 {
		// 小终端：垂直布局
		statusCards = lipgloss.JoinVertical(
			lipgloss.Left,
			m.renderConnectionCard(),
			"",
			m.renderProxyCard(),
			"",
			m.renderTrafficCard(),
		)
	} else {
		// 正常终端：水平布局
		statusCards = lipgloss.JoinHorizontal(
			lipgloss.Top,
			m.renderConnectionCard(),
			"  ",
			m.renderProxyCard(),
			"  ",
			m.renderTrafficCard(),
		)
	}

	mainContent = append(mainContent, statusCards)
	mainContent = append(mainContent, "")

	// 快速操作行
	quickActions := m.renderQuickActions()
	mainContent = append(mainContent, quickActions)

	content := lipgloss.JoinVertical(lipgloss.Left, mainContent...)
	return mainPanelStyle.Render(content)
}

// 渲染连接状态卡片
func (m Model) renderConnectionCard() string {
	var lines []string

	// 标题行 - 确保所有卡片标题高度一致
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7571F9")).
		Width(15). // 固定宽度
		Align(lipgloss.Left).
		Render("Connection")
	lines = append(lines, title)

	// 内容行 - 使用固定格式
	lines = append(lines, fmt.Sprintf("%-8s %s",
		statusLabelStyle.Render("Active:"),
		statusValueStyle.Render(fmt.Sprintf("%d", m.activeConnections))))

	lines = append(lines, fmt.Sprintf("%-8s %s",
		statusLabelStyle.Render("Total:"),
		statusValueStyle.Render(fmt.Sprintf("%d", m.totalConnections))))

	lines = append(lines, fmt.Sprintf("%-8s %s",
		statusLabelStyle.Render("Memory:"),
		statusValueStyle.Render(m.memoryUsage)))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return statusCardStyle.Render(content)
}

// 渲染代理信息卡片
func (m Model) renderProxyCard() string {
	var lines []string

	// 标题行 - 确保所有卡片标题高度一致
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7571F9")).
		Width(15). // 固定宽度
		Align(lipgloss.Left).
		Render("Proxy")
	lines = append(lines, title)

	// 内容行 - 使用固定格式
	lines = append(lines, fmt.Sprintf("%-8s %s",
		statusLabelStyle.Render("Mode:"),
		statusValueStyle.Render(m.currentMode)))

	lines = append(lines, fmt.Sprintf("%-8s %s",
		statusLabelStyle.Render("Groups:"),
		statusValueStyle.Render(fmt.Sprintf("%d", m.proxyGroups))))

	lines = append(lines, fmt.Sprintf("%-8s %s",
		statusLabelStyle.Render("Nodes:"),
		statusValueStyle.Render(fmt.Sprintf("%d", m.proxyNodes))))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return statusCardStyle.Render(content)
}

// 渲染流量统计卡片
func (m Model) renderTrafficCard() string {
	var lines []string

	// 标题行 - 确保所有卡片标题高度一致
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7571F9")).
		Width(15). // 固定宽度
		Align(lipgloss.Left).
		Render("Traffic")
	lines = append(lines, title)

	// 内容行 - 使用固定格式
	lines = append(lines, fmt.Sprintf("%-8s %s",
		statusLabelStyle.Render("Upload:"),
		statusValueStyle.Render(m.uploadSpeed)))

	lines = append(lines, fmt.Sprintf("%-8s %s",
		statusLabelStyle.Render("Download:"),
		statusValueStyle.Render(m.downloadSpeed)))

	lines = append(lines, fmt.Sprintf("%-8s %s",
		statusLabelStyle.Render("Total:"),
		statusValueStyle.Render(m.totalTraffic)))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return statusCardStyle.Render(content)
}

// 渲染快速操作
func (m Model) renderQuickActions() string {
	actions := []string{
		quickActionStyle.Render("Global"),
		quickActionStyle.Render("Rule"),
		quickActionStyle.Render("Direct"),
		quickActionStyle.Render("Reload"),
		quickActionStyle.Render("Config"),
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, actions...)
}

// 渲染底部日志面板
func (m Model) renderLogPanel() string {
	var panelContent []string

	// 面板标题
	panelTitle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7571F9")).
		Render(fmt.Sprintf("🔍 Logs Panel (%d logs) [%s]", len(m.logs), m.getFilterStatus()))

	panelContent = append(panelContent, panelTitle)

	// 日志内容
	logContent := m.renderLogs()
	if logContent != "" {
		panelContent = append(panelContent, logContent)
	} else {
		panelContent = append(panelContent, "No logs to display")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, panelContent...)
	return logPanelStyle.Render(content)
}

// 渲染侧边栏
func (m Model) renderSidebar() string {
	var sidebarLines []string

	// 标题
	sidebarLines = append(sidebarLines, lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7571F9")).
		Render("📊 Status"))

	sidebarLines = append(sidebarLines, "")

	// 快速模式切换
	sidebarLines = append(sidebarLines, "🔄 Quick Mode:")
	for _, mode := range m.modes {
		var modeText string
		if mode == m.currentMode {
			modeText = selectedModeStyle.Render("✓ " + mode)
		} else {
			modeText = modeSelectorStyle.Render("  " + mode)
		}
		sidebarLines = append(sidebarLines, modeText)
	}

	sidebarLines = append(sidebarLines, "")

	// 统计信息
	sidebarLines = append(sidebarLines, "📈 Statistics:")
	sidebarLines = append(sidebarLines, fmt.Sprintf("  Logs: %d", len(m.logs)))
	sidebarLines = append(sidebarLines, fmt.Sprintf("  Filters: %d", len(m.filters)))

	content := lipgloss.JoinVertical(lipgloss.Left, sidebarLines...)
	return sidebarStyle.Render(content)
}

// 渲染Mode菜单和Proxy Group卡片（并排显示）
func (m Model) renderModeMenuWithProxyGroups() string {
	// 渲染左侧Mode选择卡片
	modeCard := m.renderModeCard()
	
	// 渲染右侧Proxy Group选择卡片
	proxyCard := m.renderProxyGroupCard()
	
	// 水平排列两个卡片
	content := lipgloss.JoinHorizontal(lipgloss.Top, modeCard, "  ", proxyCard)
	
	return lipgloss.NewStyle().
		Padding(2).
		Render(content)
}

// 渲染Mode选择卡片
func (m Model) renderModeCard() string {
	var modeLines []string

	// 标题
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7571F9")).
		Render("🔄 Clash Mode")

	modeLines = append(modeLines, title)
	modeLines = append(modeLines, "")

	// 模式选项
	for i, mode := range m.modes {
		var modeLine string
		isFocused := !m.focusOnProxyGroup
		isSelected := i == m.selectedMode
		
		if isFocused && isSelected {
			// 焦点在mode且被选中
			modeLine = selectedModeStyle.Render("➤ " + mode)
		} else if mode == m.currentMode {
			// 当前激活的mode
			modeLine = currentModeStyle.Render("✓ " + mode)
		} else {
			// 未选中的mode
			modeLine = modeSelectorStyle.Render("  " + mode)
		}
		modeLines = append(modeLines, modeLine)
	}

	modeLines = append(modeLines, "")

	// 当前模式提示
	currentModeInfo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6272A4")).
		Render(fmt.Sprintf("Current: %s", m.currentMode))
	modeLines = append(modeLines, currentModeInfo)

	content := lipgloss.JoinVertical(lipgloss.Left, modeLines...)
	
	// 根据焦点状态设置边框颜色
	borderColor := "#6272A4"
	if !m.focusOnProxyGroup {
		borderColor = "#7571F9"
	}
	
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(1, 2).
		Width(30)
		
	return cardStyle.Render(content)
}

// 渲染Proxy Group选择卡片
func (m Model) renderProxyGroupCard() string {
	var proxyLines []string

	// 标题
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7571F9")).
		Render("🌐 Proxy Groups")

	proxyLines = append(proxyLines, title)
	proxyLines = append(proxyLines, "")

	if len(m.proxyGroupList) == 0 {
		proxyLines = append(proxyLines, lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			Render("No groups available"))
	} else {
		// 显示当前选中的proxy group
		currentGroup := m.proxyGroupList[m.selectedGroupIndex]
		groupTitle := lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FFB86C")).
			Render(fmt.Sprintf("%s", currentGroup.Name))
		proxyLines = append(proxyLines, groupTitle)
		
		// 显示当前选择
		currentProxy := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			Render(fmt.Sprintf("Current: %s", currentGroup.Now))
		proxyLines = append(proxyLines, currentProxy)
		proxyLines = append(proxyLines, "")

		// 显示该group的所有proxy选项
		maxProxies := 10 // 最多显示10个
		for i, proxy := range currentGroup.All {
			if i >= maxProxies {
				proxyLines = append(proxyLines, lipgloss.NewStyle().
					Foreground(lipgloss.Color("#6272A4")).
					Render(fmt.Sprintf("  ... and %d more", len(currentGroup.All)-maxProxies)))
				break
			}
			
			var proxyLine string
			isFocused := m.focusOnProxyGroup
			isSelected := i == m.selectedProxyIndex
			isCurrentProxy := proxy == currentGroup.Now
			
			if isFocused && isSelected {
				// 焦点在proxy group且被选中
				if isCurrentProxy {
					proxyLine = selectedModeStyle.Render("➤ ✓ " + proxy)
				} else {
					proxyLine = selectedModeStyle.Render("➤ " + proxy)
				}
			} else if isCurrentProxy {
				// 当前激活的proxy
				proxyLine = currentModeStyle.Render("  ✓ " + proxy)
			} else {
				// 未选中的proxy
				proxyLine = modeSelectorStyle.Render("    " + proxy)
			}
			proxyLines = append(proxyLines, proxyLine)
		}

		proxyLines = append(proxyLines, "")

		// 显示其他proxy groups导航
		if len(m.proxyGroupList) > 1 {
			var groupNames []string
			for i, group := range m.proxyGroupList {
				if i == m.selectedGroupIndex {
					groupNames = append(groupNames, 
						lipgloss.NewStyle().
							Foreground(lipgloss.Color("#50FA7B")).
							Bold(true).
							Render("["+group.Name+"]"))
				} else {
					groupNames = append(groupNames, 
						lipgloss.NewStyle().
							Foreground(lipgloss.Color("#6272A4")).
							Render(group.Name))
				}
			}
			groupNav := lipgloss.NewStyle().
				Foreground(lipgloss.Color("#6272A4")).
				Render(strings.Join(groupNames, " • "))
			proxyLines = append(proxyLines, groupNav)
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Left, proxyLines...)
	
	// 根据焦点状态设置边框颜色
	borderColor := "#6272A4"
	if m.focusOnProxyGroup {
		borderColor = "#7571F9"
	}
	
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(borderColor)).
		Padding(1, 2).
		Width(50)
		
	return cardStyle.Render(content)
}

// 渲染帮助信息（放在底部）
func (m Model) renderModeMenuHelp() string {
	helpLines := []string{
		"Tab: switch focus",
		"↑/↓: navigate",
	}
	
	if m.focusOnProxyGroup && len(m.proxyGroupList) > 1 {
		helpLines = append(helpLines, "←/→: switch group")
	}
	
	helpLines = append(helpLines, "Enter: confirm", "q/Esc: cancel")
	
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6272A4")).
		Render(strings.Join(helpLines, " • "))
}

// 旧的单独渲染函数保留以防需要
func (m Model) renderModeMenu() string {
	var modeLines []string

	// 标题
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#7571F9")).
		Render("🔄 Select Clash Mode")

	modeLines = append(modeLines, title)
	modeLines = append(modeLines, "")

	// 模式选项
	for i, mode := range m.modes {
		var modeLine string
		if i == m.selectedMode {
			// 选中的模式
			modeLine = selectedModeStyle.Render("➤ " + mode)
		} else {
			// 未选中的模式
			modeLine = modeSelectorStyle.Render("  " + mode)
		}
		modeLines = append(modeLines, modeLine)
	}

	modeLines = append(modeLines, "")

	// 当前模式提示
	currentModeInfo := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#6272A4")).
		Render(fmt.Sprintf("Current: %s", m.currentMode))
	modeLines = append(modeLines, currentModeInfo)

	modeLines = append(modeLines, "")

	// 操作提示
	var helpText string
	if m.currentMode == "Rule" {
		helpText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			Render("↑/↓: select • Enter: confirm • Tab: proxy groups • q/esc: cancel")
	} else {
		helpText = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6272A4")).
			Render("↑/↓: select • Enter: confirm • q/esc: cancel")
	}
	modeLines = append(modeLines, helpText)

	content := lipgloss.JoinVertical(lipgloss.Left, modeLines...)
	return modeMenuStyle.Render(content)
}

func main() {
	m := initialModel()
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	m.SetProgram(p)
	go m.fetchLogsRoutine()

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error: %v", err)
		os.Exit(1)
	}
}
