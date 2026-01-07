package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Selector Item
// =============================================================================

// SelectorItem represents an item that can be selected
type SelectorItem struct {
	ID          string // Unique identifier (e.g., UDID for simulators)
	Title       string // Display title
	Description string // Secondary info (e.g., OS version, state)
	Meta        string // Additional metadata (e.g., "[booted]")
}

// MatchScore returns how well this item matches the query (higher = better)
func (item SelectorItem) MatchScore(query string) int {
	if query == "" {
		return 100 // All items match empty query
	}

	query = strings.ToLower(query)
	title := strings.ToLower(item.Title)
	desc := strings.ToLower(item.Description)

	// Exact prefix match on title = highest score
	if strings.HasPrefix(title, query) {
		return 100
	}

	// Contains in title = high score
	if strings.Contains(title, query) {
		return 80
	}

	// Contains in description = medium score
	if strings.Contains(desc, query) {
		return 60
	}

	// Fuzzy match (all chars in order) = low score
	if fuzzyMatch(title, query) {
		return 40
	}

	return 0 // No match
}

// fuzzyMatch checks if all characters of needle appear in haystack in order
func fuzzyMatch(haystack, needle string) bool {
	hIdx := 0
	for _, char := range needle {
		found := false
		for hIdx < len(haystack) {
			if rune(haystack[hIdx]) == char {
				found = true
				hIdx++
				break
			}
			hIdx++
		}
		if !found {
			return false
		}
	}
	return true
}

// =============================================================================
// Selector Model
// =============================================================================

// SelectorModel is a fuzzy-search selector popup
type SelectorModel struct {
	// Configuration
	title      string
	items      []SelectorItem
	width      int
	maxVisible int

	// State
	input    textinput.Model
	filtered []SelectorItem
	cursor   int
	selected *SelectorItem
	aborted  bool

	// Styling
	styles Styles
}

// SelectorResult is returned when selector closes
type SelectorResult struct {
	Selected *SelectorItem
	Aborted  bool
}

// NewSelector creates a new selector
func NewSelector(title string, items []SelectorItem, width int, styles Styles) SelectorModel {
	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.Focus()
	ti.CharLimit = 50
	ti.Width = width - 6

	maxVisible := 8
	if len(items) < maxVisible {
		maxVisible = len(items)
	}

	return SelectorModel{
		title:      title,
		items:      items,
		width:      width,
		maxVisible: maxVisible,
		input:      ti,
		filtered:   items, // Initially show all
		cursor:     0,
		styles:     styles,
	}
}

// Init initializes the selector
func (m SelectorModel) Init() tea.Cmd {
	return textinput.Blink
}

// Update handles messages
func (m SelectorModel) Update(msg tea.Msg) (SelectorModel, tea.Cmd, *SelectorResult) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			m.aborted = true
			return m, nil, &SelectorResult{Aborted: true}

		case "enter":
			if len(m.filtered) > 0 && m.cursor < len(m.filtered) {
				m.selected = &m.filtered[m.cursor]
				return m, nil, &SelectorResult{Selected: m.selected}
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
			// Clear input
			m.input.SetValue("")
			m.filterItems()
			return m, nil, nil

		default:
			// Pass to text input
			m.input, cmd = m.input.Update(msg)
			m.filterItems()
			return m, cmd, nil
		}
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd, nil
}

// filterItems updates the filtered list based on current input
func (m *SelectorModel) filterItems() {
	query := m.input.Value()

	if query == "" {
		m.filtered = m.items
		m.cursor = 0
		return
	}

	// Score and filter items
	type scored struct {
		item  SelectorItem
		score int
	}

	var scored_items []scored
	for _, item := range m.items {
		score := item.MatchScore(query)
		if score > 0 {
			scored_items = append(scored_items, scored{item: item, score: score})
		}
	}

	// Sort by score (simple bubble sort for small lists)
	for i := 0; i < len(scored_items); i++ {
		for j := i + 1; j < len(scored_items); j++ {
			if scored_items[j].score > scored_items[i].score {
				scored_items[i], scored_items[j] = scored_items[j], scored_items[i]
			}
		}
	}

	m.filtered = make([]SelectorItem, len(scored_items))
	for i, s := range scored_items {
		m.filtered[i] = s.item
	}

	// Reset cursor if out of bounds
	if m.cursor >= len(m.filtered) {
		m.cursor = maxInt(0, len(m.filtered)-1)
	}
}

