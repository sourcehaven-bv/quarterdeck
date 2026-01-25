package ui

import (
	"strings"
	"testing"
)

func TestNewOutputModel(t *testing.T) {
	model := NewOutputModel()

	if len(model.lines) != 0 {
		t.Errorf("expected empty lines, got %d", len(model.lines))
	}
	if model.IsRunning() {
		t.Error("expected IsRunning to be false")
	}
}

func TestOutputModel_SetSize(t *testing.T) {
	model := NewOutputModel()
	model = model.SetSize(80, 24)

	if model.viewport.Width != 80 {
		t.Errorf("expected width 80, got %d", model.viewport.Width)
	}
	// Height is reduced by 1 for status line
	if model.viewport.Height != 23 {
		t.Errorf("expected height 23, got %d", model.viewport.Height)
	}
}

func TestOutputModel_AppendLine(t *testing.T) {
	model := NewOutputModel()
	model = model.SetSize(80, 10)

	model = model.AppendLine("line 1")
	model = model.AppendLine("line 2")
	model = model.AppendLine("line 3")

	if len(model.lines) != 3 {
		t.Errorf("expected 3 lines, got %d", len(model.lines))
	}

	if model.lines[0] != "line 1" {
		t.Errorf("expected 'line 1', got %q", model.lines[0])
	}
}

func TestOutputModel_AppendLine_MaxLines(t *testing.T) {
	model := NewOutputModel()
	model = model.SetSize(80, 10)

	// Add more than MaxOutputLines
	for i := 0; i < MaxOutputLines+100; i++ {
		model = model.AppendLine("test line")
	}

	if len(model.lines) != MaxOutputLines {
		t.Errorf("expected %d lines (max), got %d", MaxOutputLines, len(model.lines))
	}
}

func TestOutputModel_SetDone(t *testing.T) {
	model := NewOutputModel()
	model = model.SetRunning(true)

	if !model.IsRunning() {
		t.Error("expected IsRunning to be true")
	}

	model = model.SetDone(true)

	if model.IsRunning() {
		t.Error("expected IsRunning to be false after SetDone")
	}
}

func TestOutputModel_Clear(t *testing.T) {
	model := NewOutputModel()
	model = model.SetSize(80, 10)
	model = model.AppendLine("test line")
	model = model.SetDone(true)

	if len(model.lines) != 1 {
		t.Errorf("expected 1 line, got %d", len(model.lines))
	}

	model = model.Clear()

	if len(model.lines) != 0 {
		t.Errorf("expected 0 lines after clear, got %d", len(model.lines))
	}
	if model.IsRunning() {
		t.Error("expected IsRunning to be false after Clear")
	}
	if model.done {
		t.Error("expected done to be false after Clear")
	}
	// Viewport dimensions should be preserved
	if model.viewport.Width != 80 {
		t.Errorf("expected width preserved at 80, got %d", model.viewport.Width)
	}
}

func TestOutputModel_View_Empty(t *testing.T) {
	model := NewOutputModel()
	view := model.View()

	if !strings.Contains(view, "Press Enter") {
		t.Error("expected empty view to contain help text")
	}
}

func TestOutputModel_View_WithContent(t *testing.T) {
	model := NewOutputModel()
	model = model.SetSize(80, 10)
	model = model.SetRunning(true)
	model = model.AppendLine("test output line")

	view := model.View()

	if !strings.Contains(view, "test output line") {
		t.Error("expected view to contain output line")
	}
	if !strings.Contains(view, "Running") {
		t.Error("expected view to show running status")
	}
}

func TestOutputModel_View_Done(t *testing.T) {
	model := NewOutputModel()
	model = model.SetSize(80, 10)
	model = model.AppendLine("test output")
	model = model.SetDone(true)

	view := model.View()

	// Done status is no longer displayed, but content should still show
	if !strings.Contains(view, "test output") {
		t.Error("expected view to contain output after done")
	}
	// Should not show running status
	if strings.Contains(view, "Running") {
		t.Error("expected view to not show running status after done")
	}
}

// TestOutputMsg_CommandGeneration tests that OutputMsg tracks command IDs.
// This is the core bug fix: when navigating between menu items,
// messages from old commands should not appear in the new output pane.
func TestOutputMsg_CommandGeneration(t *testing.T) {
	// Create a message with command ID 1
	msg1 := NewOutputMsg("output from command 1", 1)
	if msg1.CommandID() != 1 {
		t.Errorf("expected command ID 1, got %d", msg1.CommandID())
	}
	if msg1.Text() != "output from command 1" {
		t.Errorf("expected text 'output from command 1', got %q", msg1.Text())
	}

	// Create a message with explicit command ID
	msg2 := NewOutputMsg("output from command 2", 42)
	if msg2.CommandID() != 42 {
		t.Errorf("expected command ID 42, got %d", msg2.CommandID())
	}
	if msg2.Text() != "output from command 2" {
		t.Errorf("expected text 'output from command 2', got %q", msg2.Text())
	}
}

// TestOutputDoneMsg_CommandGeneration tests that OutputDoneMsg tracks command ID
func TestOutputDoneMsg_CommandGeneration(t *testing.T) {
	msg := NewOutputDoneMsg(123)
	if msg.CommandID != 123 {
		t.Errorf("expected command ID 123, got %d", msg.CommandID)
	}
}

// TestOutputMsg_DifferentCommandIDs verifies that messages with different command IDs
// can be distinguished. This is essential for filtering stale output.
func TestOutputMsg_DifferentCommandIDs(t *testing.T) {
	// Simulate multiple commands being started
	cmd1Output := NewOutputMsg("ping localhost", 1)
	cmd2Output := NewOutputMsg("ps aux", 2)
	cmd1More := NewOutputMsg("64 bytes from localhost", 1)

	// All messages should have their correct command IDs
	if cmd1Output.CommandID() != 1 {
		t.Errorf("cmd1Output should have ID 1, got %d", cmd1Output.CommandID())
	}
	if cmd2Output.CommandID() != 2 {
		t.Errorf("cmd2Output should have ID 2, got %d", cmd2Output.CommandID())
	}
	if cmd1More.CommandID() != 1 {
		t.Errorf("cmd1More should have ID 1, got %d", cmd1More.CommandID())
	}

	// Simulate filtering: only accept messages matching current command ID
	currentCommandID := uint64(2)
	acceptedMessages := []OutputMsg{}

	for _, msg := range []OutputMsg{cmd1Output, cmd2Output, cmd1More} {
		if msg.CommandID() == currentCommandID {
			acceptedMessages = append(acceptedMessages, msg)
		}
	}

	// Only cmd2Output should be accepted
	if len(acceptedMessages) != 1 {
		t.Errorf("expected 1 accepted message, got %d", len(acceptedMessages))
	}
	if len(acceptedMessages) > 0 && acceptedMessages[0].Text() != "ps aux" {
		t.Errorf("expected 'ps aux', got %q", acceptedMessages[0].Text())
	}
}
