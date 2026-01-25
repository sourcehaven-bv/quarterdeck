package status

import (
	"strings"
	"testing"
	"time"

	"github.com/sourcehaven-bv/quarterdeck/internal/config"
)

func TestNew(t *testing.T) {
	cfg := config.StatusConfig{
		Command:  "echo test",
		Interval: 10 * time.Second,
	}

	model := New(cfg)

	if model.command != "echo test" {
		t.Errorf("expected command 'echo test', got %q", model.command)
	}
	if model.interval != 10*time.Second {
		t.Errorf("expected interval 10s, got %v", model.interval)
	}
}

func TestModel_Update_FetchMsg_Success(t *testing.T) {
	cfg := config.StatusConfig{
		Command:  "echo test",
		Interval: 5 * time.Second,
	}
	model := New(cfg)

	model, _ = model.Update(fetchMsg{output: "CPU: 45%  Mem: 2GB"})

	if model.output != "CPU: 45%  Mem: 2GB" {
		t.Errorf("expected output 'CPU: 45%%  Mem: 2GB', got %q", model.output)
	}
	if model.err != nil {
		t.Errorf("expected no error, got %v", model.err)
	}
}

func TestModel_Update_FetchMsg_Error(t *testing.T) {
	cfg := config.StatusConfig{
		Command:  "echo test",
		Interval: 5 * time.Second,
	}
	model := New(cfg)

	model, _ = model.Update(fetchMsg{err: errTest})

	if model.err == nil {
		t.Error("expected error to be set")
	}
}

var errTest = &testError{msg: "test error"}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}

func TestModel_Update_TickMsg(t *testing.T) {
	cfg := config.StatusConfig{
		Command:  "echo test",
		Interval: 5 * time.Second,
	}
	model := New(cfg)

	model, cmd := model.Update(tickMsg(time.Now()))

	if !model.fetching {
		t.Error("expected fetching to be true after tick")
	}
	if cmd == nil {
		t.Error("expected command to be returned")
	}
}

func TestModel_Update_TickMsg_WhileFetching(t *testing.T) {
	cfg := config.StatusConfig{
		Command:  "echo test",
		Interval: 5 * time.Second,
	}
	model := New(cfg)
	model.fetching = true

	model, cmd := model.Update(tickMsg(time.Now()))

	if !model.fetching {
		t.Error("expected fetching to remain true")
	}
	if cmd == nil {
		t.Error("expected tick command to be returned")
	}
}

func TestModel_View_WithOutput(t *testing.T) {
	cfg := config.StatusConfig{
		Command:  "echo test",
		Interval: 5 * time.Second,
	}
	model := New(cfg)
	model.output = "CPU: 45%  Mem: 2GB  Disk: 80%"

	view := model.View(100)

	if !strings.Contains(view, "Quarterdeck") {
		t.Error("expected view to contain title")
	}
	if !strings.Contains(view, "CPU: 45%") {
		t.Error("expected view to contain output")
	}
}

func TestModel_View_WithError(t *testing.T) {
	cfg := config.StatusConfig{
		Command:  "echo test",
		Interval: 5 * time.Second,
	}
	model := New(cfg)
	model.err = errTest

	view := model.View(100)

	if !strings.Contains(view, "Error") {
		t.Error("expected view to show error")
	}
}

func TestModel_View_LongOutputWrapping(t *testing.T) {
	cfg := config.StatusConfig{
		Command:  "echo test",
		Interval: 5 * time.Second,
	}
	model := New(cfg)
	model.output = strings.Repeat("word ", 50)

	view := model.View(80)

	lines := strings.Split(view, "\n")
	if len(lines) < 2 {
		t.Error("expected long content to wrap to multiple lines")
	}
}

func TestModel_View_Truncation(t *testing.T) {
	cfg := config.StatusConfig{
		Command:  "echo test",
		Interval: 5 * time.Second,
	}
	model := New(cfg)
	model.output = strings.Repeat("word ", 200)

	view := model.View(80)

	lines := strings.Split(view, "\n")
	if len(lines) > maxLines {
		t.Errorf("expected at most %d lines, got %d", maxLines, len(lines))
	}
	if !strings.Contains(view, "…") {
		t.Error("expected truncation indicator")
	}
}

func TestModel_Init(t *testing.T) {
	cfg := config.StatusConfig{
		Command:  "echo 'hello world'",
		Interval: 5 * time.Second,
	}
	model := New(cfg)

	cmd := model.Init()
	if cmd == nil {
		t.Error("expected Init to return a command")
	}
}

func TestModel_Init_NoCommand(t *testing.T) {
	cfg := config.StatusConfig{
		Command:  "",
		Interval: 5 * time.Second,
	}
	model := New(cfg)

	cmd := model.Init()
	// When there's no command, Init should return nil (no timers needed)
	if cmd != nil {
		t.Error("expected Init to return nil when no command is configured")
	}
}

