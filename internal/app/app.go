package app

import (
	"context"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/sourcehaven-bv/quarterdeck/internal/config"
	"github.com/sourcehaven-bv/quarterdeck/internal/status"
	"github.com/sourcehaven-bv/quarterdeck/internal/ui"
)

// Pane represents which pane is currently focused
type Pane int

const (
	PaneMenu Pane = iota
	PaneOutput
)

// Minimum terminal size constants
const (
	MinTerminalWidth  = 60
	MinTerminalHeight = 12
)

// UI layout constants
const (
	minContentHeight = 5
	outputWidthPad   = 4
	borderPad        = 2
	windowSizePad    = 4
)

// Model is the main application state
type Model struct {
	config         *config.Config
	menu           ui.MenuModel
	status         status.Model
	output         ui.OutputModel
	hintbar        ui.HintBar
	focusedPane    Pane
	width          int
	height         int
	quitting       bool
	confirm        *confirmState // non-nil when awaiting confirmation
	program        **tea.Program // pointer to pointer for streaming output (allows setting after program creation)
	activeInterval time.Duration // interval for currently focused item (0 = no interval)
	intervalItemID int           // menu index of item with active interval (-1 = none)

	// Command cancellation tracking
	commandID     uint64             // current command generation counter (incremented on each new command)
	cancelCommand context.CancelFunc // cancel function for the currently running command (nil if none)
}

type confirmState struct {
	itemName string // name of the item being confirmed
	message  string
	command  string
	isOpen   bool // true if this is an "open" action
}

// confirmMsg is sent to trigger showing a confirmation dialog
type confirmMsg struct {
	itemName string
	message  string
	command  string
	isOpen   bool
}

// intervalTickMsg is sent periodically for items with interval configured
type intervalTickMsg struct {
	itemID int // menu index to verify we're still on the same item
}

func initialModel(cfg *config.Config, programRef **tea.Program) Model {
	return Model{
		config:         cfg,
		menu:           ui.NewMenuModel(cfg.Menu),
		status:         status.New(cfg.Status),
		output:         ui.NewOutputModel(),
		hintbar:        ui.NewHintBar(),
		focusedPane:    PaneMenu,
		program:        programRef,
		intervalItemID: -1,
		commandID:      0, // Start with command ID 0
	}
}

func (m *Model) Init() tea.Cmd {
	var cmds []tea.Cmd

	// Initialize status bar
	cmds = append(cmds, m.status.Init())

	// Auto-execute first item if configured (TICKET-002)
	if len(m.menu.Items()) > 0 {
		item := m.menu.SelectedItem()
		if item.Config.Mode == config.ModeAuto && item.Config.Run != "" {
			cmds = append(cmds, m.runCommand(item.Config.Run))
			// Start interval if configured
			if item.Config.Interval > 0 {
				cmds = append(cmds, m.startInterval(item.Config.Interval, m.menu.Cursor()))
			}
		}
	}

	return tea.Batch(cmds...)
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyMsg(msg)

	case tea.WindowSizeMsg:
		m.handleWindowSize(msg)

	case ui.OutputMsg:
		// Only accept output from the current command (filter stale messages)
		if msg.CommandID() == m.commandID {
			m.output = m.output.AppendLine(msg.Text())
		}

	case ui.OutputDoneMsg:
		// Only mark as done if this is from the current command
		if msg.CommandID == m.commandID {
			m.output = m.output.SetDone(true)
		}

	case confirmMsg:
		m.confirm = &confirmState{
			itemName: msg.itemName,
			message:  msg.message,
			command:  msg.command,
			isOpen:   msg.isOpen,
		}
		return m, nil

	case intervalTickMsg:
		// Only re-run if still on the same item and not running another command
		if msg.itemID == m.intervalItemID && m.activeInterval > 0 && !m.output.IsRunning() {
			item := m.menu.SelectedItem()
			if item.Config.Run != "" {
				// Use refresh mode for smooth updates (no flicker)
				m.output = m.output.StartRefresh()
				return m, tea.Batch(m.runCommand(item.Config.Run), m.scheduleIntervalTick())
			}
		}
		return m, nil
	}

	// Always update status model for tick messages
	var statusCmd tea.Cmd
	m.status, statusCmd = m.status.Update(msg)
	if statusCmd != nil {
		cmds = append(cmds, statusCmd)
	}

	return m, tea.Batch(cmds...)
}

// handleWindowSize processes window resize events
func (m *Model) handleWindowSize(msg tea.WindowSizeMsg) {
	m.width = msg.Width
	m.height = msg.Height

	outputWidth := msg.Width - ui.MenuWidth - outputWidthPad
	if outputWidth < 10 {
		outputWidth = 10
	}
	contentHeight := msg.Height - status.Height - ui.HintBarHeight - windowSizePad
	if contentHeight < minContentHeight {
		contentHeight = minContentHeight
	}

	m.output = m.output.SetSize(outputWidth, contentHeight)
	m.hintbar = m.hintbar.SetWidth(msg.Width)
}

