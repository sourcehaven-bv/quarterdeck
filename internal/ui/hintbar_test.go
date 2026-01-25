package ui

import (
	"strings"
	"testing"

	"github.com/sourcehaven-bv/quarterdeck/internal/config"
)

func TestNewHintBar(t *testing.T) {
	hb := NewHintBar()
	if hb.width != 0 {
		t.Errorf("expected width 0, got %d", hb.width)
	}
}

func TestHintBar_SetWidth(t *testing.T) {
	hb := NewHintBar()
	hb = hb.SetWidth(100)

	if hb.width != 100 {
		t.Errorf("expected width 100, got %d", hb.width)
	}
}

func TestHintBar_View_BasicItem(t *testing.T) {
	hb := NewHintBar()
	hb = hb.SetWidth(100)

	item := config.ItemConfig{
		Name: "Test Item",
		Run:  "echo test",
	}

	view := hb.View(item)

	// Should show enter hint for non-auto items with run command
	if !strings.Contains(view, "enter") {
		t.Error("expected view to contain 'enter' hint")
	}
	if !strings.Contains(view, "run") {
		t.Error("expected view to contain 'run' hint")
	}
	// Should always show tab and quit hints
	if !strings.Contains(view, "tab") {
		t.Error("expected view to contain 'tab' hint")
	}
	if !strings.Contains(view, "quit") {
		t.Error("expected view to contain 'quit' hint")
	}
}

func TestHintBar_View_AutoItem(t *testing.T) {
	hb := NewHintBar()
	hb = hb.SetWidth(100)

	item := config.ItemConfig{
		Name: "Auto Item",
		Run:  "echo auto",
		Mode: config.ModeAuto,
	}

	view := hb.View(item)

	// Should NOT show enter hint for auto items
	if strings.Contains(view, "enter") && strings.Contains(view, "run") {
		t.Error("expected auto item to not show enter:run hint")
	}
	// Should still show tab and quit
	if !strings.Contains(view, "tab") {
		t.Error("expected view to contain 'tab' hint")
	}
}

func TestHintBar_View_WithActions(t *testing.T) {
	hb := NewHintBar()
	hb = hb.SetWidth(200)

	item := config.ItemConfig{
		Name: "Item With Actions",
		Run:  "echo test",
		Actions: []config.ActionConfig{
			{Key: "r", Name: "Reload"},
			{Key: "o", Name: "Open"},
		},
	}

	view := hb.View(item)

	// Should show action hints
	if !strings.Contains(view, "r") {
		t.Error("expected view to contain 'r' action key")
	}
	if !strings.Contains(view, "Reload") {
		t.Error("expected view to contain 'Reload' action name")
	}
	if !strings.Contains(view, "o") {
		t.Error("expected view to contain 'o' action key")
	}
	if !strings.Contains(view, "Open") {
		t.Error("expected view to contain 'Open' action name")
	}
}

func TestHintBar_View_OpenOnlyItem(t *testing.T) {
	hb := NewHintBar()
	hb = hb.SetWidth(100)

	item := config.ItemConfig{
		Name: "Open Item",
		Open: "https://example.com",
	}

	view := hb.View(item)

	// Should not show enter:run for open-only items
	if strings.Contains(view, "enter") && strings.Contains(view, "run") {
		t.Error("expected open-only item to not show enter:run hint")
	}
}

func TestHintBar_View_EmptyItem(t *testing.T) {
	hb := NewHintBar()
	hb = hb.SetWidth(100)

	item := config.ItemConfig{
		Name: "Empty Item",
	}

	view := hb.View(item)

	// Should still show basic hints
	if !strings.Contains(view, "tab") {
		t.Error("expected view to contain 'tab' hint")
	}
	if !strings.Contains(view, "quit") {
		t.Error("expected view to contain 'quit' hint")
	}
}
