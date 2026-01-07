package tui

import (
	"strings"

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
	StatusBarHeight   int
	ProgressHeight    int
	HintsBarHeight    int
	IssuesPanelHeight int
	ShowProgressBar   bool
	ShowHintsBar      bool
	ShowIssuesPanel   bool
}

// NewLayout creates a new layout with default settings
func NewLayout() Layout {
	return Layout{
		StatusBarHeight: 2, // 1 line content + 1 line border
		ProgressHeight:  1,
		HintsBarHeight:  2, // 1 line content + 1 line border
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
	if l.ShowIssuesPanel {
		h -= l.IssuesPanelHeight
	}
	if l.ShowHintsBar {
		h -= l.HintsBarHeight
	}

	return maxInt(0, h)
}

// CalculateIssuesPanelHeight calculates adaptive height based on issue count
func (l Layout) CalculateIssuesPanelHeight(issueCount int) int {
	// 1 line header + 1 line separator + issues + possible "more" line
	desired := 3 + minInt(issueCount, 5)

	// Minimum 4 lines, max 40% of screen
	minHeight := 4
	maxHeight := l.Height * 40 / 100

	if desired < minHeight {
		desired = minHeight
	}
	if desired > maxHeight {
		desired = maxHeight
	}
	return desired
}

// =============================================================================
// Layout Rendering
// =============================================================================

// RenderStatusBar renders the full-width status bar at the top
func (l Layout) RenderStatusBar(content string, styles Styles) string {
	// Content is pre-styled from StatusBar.View() - don't wrap in Width()
	// which can conflict with already-styled ANSI content
	paddedContent := lipgloss.NewStyle().
		Padding(0, 1).
		Render(content)

	// Border line
	border := lipgloss.NewStyle().
		Foreground(styles.Colors.Border).
		Render(strings.Repeat("─", l.Width))

	return paddedContent + "\n" + border
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

// RenderIssuesPanel renders the issues panel between content and hints bar
func (l Layout) RenderIssuesPanel(content string, styles Styles) string {
	if !l.ShowIssuesPanel {
		return ""
	}

	style := lipgloss.NewStyle().
		Width(l.Width).
		Height(l.IssuesPanelHeight).
		Padding(0, 1).
		BorderStyle(lipgloss.Border{Top: "─"}).
		BorderForeground(styles.Colors.Border).
		BorderTop(true)

	return style.Render(content)
}

// RenderFullLayout composes all layout elements
func (l Layout) RenderFullLayout(statusBar, progressBar, content, issuesPanel, hintsBar string, styles Styles) string {
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

	// Issues panel (if visible)
	if l.ShowIssuesPanel && issuesPanel != "" {
		parts = append(parts, l.RenderIssuesPanel(issuesPanel, styles))
	}

	// Hints bar at bottom
	if l.ShowHintsBar {
		parts = append(parts, l.RenderHintsBar(hintsBar, styles))
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}