// View renders the selector
func (m SelectorModel) View() string {
	s := m.styles

	var b strings.Builder

	// Title
	b.WriteString(s.Selector.Title.Render(m.title))
	b.WriteString("\n")

	// Divider
	b.WriteString(s.Selector.Divider.Render(strings.Repeat("─", m.width-4)))
	b.WriteString("\n")

	// Input
	b.WriteString(s.Selector.Input.Render("> " + m.input.View()))
	b.WriteString("\n")

	// Divider
	b.WriteString(s.Selector.Divider.Render(strings.Repeat("─", m.width-4)))
	b.WriteString("\n")

	// Items
	if len(m.filtered) == 0 {
		b.WriteString(s.Logs.EmptyState.Render("No matches"))
		b.WriteString("\n")
	} else {
		// Calculate visible window
		start := 0
		end := len(m.filtered)
		if end > m.maxVisible {
			// Center the cursor in the visible window
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

		for i := start; i < end; i++ {
			item := m.filtered[i]
			isSelected := i == m.cursor

			// Build item line
			var line string
			if isSelected {
				line = s.Icons.ChevronRight + " "
				line += s.Selector.ItemSelected.Render(item.Title)
			} else {
				line = "  "
				line += s.Selector.Item.Render(item.Title)
			}

			// Add description if present
			if item.Description != "" {
				line += " " + s.Selector.ItemMeta.Render(item.Description)
			}

			// Add meta tag if present
			if item.Meta != "" {
				line += " " + s.Selector.ItemMeta.Render(item.Meta)
			}

			b.WriteString(line)
			b.WriteString("\n")
		}
	}

	// Divider
	b.WriteString(s.Selector.Divider.Render(strings.Repeat("─", m.width-4)))
	b.WriteString("\n")

	// Hints
	hints := "↑↓ navigate  ⏎ select  esc cancel"
	b.WriteString(s.Selector.Hint.Render(hints))

	// Wrap in container
	content := b.String()
	return s.Selector.Container.Width(m.width).Render(content)
}

// =============================================================================
// Helper to create selector items from context
// =============================================================================

// SchemeItems creates selector items from schemes
func SchemeItems(schemes []string) []SelectorItem {
	items := make([]SelectorItem, len(schemes))
	for i, s := range schemes {
		items[i] = SelectorItem{
			ID:    s,
			Title: s,
		}
	}
	return items
}

// DestinationItems creates selector items from simulators and devices
func DestinationItems(sims []SimulatorInfo, devices []DeviceInfo) []SelectorItem {
	items := make([]SelectorItem, 0, len(sims)+len(devices))

	// Add simulators
	for _, sim := range sims {
		meta := ""
		if sim.State == "Booted" {
			meta = "[booted]"
		}
		items = append(items, SelectorItem{
			ID:          sim.UDID,
			Title:       sim.Name,
			Description: sim.RuntimeName,
			Meta:        meta,
		})
	}

	// Add devices
	for _, dev := range devices {
		items = append(items, SelectorItem{
			ID:          dev.Identifier,
			Title:       dev.Name,
			Description: dev.OSVersion,
			Meta:        "[device]",
		})
	}

	return items
}

// SimulatorInfo matches core.Simulator structure
type SimulatorInfo struct {
	Name        string
	UDID        string
	State       string
	RuntimeName string
	OSVersion   string
	Available   bool
}

// DeviceInfo matches core.Device structure
type DeviceInfo struct {
	Name       string
	Identifier string
	Platform   string
	OSVersion  string
	Model      string
}

// =============================================================================
// Centered popup rendering helper
// =============================================================================

// RenderCenteredPopup renders a popup centered on screen
func RenderCenteredPopup(content string, screenWidth, screenHeight int) string {
	return lipgloss.Place(
		screenWidth,
		screenHeight,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}
