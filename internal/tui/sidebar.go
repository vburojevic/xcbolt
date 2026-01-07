package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Sidebar Item
// =============================================================================

// SidebarItemType categorizes sidebar items
type SidebarItemType int

const (
	SidebarItemAction SidebarItemType = iota
	SidebarItemConfig
	SidebarItemRecent
)

// SidebarItemStatus represents the status of an action item
type SidebarItemStatus int

const (
	StatusNone SidebarItemStatus = iota
	StatusRunning
	StatusSuccess
	StatusError
)

// SidebarItem represents a single item in the sidebar
type SidebarItem struct {
	ID       string
	Label    string
	Shortcut string
	Type     SidebarItemType
	Status   SidebarItemStatus
	Value    string // For config items, shows current value
	Meta     string // Additional info (e.g., "47/50" for tests)
}

// =============================================================================
// Sidebar Section
// =============================================================================

// SidebarSection groups related items
type SidebarSection struct {
	Title string
	Items []SidebarItem
}

// =============================================================================
// Sidebar Model
// =============================================================================

// Sidebar manages the left sidebar state and rendering
type Sidebar struct {
	Sections []SidebarSection
	Selected int // Currently selected item index (global)
	Width    int
	Height   int
	Focused  bool
	Compact  bool // Icon-only mode
}

// NewSidebar creates a new sidebar with default sections
func NewSidebar() Sidebar {
	return Sidebar{
		Sections: []SidebarSection{
			{
				Title: "ACTIONS",
				Items: []SidebarItem{
					{ID: "build", Label: "Build", Shortcut: "b", Type: SidebarItemAction},
					{ID: "run", Label: "Run", Shortcut: "r", Type: SidebarItemAction},
					{ID: "test", Label: "Test", Shortcut: "t", Type: SidebarItemAction},
					{ID: "clean", Label: "Clean", Shortcut: "c", Type: SidebarItemAction},
				},
			},
			{
				Title: "CONFIG",
				Items: []SidebarItem{
					{ID: "scheme", Label: "Scheme", Shortcut: "s", Type: SidebarItemConfig},
					{ID: "destination", Label: "Destination", Shortcut: "d", Type: SidebarItemConfig},
				},
			},
			{
				Title: "RECENT",
				Items: []SidebarItem{
					// Will be populated dynamically
				},
			},
		},
		Selected: 0,
		Focused:  true,
	}
}

// =============================================================================
// Item Navigation
// =============================================================================

// TotalItems returns the total number of selectable items
func (s Sidebar) TotalItems() int {
	count := 0
	for _, section := range s.Sections {
		count += len(section.Items)
	}
	return count
}

// MoveUp moves selection up
func (s *Sidebar) MoveUp() {
	if s.Selected > 0 {
		s.Selected--
	}
}

// MoveDown moves selection down
func (s *Sidebar) MoveDown() {
	if s.Selected < s.TotalItems()-1 {
		s.Selected++
	}
}

// SelectedItem returns the currently selected item
func (s Sidebar) SelectedItem() *SidebarItem {
	idx := 0
	for i := range s.Sections {
		for j := range s.Sections[i].Items {
			if idx == s.Selected {
				return &s.Sections[i].Items[j]
			}
			idx++
		}
	}
	return nil
}

// SelectByID selects an item by its ID
func (s *Sidebar) SelectByID(id string) bool {
	idx := 0
	for _, section := range s.Sections {
		for _, item := range section.Items {
			if item.ID == id {
				s.Selected = idx
				return true
			}
			idx++
		}
	}
	return false
}

// =============================================================================
// State Updates
// =============================================================================

// SetItemStatus updates the status of an action item
func (s *Sidebar) SetItemStatus(id string, status SidebarItemStatus) {
	for i := range s.Sections {
		for j := range s.Sections[i].Items {
			if s.Sections[i].Items[j].ID == id {
				s.Sections[i].Items[j].Status = status
				return
			}
		}
	}
}

// SetItemMeta updates the meta info of an item
func (s *Sidebar) SetItemMeta(id string, meta string) {
	for i := range s.Sections {
		for j := range s.Sections[i].Items {
			if s.Sections[i].Items[j].ID == id {
				s.Sections[i].Items[j].Meta = meta
				return
			}
		}
	}
}

// SetConfigValue updates the value shown for a config item
func (s *Sidebar) SetConfigValue(id string, value string) {
	for i := range s.Sections {
		for j := range s.Sections[i].Items {
			if s.Sections[i].Items[j].ID == id {
				s.Sections[i].Items[j].Value = value
				return
			}
		}
	}
}

