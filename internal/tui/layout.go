package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Pane - Represents a focusable pane in the layout
// =============================================================================

// Pane identifies different panes in the split layout
type Pane int

const (
	PaneSidebar Pane = iota
	PaneContent
)

// =============================================================================
// Layout - Split pane layout manager
// =============================================================================

// Layout manages the split-pane layout with sidebar and content area
type Layout struct {
	// Dimensions
	Width  int
	Height int

	// Sidebar configuration
	SidebarWidth    int
	SidebarVisible  bool
	SidebarCollapse bool // When true, show icon-only rail

	// Reserved heights
	StatusBarHeight int
	ProgressHeight  int
	HintsBarHeight  int
	ShowProgressBar bool
	ShowHintsBar    bool

	// Focus
	FocusedPane Pane
}

// NewLayout creates a new layout with default settings
func NewLayout() Layout {
	return Layout{
		SidebarWidth:    32, // 30-35 chars as per spec
		SidebarVisible:  true,
		SidebarCollapse: false,
		StatusBarHeight: 1,
		ProgressHeight:  1,
		HintsBarHeight:  1,
		ShowProgressBar: false,
		ShowHintsBar:    true,
		FocusedPane:     PaneSidebar,
	}
}

// SetSize updates the layout dimensions
func (l *Layout) SetSize(width, height int) {
	l.Width = width
	l.Height = height

	// Responsive behavior
	if width < 80 {
		// Too narrow - collapse sidebar
		l.SidebarCollapse = true
		l.SidebarWidth = 4 // Icon rail
	} else if width < 100 {
		// Compact mode
		l.SidebarCollapse = false
		l.SidebarWidth = 24
	} else {
		// Full mode
		l.SidebarCollapse = false
		l.SidebarWidth = 32
	}
}

// ToggleSidebar toggles sidebar visibility
func (l *Layout) ToggleSidebar() {
	l.SidebarVisible = !l.SidebarVisible
}

// SwitchFocus switches focus to the next pane
func (l *Layout) SwitchFocus() {
	if l.FocusedPane == PaneSidebar {
		l.FocusedPane = PaneContent
	} else {
		l.FocusedPane = PaneSidebar
	}
}

// =============================================================================
// Dimension Calculations
// =============================================================================

// ContentWidth returns the width available for the content pane
func (l Layout) ContentWidth() int {
	if !l.SidebarVisible {
		return l.Width
	}
	// Account for sidebar + border
	return l.Width - l.SidebarWidth - 1
}

// ContentHeight returns the height available for the content pane
func (l Layout) ContentHeight() int {
	h := l.Height - l.StatusBarHeight

	if l.ShowProgressBar {
		h -= l.ProgressHeight
	}
	if l.ShowHintsBar {
		h -= l.HintsBarHeight
	}

	return maxInt(0, h)
}

// SidebarHeight returns the height available for the sidebar
func (l Layout) SidebarHeight() int {
	return l.ContentHeight()
}

// EffectiveSidebarWidth returns the actual sidebar width (0 if hidden)
func (l Layout) EffectiveSidebarWidth() int {
	if !l.SidebarVisible {
		return 0
	}
	return l.SidebarWidth
}

// =============================================================================
// Layout Rendering
// =============================================================================

// RenderSplitView renders the sidebar and content side by side
func (l Layout) RenderSplitView(sidebar, content string, styles Styles) string {
	if !l.SidebarVisible {
		return content
	}

	// Style the sidebar with border
	sidebarStyle := lipgloss.NewStyle().
		Width(l.SidebarWidth).
		Height(l.ContentHeight()).
		BorderStyle(lipgloss.Border{Right: "│"}).
		BorderForeground(styles.Colors.Border).
		BorderRight(true)

	// Style the content area
	contentStyle := lipgloss.NewStyle().
		Width(l.ContentWidth()).
		Height(l.ContentHeight())

	// Add focus indicator
	if l.FocusedPane == PaneSidebar {
		sidebarStyle = sidebarStyle.BorderForeground(styles.Colors.Accent)
	}

	styledSidebar := sidebarStyle.Render(sidebar)
	styledContent := contentStyle.Render(content)

	return lipgloss.JoinHorizontal(lipgloss.Top, styledSidebar, styledContent)
}

// RenderStatusBar renders the full-width status bar at the top
func (l Layout) RenderStatusBar(content string, styles Styles) string {
	style := lipgloss.NewStyle().
		Width(l.Width).
		Height(l.StatusBarHeight).
		Padding(0, 1).
		BorderStyle(lipgloss.Border{Bottom: "─"}).
		BorderForeground(styles.Colors.Border).
		BorderBottom(true)

	return style.Render(content)
}

// RenderProgressBar renders the progress bar below status bar
func (l Layout) RenderProgressBar(content string, styles Styles) string {
	if !l.ShowProgressBar {
		return ""
	}

	style := lipgloss.NewStyle().
		Width(l.Width).
		Height(l.ProgressHeight).
		Padding(0, 1)

	return style.Render(content)
}

// RenderHintsBar renders the hints bar at the bottom
func (l Layout) RenderHintsBar(content string, styles Styles) string {
	if !l.ShowHintsBar {
		return ""
	}

	style := lipgloss.NewStyle().
		Width(l.Width).
		Height(l.HintsBarHeight).
		Padding(0, 1).
		BorderStyle(lipgloss.Border{Top: "─"}).
		BorderForeground(styles.Colors.BorderMuted).
		BorderTop(true).
		Foreground(styles.Colors.TextSubtle)

	return style.Render(content)
}

// RenderFullLayout composes all layout elements
func (l Layout) RenderFullLayout(statusBar, progressBar, sidebar, content, hintsBar string, styles Styles) string {
	var parts []string

	// Status bar at top
	parts = append(parts, l.RenderStatusBar(statusBar, styles))

	// Progress bar (if visible)
	if l.ShowProgressBar && progressBar != "" {
		parts = append(parts, l.RenderProgressBar(progressBar, styles))
	}

	// Split view (sidebar + content)
	splitView := l.RenderSplitView(sidebar, content, styles)
	parts = append(parts, splitView)

	// Hints bar at bottom
	if l.ShowHintsBar {
		parts = append(parts, l.RenderHintsBar(hintsBar, styles))
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

// =============================================================================
// Focus Styles
// =============================================================================

// FocusedBorderColor returns the border color based on focus state
func (l Layout) FocusedBorderColor(pane Pane, styles Styles) lipgloss.AdaptiveColor {
	if l.FocusedPane == pane {
		return styles.Colors.Accent
	}
	return styles.Colors.Border
}

// IsFocused returns true if the given pane is focused
func (l Layout) IsFocused(pane Pane) bool {
	return l.FocusedPane == pane
}
