package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Command
// =============================================================================

// Command represents an action that can be executed from the palette
type Command struct {
	ID          string // Unique identifier
	Name        string // Display name
	Description string // What this command does
	Shortcut    string // Keyboard shortcut (for display)
	Category    string // Grouping category
}

// MatchScore returns how well this command matches the query
func (c Command) MatchScore(query string) int {
	if query == "" {
		return 100
	}

	query = strings.ToLower(query)
	name := strings.ToLower(c.Name)
	desc := strings.ToLower(c.Description)
	cat := strings.ToLower(c.Category)

	// Exact prefix match on name = highest score
	if strings.HasPrefix(name, query) {
		return 100
	}

	// Contains in name = high score
	if strings.Contains(name, query) {
		return 80
	}

	// Contains in category = medium-high score
	if strings.Contains(cat, query) {
		return 70
	}

	// Contains in description = medium score
	if strings.Contains(desc, query) {
		return 60
	}

	// Fuzzy match on name = low score
	if fuzzyMatch(name, query) {
		return 40
	}

	return 0
}

// =============================================================================
// Palette Model
// =============================================================================

// PaletteModel is a command palette (like VS Code's Ctrl+K)
type PaletteModel struct {
	commands   []Command
	width      int
	maxVisible int

	input    textinput.Model
	filtered []Command
	cursor   int
	selected *Command
	aborted  bool

	styles Styles
}

// PaletteResult is returned when palette closes
type PaletteResult struct {
	Command *Command
	Aborted bool
}

// DefaultCommands returns the standard command set
func DefaultCommands() []Command {
	return []Command{
		// Build/Run/Test
		{ID: "build", Name: "Build", Description: "Build the project", Shortcut: "b", Category: "Actions"},
		{ID: "run", Name: "Run", Description: "Build and run the app", Shortcut: "r", Category: "Actions"},
		{ID: "test", Name: "Test", Description: "Run tests", Shortcut: "t", Category: "Actions"},
		{ID: "clean", Name: "Clean", Description: "Clean build artifacts", Shortcut: "c", Category: "Actions"},
		{ID: "stop", Name: "Stop App", Description: "Stop running application", Shortcut: "x", Category: "Actions"},

		// Archive/Profile
		{ID: "archive", Name: "Archive", Description: "Create an archive for distribution", Category: "Build"},
		{ID: "archive-appstore", Name: "Archive for App Store", Description: "Archive with App Store profile", Category: "Build"},
		{ID: "archive-adhoc", Name: "Archive for Ad Hoc", Description: "Archive with Ad Hoc profile", Category: "Build"},
		{ID: "profile", Name: "Profile", Description: "Profile with Instruments", Category: "Build"},
		{ID: "analyze", Name: "Analyze", Description: "Run static analyzer", Category: "Build"},

		// Configuration
		{ID: "scheme", Name: "Switch Scheme", Description: "Change the active scheme", Shortcut: "s", Category: "Config"},
		{ID: "destination", Name: "Switch Destination", Description: "Change the target device/simulator", Shortcut: "d", Category: "Config"},
		{ID: "init", Name: "Initialize Config", Description: "Run the configuration wizard", Shortcut: "i", Category: "Config"},
		{ID: "refresh", Name: "Refresh Context", Description: "Rescan projects, schemes, and devices", Shortcut: "^R", Category: "Config"},

		// Utilities
		{ID: "doctor", Name: "Run Doctor", Description: "Check environment and dependencies", Category: "Utilities"},
		{ID: "logs", Name: "Stream Logs", Description: "Stream device/simulator logs", Category: "Utilities"},
		{ID: "simulator-boot", Name: "Boot Simulator", Description: "Boot the selected simulator", Category: "Utilities"},
		{ID: "simulator-shutdown", Name: "Shutdown Simulator", Description: "Shutdown all simulators", Category: "Utilities"},

		// Navigation
		{ID: "help", Name: "Show Help", Description: "Display keyboard shortcuts", Shortcut: "?", Category: "Navigation"},
		{ID: "quit", Name: "Quit", Description: "Exit xcbolt", Shortcut: "q", Category: "Navigation"},
	}
}

// NewPalette creates a new command palette
func NewPalette(screenWidth int, styles Styles) PaletteModel {
	// Calculate width: 50-60% of screen, clamped to reasonable bounds
	width := screenWidth * 55 / 100
	if width < 40 {
		width = 40
	}
	if width > 70 {
		width = 70
	}

	ti := textinput.New()
	ti.Placeholder = "Type a command..."
	ti.Focus()
	ti.CharLimit = 50
	ti.Width = width - 6

	commands := DefaultCommands()
	maxVisible := 12
	if len(commands) < maxVisible {
		maxVisible = len(commands)
	}

	return PaletteModel{
		commands:   commands,
		width:      width,
		maxVisible: maxVisible,
		input:      ti,
		filtered:   commands,
		cursor:     0,
		styles:     styles,
	}
}

// Init initializes the palette
func (m PaletteModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m PaletteModel) Update(msg tea.Msg) (PaletteModel, tea.Cmd, *PaletteResult) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.aborted = true
			return m, nil, &PaletteResult{Aborted: true}

		case "enter":
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				m.selected = &m.filtered[m.cursor]
				return m, nil, &PaletteResult{Command: m.selected}
			}
			return m, nil, nil

		case "up", "ctrl+p":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil, nil

		case "down", "ctrl+n":
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
			return m, nil, nil

		case "ctrl+u":
			m.input.SetValue("")
			m.filterCommands()
			return m, nil, nil

		default:
			m.input, cmd = m.input.Update(msg)
			m.filterCommands()
			return m, cmd, nil
		}
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd, nil
}

