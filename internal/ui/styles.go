package ui

import "github.com/charmbracelet/lipgloss"

// BorderColor returns the appropriate border color based on focus state
func BorderColor(focused bool) lipgloss.Color {
	if focused {
		return lipgloss.Color("205") // Pink/magenta when focused
	}
	return lipgloss.Color("240") // Gray when not focused
}