// handleKeyMsg processes all keyboard input
func (m *Model) handleKeyMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Handle confirmation dialog first
	if m.confirm != nil {
		return m.handleConfirmKey(msg)
	}

	switch msg.String() {
	case "q", "ctrl+c":
		// Cancel any running command before quitting
		m.cancelCurrentCommand()
		m.quitting = true
		return m, tea.Quit

	case "tab":
		m.focusedPane = m.togglePane()
		return m, nil

	case "enter":
		if m.focusedPane == PaneMenu {
			return m.handleEnterKey()
		}

	default:
		if m.focusedPane == PaneMenu {
			if result, cmd, handled := m.handleActionKey(msg.String()); handled {
				return result, cmd
			}
		}
	}

	return m.handlePaneNavigation(msg)
}

// handleConfirmKey handles y/n keypresses in confirmation dialog
func (m *Model) handleConfirmKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "Y":
		confirm := m.confirm
		m.confirm = nil
		if confirm.isOpen {
			return m, ui.OpenBrowser(confirm.command, m.commandID)
		}
		return m, m.runCommand(confirm.command)
	case "n", "N", "esc":
		m.confirm = nil
		return m, nil
	}
	return m, nil
}

// togglePane switches between menu and output panes
func (m *Model) togglePane() Pane {
	if m.focusedPane == PaneMenu {
		return PaneOutput
	}
	return PaneMenu
}

// handleEnterKey executes the selected menu item
func (m *Model) handleEnterKey() (tea.Model, tea.Cmd) {
	if m.output.IsRunning() {
		return m, nil
	}
	item := m.menu.SelectedItem()
	clearOutput, cmd := m.executeItem(item.Config)
	if clearOutput {
		m.output = m.output.Clear()
		m.output = m.output.SetRunning(true)
		// Start interval if configured
		if item.Config.Interval > 0 {
			intervalCmd := m.startInterval(item.Config.Interval, m.menu.Cursor())
			return m, tea.Batch(cmd, intervalCmd)
		}
	}
	return m, cmd
}

// handleActionKey checks for and executes action keybindings
func (m *Model) handleActionKey(key string) (tea.Model, tea.Cmd, bool) {
	action := m.menu.GetActionByKey(key)
	if action == nil {
		return m, nil, false
	}
	if m.output.IsRunning() {
		return m, nil, true
	}
	clearOutput, cmd := m.executeAction(*action)
	if clearOutput {
		// Stop interval when an action is executed to prevent overwriting action output
		m.stopInterval()
		m.output = m.output.Clear()
		m.output = m.output.SetRunning(true)
	}
	return m, cmd, true
}

// handleMenuCursorChange handles cursor changes in the menu pane
func (m *Model) handleMenuCursorChange() []tea.Cmd {
	var cmds []tea.Cmd
	// Stop any existing interval when navigating away
	m.stopInterval()
	// Cancel any running command before starting a new one
	m.cancelCurrentCommand()
	m.output = m.output.Clear()
	item := m.menu.SelectedItem()
	if item.Config.Mode != config.ModeAuto || item.Config.Run == "" {
		return cmds
	}
	m.output = m.output.SetRunning(true)
	cmds = append(cmds, m.runCommand(item.Config.Run))
	// Start interval if configured for auto-execute items
	if item.Config.Interval > 0 {
		cmds = append(cmds, m.startInterval(item.Config.Interval, m.menu.Cursor()))
	}
	return cmds
}