// TICKET-001: Test that empty output shows placeholder
func TestModel_View_EmptyOutput_ShowsPlaceholder(t *testing.T) {
	cfg := config.StatusConfig{
		Command:  "echo test",
		Interval: 5 * time.Second,
	}
	model := New(cfg)
	// No output set - model.output is empty string

	view := model.View(100)

	if !strings.Contains(view, "No status") {
		t.Error("expected view to show 'No status' placeholder when output is empty")
	}
}

// TICKET-002: Test that explicit newlines are properly truncated
func TestModel_View_ExplicitNewlines_Truncated(t *testing.T) {
	cfg := config.StatusConfig{
		Command:  "echo test",
		Interval: 5 * time.Second,
	}
	model := New(cfg)
	// Output with 6 lines (5 explicit newlines)
	model.output = "line1\nline2\nline3\nline4\nline5\nline6"

	view := model.View(80)

	lines := strings.Split(view, "\n")
	if len(lines) > maxLines {
		t.Errorf("expected at most %d lines with explicit newlines, got %d", maxLines, len(lines))
	}
	if !strings.Contains(view, "…") {
		t.Error("expected truncation indicator for content with more than 3 lines")
	}
}

// TICKET-003: Test that truncation ellipsis has space before it
func TestModel_View_Truncation_HasSpaceBeforeEllipsis(t *testing.T) {
	cfg := config.StatusConfig{
		Command:  "echo test",
		Interval: 5 * time.Second,
	}
	model := New(cfg)
	model.output = strings.Repeat("word ", 200) // Long enough to truncate

	view := model.View(80)

	if !strings.Contains(view, " …") {
		t.Error("expected space before truncation ellipsis")
	}
}

// TICKET-005: Test narrow terminal handling
func TestModel_View_NarrowTerminal(t *testing.T) {
	cfg := config.StatusConfig{
		Command:  "echo test",
		Interval: 5 * time.Second,
	}
	model := New(cfg)
	model.output = "short"

	// Very narrow terminal (smaller than title width) - should not panic
	// The output may be empty or truncated, but it shouldn't crash
	view := model.View(10)

	// Just verify it doesn't panic and produces some output
	if view == "" {
		t.Error("expected view to produce some output even on very narrow terminal")
	}
}

// Test minimum comfortable terminal width
func TestModel_View_MinimumWidth(t *testing.T) {
	cfg := config.StatusConfig{
		Command:  "echo test",
		Interval: 5 * time.Second,
	}
	model := New(cfg)
	model.output = "short"

	// Width that can fit title + some content (40 chars should be plenty)
	view := model.View(40)

	if !strings.Contains(view, "Quarterdeck") {
		t.Error("expected view to contain title")
	}
	if !strings.Contains(view, "short") {
		t.Error("expected view to contain output")
	}
}

// TICKET-006: Test concurrent fetch protection (tick while fetching)
func TestModel_Update_TickMsg_NoCommand(t *testing.T) {
	cfg := config.StatusConfig{
		Command:  "", // No command
		Interval: 5 * time.Second,
	}
	model := New(cfg)

	model, cmd := model.Update(tickMsg(time.Now()))

	// Should not set fetching to true when there's no command
	if model.fetching {
		t.Error("expected fetching to remain false when no command is configured")
	}
	// tick() returns nil when no command, so Update also returns nil
	if cmd != nil {
		t.Error("expected nil command when no command is configured")
	}
}

// TICKET-008: Test that tick() returns nil for empty command
func TestModel_Tick_NoCommand(t *testing.T) {
	cfg := config.StatusConfig{
		Command:  "",
		Interval: 5 * time.Second,
	}
	model := New(cfg)

	cmd := model.tick()
	if cmd != nil {
		t.Error("expected tick() to return nil when no command is configured")
	}
}

// TICKET-009: Test wide Unicode character handling
func TestModel_View_WideUnicodeCharacters(t *testing.T) {
	cfg := config.StatusConfig{
		Command:  "echo test",
		Interval: 5 * time.Second,
	}
	model := New(cfg)
	// East Asian wide characters - each takes 2 columns
	model.output = "日本語テスト" // 6 chars, 12 columns wide

	view := model.View(80)

	// Should contain the Unicode text
	if !strings.Contains(view, "日本語") {
		t.Error("expected view to contain Japanese characters")
	}
}

// Test fetch returns nil for empty command
func TestModel_Fetch_NoCommand(t *testing.T) {
	cfg := config.StatusConfig{
		Command:  "",
		Interval: 5 * time.Second,
	}
	model := New(cfg)

	cmd := model.fetch()
	if cmd != nil {
		t.Error("expected fetch() to return nil when no command is configured")
	}
}
