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
	title             string
	items             []SelectorItem
	recentItems       []SelectorItem // Recent items to pin at top
	width             int
	maxVisible        int
	selectedID        string
	keepSelected      bool
	showSelectedBadge bool

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
func NewSelector(title string, items []SelectorItem, screenWidth int, styles Styles) SelectorModel {
	// Calculate width: 50-60% of screen, clamped
	width := screenWidth * 55 / 100
	if width < 40 {
		width = 40
	}
	if width > 70 {
		width = 70
	}

	ti := textinput.New()
	ti.Placeholder = "Type to filter..."
	ti.Focus()
	ti.CharLimit = 50
	ti.Width = width - 6

	maxVisible := 10
	if len(items) < maxVisible {
		maxVisible = len(items)
	}

	return SelectorModel{
		title:             title,
		items:             items,
		width:             width,
		maxVisible:        maxVisible,
		input:             ti,
		filtered:          items, // Initially show all
		cursor:            0,
		styles:            styles,
		showSelectedBadge: true,
	}
}

// NewSelectorWithSelected creates a selector with a pre-selected item (by ID).
func NewSelectorWithSelected(title string, items []SelectorItem, selectedID string, screenWidth int, styles Styles) SelectorModel {
	m := NewSelector(title, items, screenWidth, styles)
	if selectedID == "" {
		return m
	}
	m.selectedID = selectedID
	m.keepSelected = true
	for i, item := range m.filtered {
		if item.ID == selectedID {
			m.cursor = i
			break
		}
	}
	return m
}

// NewSelectorWithRecents creates a selector with recent items pinned at top
func NewSelectorWithRecents(title string, items []SelectorItem, recents []SelectorItem, screenWidth int, styles Styles) SelectorModel {
	m := NewSelector(title, items, screenWidth, styles)
	m.recentItems = recents
	return m
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
			m.keepSelected = false
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil, nil

		case "down", "ctrl+n":
			m.keepSelected = false
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
		if !m.applySelectedCursor() {
			m.cursor = 0
		}
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

	if !m.applySelectedCursor() {
		// Reset cursor if out of bounds
		if m.cursor >= len(m.filtered) {
			m.cursor = maxInt(0, len(m.filtered)-1)
		}
	}
}

func (m *SelectorModel) applySelectedCursor() bool {
	if !m.keepSelected || m.selectedID == "" || len(m.filtered) == 0 {
		return false
	}
	for i, item := range m.filtered {
		if item.ID == m.selectedID {
			m.cursor = i
			return true
		}
	}
	return false
}

