package ui

import (
	"context"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/guohao117/clashtui/internal/api"
)

const maxLogs = 200

// Tab indices.
const (
	TabDashboard = iota
	TabLogs
)

// logMsg carries a batch of logs from the streaming goroutine to the TUI.
type logMsg struct {
	logs []api.ClashLog
}

// statusMsg carries periodically refreshed status data.
type statusMsg struct {
	connections api.ConnectionsSummary
	traffic     api.TrafficSnapshot
	err         error
}

// Model is the Bubble Tea model for the Clash TUI dashboard.
type Model struct {
	// API
	client *api.Client
	cancel context.CancelFunc
	logCh  chan []api.ClashLog

	// Logs
	logs     []api.ClashLog
	filters  []string
	viewport viewport.Model
	ready    bool

	// Mode menu
	currentMode  string
	modes        []string
	selectedMode int
	inModeMenu   bool

	// Proxy group
	proxyGroupList     []api.ProxyGroup
	selectedGroupIndex int
	selectedProxyIndex int
	focusOnProxyGroup  bool

	// Dashboard status
	connected bool
	startTime time.Time
	activeTab int

	// Live stats
	activeConnections int
	uploadSpeed       int64
	downloadSpeed     int64
	totalTraffic      int64

	// Terminal size
	terminalWidth  int
	terminalHeight int

	// Error (shown as popup)
	err error
}

// NewModel creates the initial Model.
func NewModel(client *api.Client) *Model {
	ctx, cancel := context.WithCancel(context.Background())
	logCh := make(chan []api.ClashLog, 16)

	// Start log streaming in the background.
	go func() {
		_ = client.StreamLogs(ctx, logCh)
	}()

	return &Model{
		client: client,
		cancel: cancel,
		logCh:  logCh,

		logs: make([]api.ClashLog, 0, maxLogs),

		currentMode:  "Rule",
		modes:        []string{"Global", "Rule", "Direct"},
		selectedMode: 1,

		proxyGroupList: make([]api.ProxyGroup, 0),

		connected: true,
		startTime: time.Now(),
		activeTab: TabDashboard,
	}
}

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		m.waitForLogs(),
		m.tickStatus(),
	)
}

// waitForLogs returns a Cmd that waits for the next log batch from the channel.
func (m *Model) waitForLogs() tea.Cmd {
	ch := m.logCh
	return func() tea.Msg {
		batch, ok := <-ch
		if !ok {
			return nil
		}
		return logMsg{logs: batch}
	}
}

// tickStatus returns a Cmd that periodically fetches status data.
func (m *Model) tickStatus() tea.Cmd {
	client := m.client
	return tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		var msg statusMsg
		conns, err := client.FetchConnections(ctx)
		if err != nil {
			msg.err = err
			return msg
		}
		msg.connections = conns

		traffic, err := client.FetchTraffic(ctx)
		if err != nil {
			msg.err = err
			return msg
		}
		msg.traffic = traffic
		return msg
	})
}
