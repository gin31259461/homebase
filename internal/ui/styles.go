package ui

import "github.com/charmbracelet/lipgloss"

var (
	TitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	OKStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	WarnStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	BadStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	DimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)