// View renders the selector
func (m SelectorModel) View() string {
	s := m.styles
	icons := s.Icons

	var b strings.Builder

	// Title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(s.Colors.Text)
	b.WriteString(titleStyle.Render(m.title))
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

	// Styles for items
	sectionStyle := lipgloss.NewStyle().
		Foreground(s.Colors.TextSubtle).
		Bold(true)

	itemStyle := lipgloss.NewStyle().
		Foreground(s.Colors.Text)

	selectedStyle := lipgloss.NewStyle().
		Foreground(s.Colors.Text).
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(s.Colors.TextMuted)

	// Recent items section (only if not filtering and we have recents)
	if m.input.Value() == "" && len(m.recentItems) > 0 {
		b.WriteString(sectionStyle.Render("  RECENT"))
		b.WriteString("\n")

		for i, item := range m.recentItems {
			if i >= 3 { // Show max 3 recents
				break
			}
			isSelected := i == m.cursor

			line := m.renderItem(item, isSelected, icons, itemStyle, selectedStyle, descStyle, s)
			b.WriteString(line)
			b.WriteString("\n")
		}
		b.WriteString("\n")

		// ALL section header
		b.WriteString(sectionStyle.Render("  ALL"))
		b.WriteString("\n")
	}

	// Items
	if len(m.filtered) == 0 {
		emptyStyle := lipgloss.NewStyle().
			Foreground(s.Colors.TextMuted).
			Italic(true)
		b.WriteString(emptyStyle.Render("  No matches"))
		b.WriteString("\n")
	} else {
		// Calculate visible window (adjust cursor for recents)
		visibleCursor := m.cursor
		if m.input.Value() == "" && len(m.recentItems) > 0 {
			// Cursor might be in recents section
			recentsCount := minInt(3, len(m.recentItems))
			if visibleCursor >= recentsCount {
				visibleCursor -= recentsCount
			}
		}

		start := 0
		end := len(m.filtered)
		if end > m.maxVisible {
			start = visibleCursor - m.maxVisible/2
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
			// Calculate if selected (accounting for recents offset)
			itemIdx := i
			if m.input.Value() == "" && len(m.recentItems) > 0 {
				itemIdx += minInt(3, len(m.recentItems))
			}
			isSelected := itemIdx == m.cursor

			line := m.renderItem(item, isSelected, icons, itemStyle, selectedStyle, descStyle, s)
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
		hintKeyStyle.Render("⏎") + hintDescStyle.Render(" select  ") +
		hintKeyStyle.Render("esc") + hintDescStyle.Render(" cancel")
	b.WriteString(hints)

	// Container with border
	containerStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(s.Colors.Border).
		Padding(1, 2)

	return containerStyle.Width(m.width).Render(b.String())
}

// renderItem renders a single selector item
func (m SelectorModel) renderItem(item SelectorItem, isSelected bool, icons Icons, itemStyle, selectedStyle, descStyle lipgloss.Style, s Styles) string {
	var line string
	if isSelected {
		line = s.StatusStyle("running").Render(icons.ChevronRight) + " "
		line += selectedStyle.Render(item.Title)
	} else {
		line = "  "
		line += itemStyle.Render(item.Title)
	}

	// Add description if present
	if item.Description != "" {
		line += " " + descStyle.Render(item.Description)
	}

	// Add meta tag with status coloring
	if item.Meta != "" {
		metaStyle := descStyle
		// Color-code device state
		if item.Meta == "[booted]" {
			metaStyle = s.StatusStyle("success")
		} else if item.Meta == "[shutdown]" {
			metaStyle = s.StatusStyle("idle")
		} else if item.Meta == "[device]" {
			metaStyle = s.StatusStyle("running")
		}
		line += " " + metaStyle.Render(item.Meta)
	}
	if m.showSelectedBadge && m.selectedID != "" && item.ID == m.selectedID && !strings.Contains(line, "[current]") {
		badgeStyle := s.StatusStyle("warning")
		line += " " + badgeStyle.Render("[current]")
	}

	return line
}

// =============================================================================
// Helper to create selector items from context
// =============================================================================

func normalizeConfigurations(configs []string, current string) []string {
	out := make([]string, 0, len(configs)+1)
	seen := make(map[string]struct{}, len(configs)+1)
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}

	if len(configs) == 0 {
		add("Debug")
		add("Release")
	} else {
		for _, c := range configs {
			add(c)
		}
	}
	add(current)
	return out
}

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

// ConfigurationItems creates selector items from build configurations.
func ConfigurationItems(configs []string, current string) []SelectorItem {
	list := normalizeConfigurations(configs, current)
	items := make([]SelectorItem, len(list))
	for i, c := range list {
		items[i] = SelectorItem{
			ID:    c,
			Title: c,
		}
	}
	return items
}

// DestinationItems creates selector items from simulators and devices
func DestinationItems(sims []SimulatorInfo, devices []DeviceInfo) []SelectorItem {
	items := make([]SelectorItem, 0, 1+len(sims)+len(devices))

	// Local Mac destination
	items = append(items, SelectorItem{
		ID:          "macos",
		Title:       "My Mac",
		Description: "macOS",
		Meta:        "[local]",
	})
	items = append(items, SelectorItem{
		ID:          "catalyst",
		Title:       "My Mac (Catalyst)",
		Description: "macOS",
		Meta:        "[catalyst]",
	})

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
