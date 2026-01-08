package tui

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// StreamTab - Enhanced raw log stream
// =============================================================================

// StreamLine represents a single line in the stream
type StreamLine struct {
	Text      string
	Timestamp time.Time
	Type      TabLineType
	Raw       string // Original unformatted text
}

// StreamTab displays the live log stream with enhancements
type StreamTab struct {
	Lines []StreamLine

	// Scroll state
	ScrollPos   int
	AutoFollow  bool
	VisibleRows int

	// Display settings
	ShowLineNumbers bool
	ShowTimestamps  bool
	PathStyle       string // "full", "short", "filename"

	// Dimensions
	Width  int
	Height int

	// Regex patterns for syntax highlighting
	filePathRegex *regexp.Regexp
	urlRegex      *regexp.Regexp
	errorRegex    *regexp.Regexp
	warningRegex  *regexp.Regexp
}

// NewStreamTab creates a new StreamTab
func NewStreamTab() *StreamTab {
	return &StreamTab{
		Lines:           make([]StreamLine, 0, 1000),
		AutoFollow:      true,
		ShowLineNumbers: true,
		ShowTimestamps:  false,
		PathStyle:       "full", // Don't shorten paths - show full for clarity
		// Match file paths: /path/to/file.ext or /path/to/dir (with optional :line:col)
		filePathRegex: regexp.MustCompile(`(/[^\s:]+(?:\.[a-zA-Z0-9]+)?)(?::(\d+)(?::(\d+))?)?`),
		// Match URLs: https://... or http://...
		urlRegex:     regexp.MustCompile(`https?://[^\s]+`),
		errorRegex:   regexp.MustCompile(`(?i)(error:|fatal error|failed)`),
		warningRegex: regexp.MustCompile(`(?i)(warning:)`),
	}
}

// SetSize updates dimensions
func (st *StreamTab) SetSize(width, height int) {
	st.Width = width
	st.Height = height
	st.VisibleRows = height
}

// Clear resets the stream
func (st *StreamTab) Clear() {
	st.Lines = st.Lines[:0]
	st.ScrollPos = 0
	st.AutoFollow = true
}

// AddLine adds a new line to the stream
func (st *StreamTab) AddLine(text string, lineType TabLineType) {
	line := StreamLine{
		Text:      text,
		Timestamp: time.Now(),
		Type:      lineType,
		Raw:       text,
	}
	st.Lines = append(st.Lines, line)

	// Auto-scroll if following
	if st.AutoFollow {
		st.ScrollPos = st.maxScrollPos()
	}
}

// =============================================================================
// Scrolling
// =============================================================================

func (st *StreamTab) maxScrollPos() int {
	max := len(st.Lines) - st.VisibleRows
	if max < 0 {
		return 0
	}
	return max
}

// ScrollUp scrolls up by n lines
func (st *StreamTab) ScrollUp(n int) {
	st.AutoFollow = false
	st.ScrollPos -= n
	if st.ScrollPos < 0 {
		st.ScrollPos = 0
	}
}

// ScrollDown scrolls down by n lines
func (st *StreamTab) ScrollDown(n int) {
	st.ScrollPos += n
	max := st.maxScrollPos()
	if st.ScrollPos >= max {
		st.ScrollPos = max
		st.AutoFollow = true
	}
}

// GotoTop scrolls to the top
func (st *StreamTab) GotoTop() {
	st.AutoFollow = false
	st.ScrollPos = 0
}

// GotoBottom scrolls to the bottom
func (st *StreamTab) GotoBottom() {
	st.ScrollPos = st.maxScrollPos()
	st.AutoFollow = true
}

// =============================================================================
// View Rendering
// =============================================================================

// View renders the stream tab content
func (st *StreamTab) View(styles Styles) string {
	if len(st.Lines) == 0 {
		return st.emptyView(styles)
	}

	// Calculate visible range
	start := st.ScrollPos
	end := start + st.VisibleRows
	if end > len(st.Lines) {
		end = len(st.Lines)
	}

	// Render visible lines
	var lines []string
	gutterWidth := 0
	if st.ShowLineNumbers {
		gutterWidth = len(fmt.Sprintf("%d", len(st.Lines))) + 1
	}
	if st.ShowTimestamps {
		gutterWidth += 9 // "HH:MM:SS "
	}

	contentWidth := st.Width - gutterWidth - 2 // 2 for padding

	for i := start; i < end; i++ {
		line := st.Lines[i]
		rendered := st.renderLine(i+1, line, gutterWidth, contentWidth, styles)
		lines = append(lines, rendered)
	}

	// Pad to fill height
	for len(lines) < st.VisibleRows {
		lines = append(lines, "")
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)

	// Add auto-follow indicator
	if st.AutoFollow {
		indicator := lipgloss.NewStyle().
			Foreground(styles.Colors.Accent).
			Render(" AUTO")
		// Position in bottom-right (simplified - just append to last line)
		_ = indicator // TODO: position properly
	}

	return content
}

// renderLine renders a single line with syntax highlighting
func (st *StreamTab) renderLine(lineNum int, line StreamLine, gutterWidth, contentWidth int, styles Styles) string {
	syntax := styles.Syntax
	var parts []string

	// Gutter (line number + timestamp)
	if st.ShowLineNumbers || st.ShowTimestamps {
		gutter := st.renderGutter(lineNum, line, gutterWidth, syntax)
		parts = append(parts, gutter)
	}

	// Content with syntax highlighting
	content := st.highlightLine(line, contentWidth, styles)
	parts = append(parts, content)

	return lipgloss.JoinHorizontal(lipgloss.Top, parts...)
}

