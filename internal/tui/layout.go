package tui

import (
	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Layout - Single-pane layout manager
// =============================================================================

// Layout manages the full-width layout with status bar, content, and hints
type Layout struct {
	// Dimensions
	Width  int
	Height int

	// Reserved heights
	StatusBarHeight int
	ProgressHeight  int
	HintsBarHeight  int
	ShowProgressBar bool
	ShowHintsBar    bool
}

// NewLayout creates a new layout with default settings
func NewLayout() Layout {
	return Layout{
		StatusBarHeight: 1,
		ProgressHeight:  1,
		HintsBarHeight:  1,
		ShowProgressBar: false,
		ShowHintsBar:    true,
	}
}

// SetSize updates the layout dimensions
func (l *Layout) SetSize(width, height int) {
	l.Width = width
	l.Height = height
}

// =============================================================================
// Dimension Calculations
// =============================================================================

// ContentWidth returns the full width available for content
func (l Layout) ContentWidth() int {
	return l.Width
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

// =============================================================================
// Layout Rendering
// =============================================================================

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
func (l Layout) RenderFullLayout(statusBar, progressBar, content, hintsBar string, styles Styles) string {
	var parts []string

	// Status bar at top
	parts = append(parts, l.RenderStatusBar(statusBar, styles))

	// Progress bar (if visible)
	if l.ShowProgressBar && progressBar != "" {
		parts = append(parts, l.RenderProgressBar(progressBar, styles))
	}

	// Content area (full width)
	contentStyle := lipgloss.NewStyle().
		Width(l.Width).
		Height(l.ContentHeight())
	parts = append(parts, contentStyle.Render(content))

	// Hints bar at bottom
	if l.ShowHintsBar {
		parts = append(parts, l.RenderHintsBar(hintsBar, styles))
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}
