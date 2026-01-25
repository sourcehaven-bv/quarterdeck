package ui

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	// MaxOutputLines is the maximum number of lines to keep in output buffer
	MaxOutputLines = 10000
	// MaxScannerBuffer is the maximum line size for the scanner (10MB)
	MaxScannerBuffer = 10 * 1024 * 1024
)

// OutputMsg is sent when there's new output from a command.
// It includes a command ID to filter out stale messages from cancelled commands.
type OutputMsg struct {
	text      string
	commandID uint64
}

// NewOutputMsg creates an OutputMsg with a specific command ID
func NewOutputMsg(text string, commandID uint64) OutputMsg {
	return OutputMsg{text: text, commandID: commandID}
}

// Text returns the output text
func (m OutputMsg) Text() string {
	return m.text
}

// CommandID returns the command generation ID
func (m OutputMsg) CommandID() uint64 {
	return m.commandID
}

// OutputDoneMsg is sent when command completes.
// It includes a command ID to match with the active command.
type OutputDoneMsg struct {
	CommandID uint64
}

// NewOutputDoneMsg creates an OutputDoneMsg with a specific command ID
func NewOutputDoneMsg(commandID uint64) OutputDoneMsg {
	return OutputDoneMsg{CommandID: commandID}
}

// OutputModel handles the output pane using viewport for scrolling
type OutputModel struct {
	viewport     viewport.Model
	lines        []string
	pendingLines []string // buffer for refresh mode
	done         bool
	running      bool
	refreshing   bool // true when collecting output for a refresh (no clear)
}

func NewOutputModel() OutputModel {
	vp := viewport.New(0, 0)
	vp.MouseWheelEnabled = true
	return OutputModel{
		viewport: vp,
		lines:    []string{},
	}
}

func (m OutputModel) SetSize(w, h int) OutputModel {
	m.viewport.Width = w
	m.viewport.Height = h - 1 // Reserve space for status line
	return m
}

func (m OutputModel) AppendLine(line string) OutputModel {
	if m.refreshing {
		// In refresh mode, collect to pending buffer without updating display
		m.pendingLines = append(m.pendingLines, line)
		if len(m.pendingLines) > MaxOutputLines {
			m.pendingLines = m.pendingLines[len(m.pendingLines)-MaxOutputLines:]
		}
		return m
	}

	m.lines = append(m.lines, line)

	// Enforce maximum line limit to prevent memory exhaustion
	if len(m.lines) > MaxOutputLines {
		m.lines = m.lines[len(m.lines)-MaxOutputLines:]
	}

	// Update viewport content
	m.viewport.SetContent(strings.Join(m.lines, "\n"))
	// Auto-scroll to bottom
	m.viewport.GotoBottom()
	return m
}

// Clear resets the output content but preserves dimensions
func (m OutputModel) Clear() OutputModel {
	m.lines = []string{}
	m.done = false
	m.running = false
	m.viewport.SetContent("")
	m.viewport.GotoTop()
	return m
}

func (m OutputModel) SetDone(done bool) OutputModel {
	m.done = done
	m.running = false
	// If we were refreshing, commit the pending content
	if m.refreshing {
		m.lines = m.pendingLines
		m.pendingLines = nil
		m.refreshing = false
		m.viewport.SetContent(strings.Join(m.lines, "\n"))
		m.viewport.GotoBottom()
	}
	return m
}

// StartRefresh begins collecting output for a refresh (doesn't clear current display)
func (m OutputModel) StartRefresh() OutputModel {
	m.refreshing = true
	m.pendingLines = []string{}
	m.running = true
	m.done = false
	return m
}

// IsRefreshing returns true if currently collecting output for a refresh
func (m OutputModel) IsRefreshing() bool {
	return m.refreshing
}

// SetRunning marks the output as having a running command
func (m OutputModel) SetRunning(running bool) OutputModel {
	m.running = running
	return m
}

// IsRunning returns true if a command is currently running
func (m OutputModel) IsRunning() bool {
	return m.running
}

func (m OutputModel) Update(msg tea.Msg) (OutputModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m OutputModel) View() string {
	if len(m.lines) == 0 {
		return emptyStyle.Render("Press Enter to run selected action\nUse Tab to switch panes")
	}

	// Status indicator (hide during refresh to avoid flicker)
	status := ""
	if m.running && !m.refreshing {
		status = runningStyle.Render(" [Running...]")
	}

	return m.viewport.View() + "\n" + status
}

var (
	emptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			Italic(true).
			Padding(2)

	runningStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))
)

// RunShellCommand executes a shell command string and returns output.
// This is the non-streaming version used when no program reference is available.
// It uses command ID 0 which means it won't be filtered.
func RunShellCommand(command string, commandID uint64) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("sh", "-c", command)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return NewOutputMsg("Error: "+err.Error()+"\n"+string(output), commandID)
		}
		return NewOutputMsg(string(output), commandID)
	}
}

// StreamShellCommand executes a command and streams output line by line.
// The context can be cancelled to stop the command and its goroutine.
// The commandID is used to filter out stale messages from cancelled commands.
func StreamShellCommand(ctx context.Context, p *tea.Program, command string, commandID uint64) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.CommandContext(ctx, "sh", "-c", command)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return NewOutputMsg("Error: "+err.Error(), commandID)
		}
		cmd.Stderr = cmd.Stdout // Combine stderr with stdout

		if err := cmd.Start(); err != nil {
			return NewOutputMsg("Error: "+err.Error(), commandID)
		}

		scanner := bufio.NewScanner(stdout)
		// Increase buffer size for long lines (up to 10MB)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, MaxScannerBuffer)

		for scanner.Scan() {
			// Check if context was cancelled before sending
			select {
			case <-ctx.Done():
				// Context cancelled, stop sending output and exit
				// Kill the process (CommandContext handles this, but be explicit)
				_ = cmd.Process.Kill()
				return NewOutputDoneMsg(commandID)
			default:
				// Continue processing
			}

			line := scanner.Text()
			// Truncate very long lines for display
			if len(line) > 1024*1024 {
				line = line[:1024*1024] + "... [line truncated]"
			}
			p.Send(NewOutputMsg(line, commandID))
		}

		// Check for scanner errors (ignore if context was cancelled)
		if err := scanner.Err(); err != nil && ctx.Err() == nil {
			p.Send(NewOutputMsg(fmt.Sprintf("Error reading output: %v", err), commandID))
		}

		// Wait for command to finish (ignore error if context was cancelled)
		if err := cmd.Wait(); err != nil && ctx.Err() == nil {
			p.Send(NewOutputMsg(fmt.Sprintf("Command exited with error: %v", err), commandID))
		}
		return NewOutputDoneMsg(commandID)
	}
}

// OpenBrowser opens a URL in the default browser.
// Browser opens don't need cancellation or command IDs since they're fire-and-forget.
func OpenBrowser(url string, commandID uint64) tea.Cmd {
	return func() tea.Msg {
		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "linux":
			cmd = exec.Command("xdg-open", url)
		default:
			return NewOutputMsg("Cannot open browser on "+runtime.GOOS, commandID)
		}
		if err := cmd.Start(); err != nil {
			return NewOutputMsg("Error opening browser: "+err.Error(), commandID)
		}
		return NewOutputMsg("Opened "+url+" in browser", commandID)
	}
}