func (m *PaletteModel) filterCommands() {
	query := m.input.Value()

	if query == "" {
		m.filtered = m.commands
		m.cursor = 0
		return
	}

	type scored struct {
		cmd   Command
		score int
	}

	var scored_items []scored
	for _, c := range m.commands {
		score := c.MatchScore(query)
		if score > 0 {
			scored_items = append(scored_items, scored{cmd: c, score: score})
		}
	}

	// Sort by score
	for i := 0; i < len(scored_items); i++ {
		for j := i + 1; j < len(scored_items); j++ {
			if scored_items[j].score > scored_items[i].score {
				scored_items[i], scored_items[j] = scored_items[j], scored_items[i]
			}
		}
	}

	m.filtered = make([]Command, len(scored_items))
	for i, s := range scored_items {
		m.filtered[i] = s.cmd
	}

	if m.cursor >= len(m.filtered) {
		m.cursor = maxInt(0, len(m.filtered)-1)
	}
}

// View renders the palette
func (m PaletteModel) View() string {
	s := m.styles
	icons := s.Icons

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(s.Colors.Text)
	b.WriteString(titleStyle.Render("Commands"))
	b.WriteString("\n")

	// Divider
	dividerStyle := lipgloss.NewStyle().Foreground(s.Colors.BorderMuted)
	b.WriteString(dividerStyle.Render(strings.Repeat("─", m.width-4)))
	b.WriteString("\n")

	// Input with prompt
	promptStyle := lipgloss.NewStyle().
		Foreground(s.Colors.Accent).
		Bold(true)
	b.WriteString(promptStyle.Render("> ") + m.input.View())
	b.WriteString("\n")

	// Divider
	b.WriteString(dividerStyle.Render(strings.Repeat("─", m.width-4)))
	b.WriteString("\n")

	// Commands
	if len(m.filtered) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(s.Colors.TextMuted).
			Italic(true)
		b.WriteString(emptyStyle.Render("  No matching commands"))
		b.WriteString("\n")
	} else {
		// Calculate visible window
		start := 0
		end := len(m.filtered)
		if end > m.maxVisible {
			start = m.cursor - m.maxVisible/2
			if start < 0 {
				start = 0
			}
			end = start + m.maxVisible
			if end > len(m.filtered) {
				end = len(m.filtered)
				start = end - m.maxVisible
			}
		}

		categoryStyle := lipgloss.NewStyle().
			Foreground(s.Colors.TextSubtle).
			Bold(true).
			MarginTop(1)

		itemStyle := lipgloss.NewStyle().
			Foreground(s.Colors.Text)

		selectedStyle := lipgloss.NewStyle().
			Foreground(s.Colors.Text).
			Bold(true)

		shortcutStyle := lipgloss.NewStyle().
			Foreground(s.Colors.Accent)

		descStyle := lipgloss.NewStyle().
			Foreground(s.Colors.TextMuted).
			Italic(true)

		currentCategory := ""
		for i := start; i < end; i++ {
			cmd := m.filtered[i]
			isSelected := i == m.cursor

			// Category header (only show if changed and not filtering)
			if m.input.Value() == "" && cmd.Category != currentCategory {
				currentCategory = cmd.Category
				if i > start {
					b.WriteString("\n")
				}
				b.WriteString(categoryStyle.Render("  " + strings.ToUpper(currentCategory)))
				b.WriteString("\n")
			}

			// Build command line
			var line string
			if isSelected {
				line = s.StatusStyle("running").Render(icons.ChevronRight) + " "
				line += selectedStyle.Render(cmd.Name)
			} else {
				line = "  "
				line += itemStyle.Render(cmd.Name)
			}

			// Add shortcut if present (right-aligned would be nice but tricky)
			if cmd.Shortcut != "" {
				line += " " + shortcutStyle.Render("["+cmd.Shortcut+"]")
			}

			// Add description when filtering (shows context for fuzzy matches)
			if cmd.Description != "" && m.input.Value() != "" {
				line += " " + descStyle.Render(cmd.Description)
			}

			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	// Divider
	b.WriteString(dividerStyle.Render(strings.Repeat("─", m.width-4)))
	b.WriteString("\n")

	// Hints
	hintKeyStyle := lipgloss.NewStyle().Foreground(s.Colors.Accent)
	hintDescStyle := lipgloss.NewStyle().Foreground(s.Colors.TextSubtle)
	hints := hintKeyStyle.Render("↑↓") + hintDescStyle.Render(" navigate  ") +
		hintKeyStyle.Render("⏎") + hintDescStyle.Render(" run  ") +
		hintKeyStyle.Render("esc") + hintDescStyle.Render(" cancel")
	b.WriteString(hints)

	content := b.String()

	// Container with border
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(s.Colors.Border).
		Padding(1, 2)

	return containerStyle.Width(m.width).Render(content)
}

// =============================================================================
// Centered popup helper
// =============================================================================

// RenderPaletteCentered renders the palette centered on screen
func RenderPaletteCentered(content string, screenWidth, screenHeight int) string {
	return lipgloss.Place(
		screenWidth,
		screenHeight,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}
