package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	clashHost = "http://127.0.0.1:9090"
	authToken = "your-secret-here"
)

type ClashLog struct {
	Type    string `json:"type"`
	Payload string `json:"payload"`
	Time    string `json:"time,omitempty"`
}

type Model struct {
	logs    []ClashLog
	err     error
	client  *http.Client
	done    chan struct{}
	program *tea.Program
}

type LogMsg struct {
	logs []ClashLog
}

// 定义样式
var (
	// 标题样式
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7571F9")).
			MarginBottom(1)

	// 日志类型样式
	typeStyles = map[string]lipgloss.Style{
		"debug":   lipgloss.NewStyle().Foreground(lipgloss.Color("#A9A9A9")), // 灰色
		"info":    lipgloss.NewStyle().Foreground(lipgloss.Color("#7571F9")), // 蓝色
		"warning": lipgloss.NewStyle().Foreground(lipgloss.Color("#FFB86C")), // 橙色
		"error":   lipgloss.NewStyle().Foreground(lipgloss.Color("#FF5555")), // 红色
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
			MarginTop(1)
)

func initialModel() *Model {
	return &Model{
		logs: make([]ClashLog, 0, 100),
		done: make(chan struct{}),
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
				// 使用 select 来处理超时
				select {
				case <-batchTimer.C:
					// 超时，发送当前批次
					if len(batch) > 0 {
						m.program.Send(LogMsg{logs: batch})
						batch = nil
					}
					batchTimer.Reset(batchTimeout)
				default:
					// 尝试读取一条日志
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
			// 发送剩余的日志
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
	// 不需要特殊的初始命令
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" {
			close(m.done)
			return m, tea.Quit
		}
	case LogMsg:
		m.logs = append(m.logs, msg.logs...)
		if len(m.logs) > 100 {
			m.logs = m.logs[len(m.logs)-100:]
		}
	}
	return m, nil
}

func (m Model) View() string {
	if m.err != nil {
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF5555")).
			Render(fmt.Sprintf("Error: %v\n", m.err))
	}

	// 标题
	s := titleStyle.Render(fmt.Sprintf("🔍 Clash Logs Monitor (%d logs)", len(m.logs)))
	s += "\n"

	// 日志内容
	for _, log := range m.logs {
		// 获取日志类型的样式，如果没有特定样式就使用默认样式
		typeStyle, ok := typeStyles[log.Type]
		if !ok {
			typeStyle = payloadStyle
		}

		// 格式化单条日志
		logLine := lipgloss.JoinHorizontal(
			lipgloss.Left,
			timeStyle.Render(fmt.Sprintf("[%s]", log.Time)),
			" ",
			typeStyle.Render(fmt.Sprintf("%-7s", log.Type)),
			" ",
			payloadStyle.Render(log.Payload),
		)
		s += logLine + "\n"
	}
	s += "\nPress q to quit\n"
	return s
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