// renderGutter renders the line number and/or timestamp
func (st *StreamTab) renderGutter(lineNum int, line StreamLine, width int, syntax SyntaxColors) string {
	var gutter string

	if st.ShowLineNumbers {
		numStyle := lipgloss.NewStyle().
			Foreground(syntax.LineNumber).
			Width(4).
			Align(lipgloss.Right)
		gutter = numStyle.Render(fmt.Sprintf("%d", lineNum))
	}

	if st.ShowTimestamps {
		tsStyle := lipgloss.NewStyle().Foreground(syntax.Timestamp)
		ts := line.Timestamp.Format("15:04:05")
		if gutter != "" {
			gutter += " "
		}
		gutter += tsStyle.Render(ts)
	}

	if gutter != "" {
		gutter += " "
	}

	return gutter
}

// highlightLine applies syntax highlighting to a line
func (st *StreamTab) highlightLine(line StreamLine, maxWidth int, styles Styles) string {
	text := line.Text
	syntax := styles.Syntax
	colors := styles.Colors

	// Truncate if needed
	if len(text) > maxWidth && maxWidth > 3 {
		text = text[:maxWidth-3] + "..."
	}

	// Apply base style based on line type
	var style lipgloss.Style
	switch line.Type {
	case TabLineTypeError:
		style = lipgloss.NewStyle().Foreground(colors.Error)
	case TabLineTypeWarning:
		style = lipgloss.NewStyle().Foreground(colors.Warning)
	case TabLineTypeNote:
		style = lipgloss.NewStyle().Foreground(syntax.Comment)
	case TabLineTypeVerbose:
		style = lipgloss.NewStyle().Foreground(syntax.Verbose)
	case TabLineTypePhaseHeader:
		style = lipgloss.NewStyle().Foreground(colors.Accent).Bold(true)
	case TabLineTypeProgress:
		style = lipgloss.NewStyle().Foreground(colors.Running)
	default:
		style = lipgloss.NewStyle().Foreground(colors.Text)
	}

	// For non-error lines, try to highlight file paths
	if line.Type != TabLineTypeError && line.Type != TabLineTypeWarning {
		highlighted := st.highlightFilePaths(text, syntax)
		if highlighted != text {
			return highlighted
		}
	}

	return style.Render(text)
}

// highlightFilePaths highlights file paths and URLs in a line
func (st *StreamTab) highlightFilePaths(text string, syntax SyntaxColors) string {
	pathStyle := lipgloss.NewStyle().Foreground(syntax.FilePath)

	// Collect all matches with their positions
	type match struct {
		start, end int
		text       string
	}
	var allMatches []match

	// Find URL matches first (they take priority)
	urlMatches := st.urlRegex.FindAllStringIndex(text, -1)
	for _, m := range urlMatches {
		allMatches = append(allMatches, match{start: m[0], end: m[1], text: text[m[0]:m[1]]})
	}

	// Find file path matches
	pathMatches := st.filePathRegex.FindAllStringIndex(text, -1)
	for _, m := range pathMatches {
		// Skip if this overlaps with a URL match
		overlaps := false
		for _, um := range allMatches {
			if m[0] < um.end && m[1] > um.start {
				overlaps = true
				break
			}
		}
		if !overlaps {
			allMatches = append(allMatches, match{start: m[0], end: m[1], text: text[m[0]:m[1]]})
		}
	}

	if len(allMatches) == 0 {
		return text
	}

	// Sort matches by start position (descending) to process from end
	for i := 0; i < len(allMatches)-1; i++ {
		for j := i + 1; j < len(allMatches); j++ {
			if allMatches[i].start < allMatches[j].start {
				allMatches[i], allMatches[j] = allMatches[j], allMatches[i]
			}
		}
	}

	// Apply highlighting in reverse order to preserve indices
	result := text
	for _, m := range allMatches {
		highlighted := pathStyle.Render(m.text)
		result = result[:m.start] + highlighted + result[m.end:]
	}

	return result
}

// shortenPath shortens a file path for display
func (st *StreamTab) shortenPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) <= 3 {
		return path
	}
	// Show: .../parent/file.swift
	return ".../" + strings.Join(parts[len(parts)-2:], "/")
}

// emptyView renders the empty state
func (st *StreamTab) emptyView(styles Styles) string {
	icons := styles.Icons

	// Large icon (5x size effect) - use Idle icon (different from tab icon)
	iconStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextMuted).
		Bold(true).
		Padding(1, 0)
	bigIcon := iconStyle.Render(icons.Idle + "  " + icons.Idle + "  " + icons.Idle + "  " + icons.Idle + "  " + icons.Idle)

	msg := lipgloss.NewStyle().
		Foreground(styles.Colors.TextSubtle).
		Render("Waiting for build output...")

	hint := lipgloss.NewStyle().
		Foreground(styles.Colors.TextSubtle).
		Render("Press b to build, r to run, t to test")

	content := lipgloss.JoinVertical(lipgloss.Center, "", bigIcon, "", msg, "", hint)

	return lipgloss.Place(
		st.Width,
		st.Height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// =============================================================================
// Copy Support
// =============================================================================

// GetCurrentLine returns the currently selected/visible line for copying
func (st *StreamTab) GetCurrentLine() string {
	if len(st.Lines) == 0 {
		return ""
	}
	idx := st.ScrollPos
	if idx >= len(st.Lines) {
		idx = len(st.Lines) - 1
	}
	return st.Lines[idx].Raw
}

// GetVisibleContent returns all visible content for copying
func (st *StreamTab) GetVisibleContent() string {
	if len(st.Lines) == 0 {
		return ""
	}

	start := st.ScrollPos
	end := start + st.VisibleRows
	if end > len(st.Lines) {
		end = len(st.Lines)
	}

	var lines []string
	for i := start; i < end; i++ {
		lines = append(lines, st.Lines[i].Raw)
	}

	return strings.Join(lines, "\n")
}
