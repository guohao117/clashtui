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
)

type ClashLog struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
	Time    string `json:"time,omitempty"`
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
	}
}

func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
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

			req.Header.Add("Authorization", "Bearer "+authToken)
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
		switch msg.String() {
		case "q":
			close(m.done)
			return m, tea.Quit
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
	case tea.WindowSizeMsg:
		if !m.ready {
			m.viewport = viewport.New(msg.Width-4, msg.Height-10)
			m.viewport.Style = logContainerStyle
			m.ready = true
		} else {
			m.viewport.Width = msg.Width - 4
			m.viewport.Height = msg.Height - 10
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

	if m.err != nil {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")).
			Render(fmt.Sprintf("Error: %v\n", m.err))
	}

	header := headerStyle.Render(fmt.Sprintf("🔍 Clash Logs Monitor (%d logs)", len(m.logs)))
	filterStatus := filterStyle.Render(fmt.Sprintf("Filters: %s", m.getFilterStatus()))
	footer := footerStyle.Render("q: quit • d: debug • i: info • w: warning • e: error • a: all")

	return mainStyle.Render(lipgloss.JoinVertical(
		lipgloss.Left,
		header,
		filterStatus,
		m.viewport.View(),
		footer,
	))
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