// SetRecents updates the recent combos section
func (s *Sidebar) SetRecents(recents []SidebarItem) {
	for i := range s.Sections {
		if s.Sections[i].Title == "RECENT" {
			s.Sections[i].Items = recents
			return
		}
	}
}

// ClearAllStatus resets all action statuses to none
func (s *Sidebar) ClearAllStatus() {
	for i := range s.Sections {
		for j := range s.Sections[i].Items {
			if s.Sections[i].Items[j].Type == SidebarItemAction {
				s.Sections[i].Items[j].Status = StatusNone
			}
		}
	}
}

// =============================================================================
// Rendering
// =============================================================================

// View renders the sidebar
func (s Sidebar) View(styles Styles) string {
	var b strings.Builder

	globalIdx := 0

	for sectionIdx, section := range s.Sections {
		// Skip empty sections
		if len(section.Items) == 0 {
			continue
		}

		// Section title
		if !s.Compact {
			titleStyle := lipgloss.NewStyle().
				Foreground(styles.Colors.TextSubtle).
				Bold(true).
				MarginTop(func() int {
					if sectionIdx == 0 {
						return 0
					}
					return 1
				}())
			b.WriteString(titleStyle.Render(section.Title))
			b.WriteString("\n")
		}

		// Section items
		for _, item := range section.Items {
			isSelected := globalIdx == s.Selected && s.Focused

			line := s.renderItem(item, isSelected, styles)
			b.WriteString(line)
			b.WriteString("\n")
			globalIdx++
		}
	}

	return strings.TrimSuffix(b.String(), "\n")
}

// renderItem renders a single sidebar item
func (s Sidebar) renderItem(item SidebarItem, selected bool, styles Styles) string {
	icons := styles.Icons

	// Compact mode - just icons
	if s.Compact {
		icon := s.getItemIcon(item, styles)
		if selected {
			return styles.Selector.ItemSelected.Render(icon)
		}
		return styles.Selector.Item.Render(icon)
	}

	var parts []string

	// Selection indicator
	if selected {
		parts = append(parts, styles.StatusStyle("running").Render(icons.ChevronRight))
	} else {
		parts = append(parts, "  ")
	}

	// Status icon for action items
	if item.Type == SidebarItemAction {
		statusIcon := s.getStatusIcon(item.Status, styles)
		parts = append(parts, statusIcon)
	}

	// Label
	labelStyle := styles.Selector.Item
	if selected {
		labelStyle = styles.Selector.ItemSelected
	}
	parts = append(parts, labelStyle.Render(item.Label))

	// Shortcut
	if item.Shortcut != "" {
		shortcutStyle := lipgloss.NewStyle().
			Foreground(styles.Colors.TextSubtle).
			Width(4).
			Align(lipgloss.Right)
		parts = append(parts, shortcutStyle.Render("["+item.Shortcut+"]"))
	}

	// Value (for config items) or Meta (for action items)
	if item.Value != "" {
		valueStyle := lipgloss.NewStyle().
			Foreground(styles.Colors.TextMuted).
			MaxWidth(s.Width - 20)
		// Truncate if needed
		value := truncateString(item.Value, s.Width-20)
		parts = append(parts, valueStyle.Render(value))
	} else if item.Meta != "" {
		metaStyle := lipgloss.NewStyle().
			Foreground(styles.Colors.TextMuted)
		parts = append(parts, metaStyle.Render(item.Meta))
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

// getStatusIcon returns the appropriate status icon
func (s Sidebar) getStatusIcon(status SidebarItemStatus, styles Styles) string {
	icons := styles.Icons
	switch status {
	case StatusRunning:
		return styles.StatusStyle("running").Render(icons.Running) + " "
	case StatusSuccess:
		return styles.StatusStyle("success").Render(icons.Success) + " "
	case StatusError:
		return styles.StatusStyle("error").Render(icons.Error) + " "
	default:
		return "  " // Placeholder for alignment
	}
}

// getItemIcon returns the icon for an item (used in compact mode)
func (s Sidebar) getItemIcon(item SidebarItem, styles Styles) string {
	icons := styles.Icons
	switch item.ID {
	case "build":
		return icons.Build
	case "run":
		return icons.Run
	case "test":
		return icons.Test
	case "clean":
		return icons.Clean
	case "scheme":
		return icons.Settings
	case "destination":
		return icons.Settings
	default:
		return icons.ChevronRight
	}
}

// =============================================================================
// Helpers
// =============================================================================

// truncateString truncates a string to maxLen with ellipsis
func truncateString(s string, maxLen int) string {
	if maxLen <= 3 {
		return s
	}
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
