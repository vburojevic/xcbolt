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
		TabBarHeight:    3, // 2 content lines + 1 border line
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
// Note: This is a rough estimate - actual height is calculated dynamically in RenderFullLayout
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
// IMPORTANT: Forces single-line to prevent height changes causing flicker
func (l Layout) RenderHeader(content string, styles Styles) string {
	// Ensure we always have visible content
	if content == "" {
		content = "xcbolt"
	}

	if l.MinimalMode {
		// Minimal mode: single line, fixed width
		return lipgloss.NewStyle().
			Width(l.Width).
			MaxWidth(l.Width).
			MaxHeight(1).
			Render(" " + content)
	}

	// Full mode: content line (single-line, clipped) + separator
	// Force single-line to prevent wrapping which causes flicker
	contentLine := lipgloss.NewStyle().
		Width(l.Width).
		MaxWidth(l.Width).
		MaxHeight(1).
		Padding(0, 1).
		Render(content)

	separator := lipgloss.NewStyle().
		Foreground(styles.Colors.Border).
		Render(strings.Repeat("─", l.Width))

	return contentLine + "\n" + separator
}

// RenderProgressBar renders the progress bar below status bar
func (l Layout) RenderProgressBar(content string, styles Styles) string {
	if !l.ShowProgressBar || l.MinimalMode {
		return ""
	}

	return " " + content
}

// RenderHintsBar renders the hints bar at the bottom
func (l Layout) RenderHintsBar(content string, styles Styles) string {
	if !l.ShowHintsBar {
		return ""
	}

	if l.MinimalMode {
		return lipgloss.NewStyle().
			Foreground(styles.Colors.TextSubtle).
			Width(l.Width).
			MaxWidth(l.Width).
			MaxHeight(1).
			Render(" " + content)
	}

	separator := lipgloss.NewStyle().
		Foreground(styles.Colors.BorderMuted).
		Render(strings.Repeat("─", l.Width))

	contentLine := lipgloss.NewStyle().
		Foreground(styles.Colors.TextSubtle).
		Width(l.Width).
		MaxWidth(l.Width).
		MaxHeight(1).
		Render(" " + content)

	return separator + "\n" + contentLine
}

// RenderFullLayout composes all layout elements
// KEY FIX: Calculate content height DYNAMICALLY based on actual rendered header/hints heights
func (l Layout) RenderFullLayout(statusBar, progressBar, content, hintsBar string, styles Styles) string {
	if l.MinimalMode {
		// Minimal mode: header + content + optional hints
		header := l.RenderHeader(statusBar, styles)
		hints := l.RenderHintsBar(hintsBar, styles)
		headerHeight := lipgloss.Height(header)
		hintsHeight := lipgloss.Height(hints)

		contentHeight := maxInt(0, l.Height-headerHeight-hintsHeight)
		contentStyle := lipgloss.NewStyle().
			Width(l.Width).
			Height(contentHeight).
			MaxHeight(contentHeight)

		renderedContent := contentStyle.Render(content)
		if hints != "" {
			return lipgloss.JoinVertical(lipgloss.Left,
				header,
				renderedContent,
				hints,
			)
		}

		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			renderedContent,
		)
	}

	// Render header and hints first to measure their actual heights
	header := l.RenderHeader(statusBar, styles)
	hints := l.RenderHintsBar(hintsBar, styles)

	headerHeight := lipgloss.Height(header)
	hintsHeight := lipgloss.Height(hints)

	// Calculate remaining height for content
	contentHeight := l.Height - headerHeight - hintsHeight

	// Account for progress bar if visible
	var progress string
	if l.ShowProgressBar && progressBar != "" {
		progress = l.RenderProgressBar(progressBar, styles)
		contentHeight -= lipgloss.Height(progress)
	}

	contentHeight = maxInt(0, contentHeight)

	// Render content with calculated height
	contentStyle := lipgloss.NewStyle().
		Width(l.Width).
		Height(contentHeight).
		MaxHeight(contentHeight)
	renderedContent := contentStyle.Render(content)

	// Join all parts
	if progress != "" {
		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			progress,
			renderedContent,
			hints,
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		renderedContent,
		hints,
	)
}

// RenderSplitLayout renders a split view with top and bottom panes
func (l Layout) RenderSplitLayout(statusBar, progressBar, topContent, bottomContent, hintsBar string, topFocused bool, styles Styles) string {
	if l.MinimalMode {
		// Minimal mode: just show top content (no split)
		return l.RenderFullLayout(statusBar, progressBar, topContent, hintsBar, styles)
	}

	// Render header and hints first to measure their actual heights
	header := l.RenderHeader(statusBar, styles)
	hints := l.RenderHintsBar(hintsBar, styles)

	headerHeight := lipgloss.Height(header)
	hintsHeight := lipgloss.Height(hints)

	// Calculate remaining height for content (minus 1 for divider)
	totalContentHeight := l.Height - headerHeight - hintsHeight - 1

	// Account for progress bar if visible
	var progress string
	if l.ShowProgressBar && progressBar != "" {
		progress = l.RenderProgressBar(progressBar, styles)
		totalContentHeight -= lipgloss.Height(progress)
	}

	totalContentHeight = maxInt(0, totalContentHeight)

	// Split 60/40
	topHeight := totalContentHeight * 60 / 100
	bottomHeight := totalContentHeight - topHeight

	// Render panes
	topStyle := lipgloss.NewStyle().Width(l.Width).Height(topHeight).MaxHeight(topHeight)
	bottomStyle := lipgloss.NewStyle().Width(l.Width).Height(bottomHeight).MaxHeight(bottomHeight)

	renderedTop := topStyle.Render(topContent)
	renderedBottom := bottomStyle.Render(bottomContent)

	// Build divider with labels
	dividerStyle := lipgloss.NewStyle().Foreground(styles.Colors.BorderMuted)
	dividerChar := "─"

	topLabel := " Build "
	bottomLabel := " Console "
	if topFocused {
		topLabel = lipgloss.NewStyle().Foreground(styles.Colors.Accent).Bold(true).Render("●") + " Build "
	} else {
		bottomLabel = lipgloss.NewStyle().Foreground(styles.Colors.Accent).Bold(true).Render("●") + " Console "
	}

	labelStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)
	leftLabel := labelStyle.Render(topLabel)
	rightLabel := labelStyle.Render(bottomLabel)

	leftWidth := lipgloss.Width(leftLabel)
	rightWidth := lipgloss.Width(rightLabel)
	middleWidth := l.Width - leftWidth - rightWidth - 4

	var divider string
	if middleWidth > 0 {
		middleDivider := dividerStyle.Render(strings.Repeat(dividerChar, middleWidth))
		divider = leftLabel + "─" + middleDivider + "─" + rightLabel
	} else {
		divider = dividerStyle.Render(strings.Repeat(dividerChar, l.Width))
	}

	// Join all parts
	if progress != "" {
		return lipgloss.JoinVertical(lipgloss.Left,
			header,
			progress,
			renderedTop,
			divider,
			renderedBottom,
			hints,
		)
	}

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		renderedTop,
		divider,
		renderedBottom,
		hints,
	)
}