// handlePaneNavigation passes navigation keys to the focused pane
func (m *Model) handlePaneNavigation(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	if m.focusedPane == PaneMenu {
		var cmd tea.Cmd
		m.menu, cmd = m.menu.Update(msg)
		cmds = append(cmds, cmd)
		if m.menu.CursorChanged() {
			cmds = append(cmds, m.handleMenuCursorChange()...)
		}
	} else {
		var cmd tea.Cmd
		m.output, cmd = m.output.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update status model
	var statusCmd tea.Cmd
	m.status, statusCmd = m.status.Update(msg)
	if statusCmd != nil {
		cmds = append(cmds, statusCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m *Model) executeItem(item config.ItemConfig) (bool, tea.Cmd) {
	if item.Open != "" {
		if item.Mode == config.ModeConfirm {
			return false, func() tea.Msg {
				return confirmMsg{
					itemName: item.Name,
					message:  "Open: " + item.Open,
					command:  item.Open,
					isOpen:   true,
				}
			}
		}
		return false, ui.OpenBrowser(item.Open, m.commandID)
	}
	if item.Run != "" {
		if item.Mode == config.ModeConfirm {
			return false, func() tea.Msg {
				return confirmMsg{
					itemName: item.Name,
					message:  "Run: " + item.Run,
					command:  item.Run,
					isOpen:   false,
				}
			}
		}
		return true, m.runCommand(item.Run) // true = clear output first
	}
	return false, nil
}

func (m *Model) executeAction(action config.ActionConfig) (bool, tea.Cmd) {
	if action.Open != "" {
		if action.Mode == config.ModeConfirm {
			return false, func() tea.Msg {
				return confirmMsg{
					itemName: action.Name,
					message:  "Open: " + action.Open,
					command:  action.Open,
					isOpen:   true,
				}
			}
		}
		return false, ui.OpenBrowser(action.Open, m.commandID)
	}
	if action.Run != "" {
		if action.Mode == config.ModeConfirm {
			return false, func() tea.Msg {
				return confirmMsg{
					itemName: action.Name,
					message:  "Run: " + action.Run,
					command:  action.Run,
					isOpen:   false,
				}
			}
		}
		return true, m.runCommand(action.Run) // true = clear output first
	}
	return false, nil
}

// runCommand executes a command and returns a tea.Cmd.
// It increments the command ID and creates a cancellable context.
// Note: This method modifies m, so it should be called on a pointer receiver
// or the caller should use the returned Model.
func (m *Model) runCommand(command string) tea.Cmd {
	// Cancel any previous command
	m.cancelCurrentCommand()

	// Increment command ID for this new command
	m.commandID++
	cmdID := m.commandID

	// Create a new cancellable context for this command
	ctx, cancel := context.WithCancel(context.Background())
	m.cancelCommand = cancel

	if m.program != nil && *m.program != nil {
		return ui.StreamShellCommand(ctx, *m.program, command, cmdID)
	}
	return ui.RunShellCommand(command, cmdID)
}

// cancelCurrentCommand cancels the currently running command if any
func (m *Model) cancelCurrentCommand() {
	if m.cancelCommand != nil {
		m.cancelCommand()
		m.cancelCommand = nil
	}
}

func (m *Model) scheduleIntervalTick() tea.Cmd {
	if m.activeInterval <= 0 || m.intervalItemID < 0 {
		return nil
	}
	itemID := m.intervalItemID
	interval := m.activeInterval
	return tea.Tick(interval, func(_ time.Time) tea.Msg {
		return intervalTickMsg{itemID: itemID}
	})
}

// startInterval returns a command that sets up interval tracking and schedules the first tick
func (m *Model) startInterval(interval time.Duration, itemID int) tea.Cmd {
	m.activeInterval = interval
	m.intervalItemID = itemID
	return m.scheduleIntervalTick()
}

// stopInterval clears interval tracking
func (m *Model) stopInterval() {
	m.activeInterval = 0
	m.intervalItemID = -1
}

func (m *Model) View() string {
	if m.quitting {
		return "Goodbye!\n"
	}

	// Show warning for small terminals (TICKET-011)
	if m.width < MinTerminalWidth || m.height < MinTerminalHeight {
		return lipgloss.NewStyle().
			Padding(1, 2).
			Render(fmt.Sprintf("Terminal too small!\n\nMinimum size: %dx%d\nCurrent size: %dx%d\n\nPlease resize your terminal.",
				MinTerminalWidth, MinTerminalHeight, m.width, m.height))
	}

	// Status bar on top
	statusBar := m.status.View(m.width)

	// Calculate content height (subtract status bar and hint bar)
	contentHeight := m.height - status.Height - ui.HintBarHeight - borderPad
	if contentHeight < minContentHeight {
		contentHeight = minContentHeight
	}

	menuStyle := lipgloss.NewStyle().
		Width(ui.MenuWidth).
		Height(contentHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.BorderColor(m.focusedPane == PaneMenu))

	outputStyle := lipgloss.NewStyle().
		Width(m.width - ui.MenuWidth - outputWidthPad).
		Height(contentHeight).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(ui.BorderColor(m.focusedPane == PaneOutput))

	menu := menuStyle.Render(m.menu.View())
	output := outputStyle.Render(m.output.View())

	content := lipgloss.JoinHorizontal(lipgloss.Top, menu, output)

	// Confirmation dialog overlay
	if m.confirm != nil {
		// Build confirmation message with item name for better context (TICKET-014)
		confirmText := fmt.Sprintf("Confirm: %s\n\n%s\n\n[y]es  [n]o", m.confirm.itemName, m.confirm.message)
		confirmBox := lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("214")).
			Padding(1, 2).
			Render(confirmText)

		// Center the confirm box (simple approach)
		content = lipgloss.Place(
			m.width-2,
			contentHeight,
			lipgloss.Center,
			lipgloss.Center,
			confirmBox,
		)
	}

	// Hint bar at bottom
	selectedItem := m.menu.SelectedItem()
	hintBar := m.hintbar.View(selectedItem.Config)

	return lipgloss.JoinVertical(lipgloss.Left, statusBar, content, hintBar)
}

func Run(cfg *config.Config) error {
	// Use pointer-to-pointer so we can set the program after creation
	var programRef *tea.Program
	m := initialModel(cfg, &programRef)

	p := tea.NewProgram(
		&m, // Pass pointer to model for pointer receiver methods
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	// Set program reference for streaming output
	programRef = p

	_, err := p.Run()
	return err
}
