package ui

import "github.com/charmbracelet/lipgloss"

// Color palette based on Dracula theme.
const (
	ColorPurple     = "#7571F9"
	ColorGray       = "#6272A4"
	ColorOrange     = "#FFB86C"
	ColorRed        = "#FF5555"
	ColorGreen      = "#50FA7B"
	ColorForeground = "#F8F8F2"
	ColorDimGray    = "#A9A9A9"
	ColorBackground = "#1E1E2E"
)

// Shared styles used across renderer components.
var (
	LogContainerStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color(ColorGray)).
				Padding(0, 1).
				MarginTop(1)

	TypeStyles = map[string]lipgloss.Style{
		"debug":   lipgloss.NewStyle().Foreground(lipgloss.Color(ColorDimGray)),
		"info":    lipgloss.NewStyle().Foreground(lipgloss.Color(ColorPurple)),
		"warning": lipgloss.NewStyle().Foreground(lipgloss.Color(ColorOrange)),
		"error":   lipgloss.NewStyle().Foreground(lipgloss.Color(ColorRed)),
	}

	TimeStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray)).Italic(true)
	PayloadStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorForeground))

	FooterStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorGray)).
			MarginTop(1).
			Align(lipgloss.Center)

	ModeSelectorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorForeground)).
				Padding(0, 1)

	SelectedModeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorGreen)).
				Bold(true).
				Padding(0, 1)

	CurrentModeStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorOrange)).
				Bold(true)

	TopBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(ColorForeground)).
			Background(lipgloss.Color(ColorBackground)).
			Padding(0, 1).
			Bold(true)

	ConnectedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGreen)).Bold(true)
	DisconnectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorRed)).Bold(true)

	StatusCardStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorPurple)).
			Padding(0, 1).
			Margin(0, 0, 1, 0)

	StatusValueStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGreen)).Bold(true)
	StatusLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(ColorGray))

	MainPanelStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorGray))

	LogPanelStyle = lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color(ColorGray))

	QuickActionStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorForeground)).
				Background(lipgloss.Color(ColorPurple)).
				Padding(0, 1).
				Margin(0, 1, 0, 0)

	ActiveTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color(ColorForeground)).
			Background(lipgloss.Color(ColorPurple)).
			Padding(0, 2)

	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(ColorGray)).
				Padding(0, 2)

	TabBarStyle = lipgloss.NewStyle().
			MarginBottom(1)
)

// CardStyle returns a bordered card style with the given focus state.
func CardStyle(focused bool, width int) lipgloss.Style {
	color := ColorGray
	if focused {
		color = ColorPurple
	}
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(color)).
		Padding(1, 2).
		Width(width)
}
