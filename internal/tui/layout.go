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
	StatusBarHeight int
	ProgressHeight  int
	HintsBarHeight  int
	ShowProgressBar bool
	ShowHintsBar    bool

	// Tab bar settings
	TabBarHeight int
	ShowTabBar   bool

	// Minimal mode for small terminals
	MinimalMode bool
}

// NewLayout creates a new layout with default settings
func NewLayout() Layout {
	return Layout{
		StatusBarHeight: 2, // 1 line content + 1 line border
		ProgressHeight:  1,
		HintsBarHeight:  2, // 1 line content + 1 line border
		TabBarHeight:    2, // 2 lines for tab bar
		ShowProgressBar: false,
		ShowHintsBar:    true,
		ShowTabBar:      true,
	}
}

// SetSize updates the layout dimensions
func (l *Layout) SetSize(width, height int) {
	l.Width = width
	l.Height = height

	// Enable minimal mode for small terminals
	l.MinimalMode = width < 80 || height < 20
}

// =============================================================================
// Dimension Calculations
// =============================================================================

// ContentWidth returns the full width available for content
func (l Layout) ContentWidth() int {
	return l.Width
}

// ContentHeight returns the height available for the content pane
// Note: TabView handles its own tab bar height internally, so we don't subtract it here
func (l Layout) ContentHeight() int {
	if l.MinimalMode {
		// In minimal mode: 1 line status, rest for content
		return maxInt(0, l.Height-1)
	}

	h := l.Height - l.StatusBarHeight

	if l.ShowProgressBar {
		h -= l.ProgressHeight
	}
	if l.ShowHintsBar {
		h -= l.HintsBarHeight
	}

	return maxInt(0, h)
}

// SplitTopHeight returns the height for the top pane (60%) in split view
func (l Layout) SplitTopHeight() int {
	total := l.ContentHeight()
	// Reserve 1 line for the divider
	return maxInt(0, (total-1)*60/100)
}

// SplitBottomHeight returns the height for the bottom pane (40%) in split view
func (l Layout) SplitBottomHeight() int {
	total := l.ContentHeight()
	topHeight := l.SplitTopHeight()
	// Reserve 1 line for the divider
	return maxInt(0, total-topHeight-1)
}

// =============================================================================
// Layout Rendering
// =============================================================================

// RenderHeader renders the full-width header at the top
func (l Layout) RenderHeader(content string, styles Styles) string {
	// Ensure we always have visible content
	if content == "" {
		content = "xcbolt"
	}

	// DEBUG: Simplest possible approach - raw strings, no lipgloss styling
	// This tests if the issue is with lipgloss or with the layout logic
	line1 := "=== HEADER: " + content
	line2 := strings.Repeat("─", l.Width)
	return line1 + "\n" + line2
}

// RenderProgressBar renders the progress bar below status bar
func (l Layout) RenderProgressBar(content string, styles Styles) string {
	if !l.ShowProgressBar || l.MinimalMode {
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
	if !l.ShowHintsBar || l.MinimalMode {
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
	if l.MinimalMode {
		// Minimal mode: header + content only
		var parts []string
		parts = append(parts, l.RenderHeader(statusBar, styles))

		contentStyle := lipgloss.NewStyle().
			Width(l.Width).
			Height(l.ContentHeight())
		parts = append(parts, contentStyle.Render(content))

		return lipgloss.JoinVertical(lipgloss.Left, parts...)
	}

	var parts []string

	// Header at top
	parts = append(parts, l.RenderHeader(statusBar, styles))

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

// RenderSplitLayout renders a split view with top and bottom panes
func (l Layout) RenderSplitLayout(statusBar, progressBar, topContent, bottomContent, hintsBar string, topFocused bool, styles Styles) string {
	if l.MinimalMode {
		// Minimal mode: just show top content (no split)
		return l.RenderFullLayout(statusBar, progressBar, topContent, hintsBar, styles)
	}

	var parts []string

	// Header at top
	parts = append(parts, l.RenderHeader(statusBar, styles))

	// Progress bar (if visible)
	if l.ShowProgressBar && progressBar != "" {
		parts = append(parts, l.RenderProgressBar(progressBar, styles))
	}

	// Top pane (Build Logs)
	topHeight := l.SplitTopHeight()
	topStyle := lipgloss.NewStyle().
		Width(l.Width).
		Height(topHeight)
	parts = append(parts, topStyle.Render(topContent))

	// Divider with focus indicator
	dividerStyle := lipgloss.NewStyle().Foreground(styles.Colors.BorderMuted)
	dividerChar := "─"
	divider := strings.Repeat(dividerChar, l.Width)

	// Show pane labels in divider
	topLabel := " Build "
	bottomLabel := " Console "
	if topFocused {
		topLabel = lipgloss.NewStyle().Foreground(styles.Colors.Accent).Bold(true).Render("●") + " Build "
	} else {
		bottomLabel = lipgloss.NewStyle().Foreground(styles.Colors.Accent).Bold(true).Render("●") + " Console "
	}

	// Build divider with labels
	labelStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)
	leftLabel := labelStyle.Render(topLabel)
	rightLabel := labelStyle.Render(bottomLabel)

	// Calculate remaining space
	leftWidth := lipgloss.Width(leftLabel)
	rightWidth := lipgloss.Width(rightLabel)
	middleWidth := l.Width - leftWidth - rightWidth - 4 // 4 for spacing

	if middleWidth > 0 {
		middleDivider := dividerStyle.Render(strings.Repeat(dividerChar, middleWidth))
		parts = append(parts, leftLabel+"─"+middleDivider+"─"+rightLabel)
	} else {
		parts = append(parts, dividerStyle.Render(divider))
	}

	// Bottom pane (Console)
	bottomHeight := l.SplitBottomHeight()
	bottomStyle := lipgloss.NewStyle().
		Width(l.Width).
		Height(bottomHeight)
	parts = append(parts, bottomStyle.Render(bottomContent))

	// Hints bar at bottom
	if l.ShowHintsBar {
		parts = append(parts, l.RenderHintsBar(hintsBar, styles))
	}

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}
