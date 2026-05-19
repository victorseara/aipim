package tui

import "github.com/charmbracelet/lipgloss"

var (
	primaryColor = lipgloss.Color("#7C3AED")
	mutedColor   = lipgloss.Color("#6B7280")
	successColor = lipgloss.Color("#10B981")
	dangerColor  = lipgloss.Color("#EF4444")
	borderColor  = lipgloss.Color("#374151")

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(primaryColor).
			MarginBottom(1)

	profileItemStyle = lipgloss.NewStyle().
				PaddingLeft(2)

	selectedItemStyle = profileItemStyle.Copy().
				Foreground(primaryColor).
				Bold(true)

	mutedStyle = lipgloss.NewStyle().Foreground(mutedColor)

	helpTextStyle = lipgloss.NewStyle().
			Foreground(mutedColor).
			MarginTop(1)

	panelStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(borderColor).
			Padding(1, 2)

	successStyle = lipgloss.NewStyle().Foreground(successColor).Bold(true)
	dangerStyle  = lipgloss.NewStyle().Foreground(dangerColor).Bold(true)
)
