package app

import (
	"testing"
	"time"

	"github.com/sourcehaven-bv/quarterdeck/internal/config"
)

// TestIntervalStopsWhenActionExecuted verifies that when an action is executed
// via its shortcut key, the interval for the parent menu item stops.
// This prevents the action's output from being overwritten by the interval.
//
// BUG: When an action is executed, the interval continues running and
// overwrites the action's output when the next tick fires.
func TestIntervalStopsWhenActionExecuted(t *testing.T) {
	// Create a config with an item that has an interval and an action
	cfg := &config.Config{
		Menu: []config.CategoryConfig{
			{
				Category: "Test",
				Items: []config.ItemConfig{
					{
						Name:     "Test Item",
						Run:      "echo item",
						Mode:     config.ModeAuto,
						Interval: 5 * time.Second,
						Actions: []config.ActionConfig{
							{
								Key:  "a",
								Name: "Action A",
								Run:  "echo action",
							},
						},
					},
				},
			},
		},
	}

	// Create the model
	m := initialModel(cfg, nil)
	// Initialize to trigger auto-execute
	m.Init()

	// Verify interval is active after initialization
	if m.activeInterval != 5*time.Second {
		t.Errorf("expected activeInterval to be 5s after init, got %v", m.activeInterval)
	}
	if m.intervalItemID != 0 {
		t.Errorf("expected intervalItemID to be 0 after init, got %d", m.intervalItemID)
	}

	// Simulate the item's command completing (so action can run)
	m.output = m.output.SetDone(true)

	// Execute an action by pressing 'a' via handleActionKey (the real entry point)
	result, _, handled := m.handleActionKey("a")
	if !handled {
		t.Fatal("expected action 'a' to be handled")
	}
	// Update m from the result (handleActionKey returns tea.Model which is *Model)
	m = *result.(*Model)

	// The bug: interval should be stopped after executing an action
	// Expected: activeInterval == 0 and intervalItemID == -1
	// Actual (bug): interval remains active
	if m.activeInterval != 0 {
		t.Errorf("BUG: expected activeInterval to be 0 after action, got %v (interval should stop)", m.activeInterval)
	}
	if m.intervalItemID != -1 {
		t.Errorf("BUG: expected intervalItemID to be -1 after action, got %d (interval should stop)", m.intervalItemID)
	}
}

// TestIntervalRestartsWhenEnterPressed verifies that after an action stops
// the interval, pressing Enter on the item restarts the interval.
func TestIntervalRestartsWhenEnterPressed(t *testing.T) {
	// Create a config with an item that has an interval and an action
	cfg := &config.Config{
		Menu: []config.CategoryConfig{
			{
				Category: "Test",
				Items: []config.ItemConfig{
					{
						Name:     "Test Item",
						Run:      "echo item",
						Mode:     config.ModeDefault, // Not auto - requires Enter to run
						Interval: 5 * time.Second,
						Actions: []config.ActionConfig{
							{
								Key:  "a",
								Name: "Action A",
								Run:  "echo action",
							},
						},
					},
				},
			},
		},
	}

	// Create the model (interval not started because mode is not auto)
	m := initialModel(cfg, nil)
	m.Init()

	// Initially interval should not be active (mode is not auto)
	if m.activeInterval != 0 {
		t.Errorf("expected activeInterval to be 0 initially, got %v", m.activeInterval)
	}

	// Simulate pressing Enter to start the item's command
	// handleEnterKey should start the interval
	m.output = m.output.SetDone(true) // simulate previous command done
	_, cmd := m.handleEnterKey()

	// Interval should now be active
	if m.activeInterval != 5*time.Second {
		t.Errorf("expected activeInterval to be 5s after Enter, got %v", m.activeInterval)
	}

	// Verify that a command was returned (meaning handleEnterKey ran the command)
	if cmd == nil {
		t.Error("expected command to be returned from handleEnterKey")
	}
}
