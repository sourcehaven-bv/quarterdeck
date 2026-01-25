package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sourcehaven-bv/quarterdeck/internal/config"
)

func TestNewMenuModel(t *testing.T) {
	categories := []config.CategoryConfig{
		{
			Category: "System",
			Items: []config.ItemConfig{
				{Name: "Disk Usage", Run: "df -h"},
				{Name: "Processes", Run: "ps aux"},
			},
		},
		{
			Category: "Network",
			Items: []config.ItemConfig{
				{Name: "Connections", Run: "netstat -an"},
			},
		},
	}

	model := NewMenuModel(categories)

	if len(model.items) != 3 {
		t.Errorf("expected 3 items, got %d", len(model.items))
	}

	if model.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", model.cursor)
	}

	// Check first item
	if model.items[0].Name != "Disk Usage" {
		t.Errorf("expected first item 'Disk Usage', got %q", model.items[0].Name)
	}
	if model.items[0].Category != "System" {
		t.Errorf("expected category 'System', got %q", model.items[0].Category)
	}
}

func TestMenuModel_Navigation(t *testing.T) {
	categories := []config.CategoryConfig{
		{
			Category: "Test",
			Items: []config.ItemConfig{
				{Name: "Item 1", Run: "echo 1"},
				{Name: "Item 2", Run: "echo 2"},
				{Name: "Item 3", Run: "echo 3"},
			},
		},
	}

	model := NewMenuModel(categories)

	// Test moving down
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if model.cursor != 1 {
		t.Errorf("expected cursor at 1 after 'j', got %d", model.cursor)
	}

	// Test moving down with arrow
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	if model.cursor != 2 {
		t.Errorf("expected cursor at 2 after down arrow, got %d", model.cursor)
	}

	// Test boundary - can't go past last item
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if model.cursor != 2 {
		t.Errorf("expected cursor to stay at 2, got %d", model.cursor)
	}

	// Test moving up
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if model.cursor != 1 {
		t.Errorf("expected cursor at 1 after 'k', got %d", model.cursor)
	}

	// Test go to top with 'g'
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if model.cursor != 0 {
		t.Errorf("expected cursor at 0 after 'g', got %d", model.cursor)
	}

	// Test go to bottom with 'G'
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if model.cursor != 2 {
		t.Errorf("expected cursor at 2 after 'G', got %d", model.cursor)
	}

	// Test boundary - can't go above first item
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if model.cursor != 0 {
		t.Errorf("expected cursor to stay at 0, got %d", model.cursor)
	}
}

func TestMenuModel_SelectedItem(t *testing.T) {
	categories := []config.CategoryConfig{
		{
			Category: "Test",
			Items: []config.ItemConfig{
				{Name: "First", Run: "echo first"},
				{Name: "Second", Run: "echo second"},
			},
		},
	}

	model := NewMenuModel(categories)

	item := model.SelectedItem()
	if item.Name != "First" {
		t.Errorf("expected 'First', got %q", item.Name)
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	item = model.SelectedItem()
	if item.Name != "Second" {
		t.Errorf("expected 'Second', got %q", item.Name)
	}
}

func TestMenuModel_GetActionByKey(t *testing.T) {
	categories := []config.CategoryConfig{
		{
			Category: "Test",
			Items: []config.ItemConfig{
				{
					Name: "Test Item",
					Run:  "echo test",
					Actions: []config.ActionConfig{
						{Key: "r", Name: "Reload", Run: "echo reload"},
						{Key: "o", Name: "Open", Open: "https://example.com"},
					},
				},
			},
		},
	}

	model := NewMenuModel(categories)

	// Test finding existing action
	action := model.GetActionByKey("r")
	if action == nil {
		t.Fatal("expected to find action 'r'")
	}
	if action.Name != "Reload" {
		t.Errorf("expected action name 'Reload', got %q", action.Name)
	}

	// Test finding another action
	action = model.GetActionByKey("o")
	if action == nil {
		t.Fatal("expected to find action 'o'")
	}
	if action.Open != "https://example.com" {
		t.Errorf("expected action open URL, got %q", action.Open)
	}

	// Test non-existent action
	action = model.GetActionByKey("x")
	if action != nil {
		t.Error("expected nil for non-existent action")
	}
}

func TestMenuModel_CursorChanged(t *testing.T) {
	categories := []config.CategoryConfig{
		{
			Category: "Test",
			Items: []config.ItemConfig{
				{Name: "Item 1", Run: "echo 1"},
				{Name: "Item 2", Run: "echo 2"},
			},
		},
	}

	model := NewMenuModel(categories)

	// First call should return true (initial state)
	if !model.CursorChanged() {
		t.Error("expected CursorChanged to return true on first call")
	}

	// Second call without movement should return false
	if model.CursorChanged() {
		t.Error("expected CursorChanged to return false without movement")
	}

	// Move cursor
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Should return true after movement
	if !model.CursorChanged() {
		t.Error("expected CursorChanged to return true after movement")
	}
}

func TestMenuModel_View(t *testing.T) {
	categories := []config.CategoryConfig{
		{
			Category: "System",
			Items: []config.ItemConfig{
				{Name: "Disk Usage", Run: "df -h"},
			},
		},
	}

	model := NewMenuModel(categories)
	view := model.View()

	// Check that view contains expected elements
	if view == "" {
		t.Error("expected non-empty view")
	}

	// View should contain the category name
	if !strings.Contains(view, "System") {
		t.Error("expected view to contain category name")
	}

	// View should contain the item name
	if !strings.Contains(view, "Disk Usage") {
		t.Error("expected view to contain item name")
	}
}
