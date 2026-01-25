package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sourcehaven-bv/quarterdeck/internal/config"
)

const MenuWidth = 30

// MenuItem represents a flattened menu item with its category
type MenuItem struct {
	Name     string
	Category string
	Config   config.ItemConfig
}

// MenuModel handles the menu state
type MenuModel struct {
	items    []MenuItem
	cursor   int
	prevItem int // track previous cursor for auto-execute
}

func NewMenuModel(categories []config.CategoryConfig) MenuModel {
	var items []MenuItem
	for _, cat := range categories {
		for _, item := range cat.Items {
			items = append(items, MenuItem{
				Name:     item.Name,
				Category: cat.Category,
				Config:   item,
			})
		}
	}
	return MenuModel{
		items:    items,
		cursor:   0,
		prevItem: -1,
	}
}

// CursorChanged returns true if cursor moved since last check
func (m *MenuModel) CursorChanged() bool {
	changed := m.cursor != m.prevItem
	m.prevItem = m.cursor
	return changed
}

func (m MenuModel) Update(msg tea.Msg) (MenuModel, tea.Cmd) {
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.String() {
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
		case "home", "g":
			m.cursor = 0
		case "end", "G":
			m.cursor = len(m.items) - 1
		}
	}
	return m, nil
}

func (m MenuModel) View() string {
	s := ""

	currentCategory := ""
	for i, item := range m.items {
		// Show category header
		if item.Category != currentCategory {
			currentCategory = item.Category
			if i > 0 {
				s += "\n"
			}
			s += categoryStyle.Render(currentCategory) + "\n"
		}

		cursor := "  "
		if m.cursor == i {
			cursor = "> "
		}

		line := cursor + item.Name
		if m.cursor == i {
			s += selectedStyle.Render(line)
		} else {
			s += itemStyle.Render(line)
		}
		s += "\n"
	}

	return s
}

func (m MenuModel) SelectedItem() MenuItem {
	if m.cursor >= 0 && m.cursor < len(m.items) {
		return m.items[m.cursor]
	}
	return MenuItem{}
}

// GetActionByKey returns the action config for a given key, if any
func (m MenuModel) GetActionByKey(key string) *config.ActionConfig {
	item := m.SelectedItem()
	for _, action := range item.Config.Actions {
		if action.Key == key {
			return &action
		}
	}
	return nil
}

// Items returns all menu items
func (m MenuModel) Items() []MenuItem {
	return m.items
}

// Cursor returns the current cursor position
func (m MenuModel) Cursor() int {
	return m.cursor
}

// Styles
var (
	categoryStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true)

	itemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	selectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(true)
)
