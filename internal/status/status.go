package status

import (
	"context"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/sourcehaven-bv/quarterdeck/internal/config"
)

const (
	// StatusTimeout is the max time to wait for a status command
	StatusTimeout = 10 * time.Second
	// MaxStatusOutputSize is the max bytes to read from status command
	MaxStatusOutputSize = 1024 * 1024 // 1MB
)

// Height is the maximum height of the status bar (up to 3 lines of content)
const Height = 3

// Model holds the status bar state
type Model struct {
	output   string
	loading  bool
	err      error
	command  string
	interval time.Duration
	fetching bool // track if a fetch is in progress to prevent concurrent fetches
}

func New(cfg config.StatusConfig) Model {
	return Model{
		command:  cfg.Command,
		interval: cfg.Interval,
	}
}

// Messages
type fetchMsg struct {
	output string
	err    error
}

type tickMsg time.Time

func (m Model) Init() tea.Cmd {
	// Don't start any timers or fetches if there's no command
	if m.command == "" {
		return nil
	}
	return tea.Batch(
		m.fetch(),
		m.tick(),
	)
}

func (m Model) tick() tea.Cmd {
	// Don't schedule ticks if there's no command to run
	if m.command == "" {
		return nil
	}
	return tea.Tick(m.interval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) fetch() tea.Cmd {
	if m.command == "" {
		return nil
	}
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), StatusTimeout)
		defer cancel()

		cmd := exec.CommandContext(ctx, "sh", "-c", m.command)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return fetchMsg{err: fmt.Errorf("failed to create pipe: %w", err)}
		}

		if startErr := cmd.Start(); startErr != nil {
			return fetchMsg{err: fmt.Errorf("failed to start command: %w", startErr)}
		}

		// Read with limit - LimitReader will stop at MaxStatusOutputSize+1 bytes
		// We read one extra byte to detect if the output exceeds the limit
		limited := io.LimitReader(stdout, MaxStatusOutputSize+1)
		output, err := io.ReadAll(limited)
		if err != nil {
			return fetchMsg{err: fmt.Errorf("failed to read output: %w", err)}
		}

		// Check for output size limit before waiting for command
		// This allows us to fail fast on oversized output
		if len(output) > MaxStatusOutputSize {
			// Kill the process since we're not going to use the output anyway
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			return fetchMsg{err: fmt.Errorf("output exceeds %d bytes limit", MaxStatusOutputSize)}
		}

		waitErr := cmd.Wait()

		// Check timeout first - it's the most specific error
		// The context deadline being exceeded is checked after Wait() returns
		// because Wait() may return early with a killed process error
		if ctx.Err() == context.DeadlineExceeded {
			return fetchMsg{err: fmt.Errorf("command timed out after %v", StatusTimeout)}
		}

		if waitErr != nil {
			return fetchMsg{err: fmt.Errorf("command failed: %w", waitErr)}
		}

		return fetchMsg{output: strings.TrimSpace(string(output))}
	}
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tickMsg:
		// Only fetch if we have a command and aren't already fetching
		if m.command != "" && !m.fetching {
			m.fetching = true
			return m, tea.Batch(m.fetch(), m.tick())
		}
		// Continue ticking even if we skipped the fetch
		return m, m.tick()

	case fetchMsg:
		m.fetching = false
		if msg.err != nil {
			m.err = msg.err
			m.loading = false
		} else {
			m.output = msg.output
			m.loading = false
			m.err = nil
		}
	}
	return m, nil
}

const (
	maxLines         = 3
	titleText        = "Quarterdeck"
	separator        = "    "
	minContentWidth  = 20
	emptyPlaceholder = "No status"
)

func (m Model) View(width int) string {
	style := lipgloss.NewStyle().
		Width(width).
		Padding(0, 2).
		Background(lipgloss.Color("235"))

	titleStr := titleStyle.Render(titleText)
	titleWidth := ansi.StringWidth(titleStr) + ansi.StringWidth(separator)

	var content string
	var isError bool
	switch {
	case m.err != nil:
		content = fmt.Sprintf("Error: %s", m.err.Error())
		isError = true
	case m.output != "":
		content = m.output
	default:
		content = dimStyle.Render(emptyPlaceholder)
	}

	// Calculate available width, ensuring it's at least 1 but not exceeding actual available space
	availableWidth := width - titleWidth - 4
	if availableWidth < 1 {
		availableWidth = 1
	}
	// Apply minimum only if we have enough space
	if availableWidth < minContentWidth && width >= titleWidth+4+minContentWidth {
		availableWidth = minContentWidth
	}

	// Use ansi.Wrap which properly handles wide Unicode characters and ANSI codes
	wrapped := ansi.Wrap(content, availableWidth, " ")
	lines := strings.Split(wrapped, "\n")

	truncated := false
	if len(lines) > maxLines {
		lines = lines[:maxLines]
		truncated = true
	}

	// Apply error styling to each line to preserve ANSI codes across wrapped lines
	if isError {
		for i := range lines {
			lines[i] = errorStyle.Render(lines[i])
		}
	}

	indent := strings.Repeat(" ", titleWidth)
	var result strings.Builder
	for i, line := range lines {
		if i == 0 {
			result.WriteString(titleStr)
			result.WriteString(separator)
			result.WriteString(line)
		} else {
			result.WriteString("\n")
			result.WriteString(indent)
			result.WriteString(line)
		}
	}

	if truncated {
		result.WriteString(dimStyle.Render(" …"))
	}

	return style.Render(result.String())
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("229"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))
)
