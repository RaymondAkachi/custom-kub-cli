package ui

import "github.com/charmbracelet/lipgloss"

// styles contains all the styling definitions for the UI
var styles = struct {
	TitleStyle     lipgloss.Style
	HeaderStyle    lipgloss.Style
	SelectedStyle  lipgloss.Style
	PromptStyle    lipgloss.Style
	ErrorStyle     lipgloss.Style
	SuccessStyle   lipgloss.Style
	InfoStyle      lipgloss.Style
	CommandStyle   lipgloss.Style
	OutputStyle    lipgloss.Style
	LoadingStyle   lipgloss.Style
	HelpStyle      lipgloss.Style
}{
	TitleStyle: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Bold(true),

	HeaderStyle: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true).
		Margin(1, 0),

	SelectedStyle: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1),

	PromptStyle: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Bold(true),

	ErrorStyle: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FF5F87")).
		Bold(true),

	SuccessStyle: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Bold(true),

	InfoStyle: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7C7C7C")),

	CommandStyle: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#FFFFFF")).
		Background(lipgloss.Color("#333333")).
		Padding(0, 1).
		Margin(0, 0, 1, 0),

	OutputStyle: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CCCCCC")).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#444444")).
		Padding(1).
		Margin(1, 0),

	LoadingStyle: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7D56F4")).
		Bold(true),

	HelpStyle: lipgloss.NewStyle().
		Foreground(lipgloss.Color("#CCCCCC")).
		MarginLeft(2),
}