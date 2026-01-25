package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sourcehaven-bv/quarterdeck/internal/config"
)

const HintBarHeight = 1

// HintBar renders contextual keybinding hints
type HintBar struct {
	width int
}

func NewHintBar() HintBar {
	return HintBar{}
}

func (h HintBar) SetWidth(w int) HintBar {
	h.width = w
	return h
}

func (h HintBar) View(item config.ItemConfig) string {
	style := lipgloss.NewStyle().
		Width(h.width).
		Background(lipgloss.Color("236")).
		Padding(0, 1)

	// Pre-allocate hints slice: actions + up to 3 default hints (enter, tab, quit)
	hints := make([]string, 0, len(item.Actions)+3)

	// Add action keybindings from the selected item
	for _, action := range item.Actions {
		hint := hintKeyStyle.Render(action.Key) + hintSepStyle.Render(":") + hintTextStyle.Render(action.Name)
		hints = append(hints, hint)
	}

	// Add default keybindings
	if item.Run != "" && item.Mode != config.ModeAuto {
		hints = append(hints, hintKeyStyle.Render("enter")+hintSepStyle.Render(":")+hintTextStyle.Render("run"))
	}
	hints = append(hints,
		hintKeyStyle.Render("tab")+hintSepStyle.Render(":")+hintTextStyle.Render("switch pane"),
		hintKeyStyle.Render("q")+hintSepStyle.Render(":")+hintTextStyle.Render("quit"),
	)

	return style.Render(strings.Join(hints, "    "))
}

var (
	hintKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Bold(true)

	hintSepStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	hintTextStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
)
