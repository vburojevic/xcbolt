package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Issue - Represents a build error or warning
// =============================================================================

// Issue represents a collected error or warning
type Issue struct {
	Type     IssueType
	Message  string
	FilePath string
	Line     int
	Column   int
	Context  string // Surrounding code context
	Expanded bool   // Whether the issue is expanded to show details
}

// IssueType categorizes issues
type IssueType int

const (
	IssueError IssueType = iota
	IssueWarning
)

// =============================================================================
// Issues Panel - Collects and displays all errors/warnings
// =============================================================================

// IssuesPanel manages collected issues
type IssuesPanel struct {
	Issues      []Issue
	SelectedIdx int
	Collapsed   bool // Whether the whole panel is collapsed
	Width       int
	Height      int
	MaxVisible  int // Max issues to show at once
}

// NewIssuesPanel creates a new issues panel
func NewIssuesPanel() IssuesPanel {
	return IssuesPanel{
		Issues:     []Issue{},
		MaxVisible: 5,
	}
}

// Clear removes all issues
func (p *IssuesPanel) Clear() {
	p.Issues = []Issue{}
	p.SelectedIdx = 0
}

// AddIssue adds a new issue
func (p *IssuesPanel) AddIssue(issue Issue) {
	p.Issues = append(p.Issues, issue)
}

// AddFromLogLine parses a log line and adds issue if it's an error/warning
func (p *IssuesPanel) AddFromLogLine(line string) {
	lower := strings.ToLower(line)

	var issueType IssueType = -1

	if strings.Contains(lower, "error:") {
		issueType = IssueError
	} else if strings.Contains(lower, "warning:") {
		issueType = IssueWarning
	}

	if issueType < 0 {
		return
	}

	// Parse the line for file location
	filePath, lineNum, column := parseIssueLocation(line)

	// Extract just the message part
	message := extractIssueMessage(line)

	issue := Issue{
		Type:     issueType,
		Message:  message,
		FilePath: filePath,
		Line:     lineNum,
		Column:   column,
	}

	p.Issues = append(p.Issues, issue)
}

// ErrorCount returns the number of errors
func (p IssuesPanel) ErrorCount() int {
	count := 0
	for _, issue := range p.Issues {
		if issue.Type == IssueError {
			count++
		}
	}
	return count
}

// WarningCount returns the number of warnings
func (p IssuesPanel) WarningCount() int {
	count := 0
	for _, issue := range p.Issues {
		if issue.Type == IssueWarning {
			count++
		}
	}
	return count
}

// HasIssues returns true if there are any issues
func (p IssuesPanel) HasIssues() bool {
	return len(p.Issues) > 0
}

// MoveUp moves selection up
func (p *IssuesPanel) MoveUp() {
	if p.SelectedIdx > 0 {
		p.SelectedIdx--
	}
}

// MoveDown moves selection down
func (p *IssuesPanel) MoveDown() {
	if p.SelectedIdx < len(p.Issues)-1 {
		p.SelectedIdx++
	}
}

// ToggleExpand toggles the expanded state of the selected issue
func (p *IssuesPanel) ToggleExpand() {
	if p.SelectedIdx >= 0 && p.SelectedIdx < len(p.Issues) {
		p.Issues[p.SelectedIdx].Expanded = !p.Issues[p.SelectedIdx].Expanded
	}
}

// ToggleCollapse toggles the collapsed state of the whole panel
func (p *IssuesPanel) ToggleCollapse() {
	p.Collapsed = !p.Collapsed
}

// =============================================================================
// Rendering
// =============================================================================

// View renders the issues panel
func (p IssuesPanel) View(styles Styles) string {
	if len(p.Issues) == 0 {
		return ""
	}

	var b strings.Builder
	icons := styles.Icons

	// Header
	header := p.renderHeader(styles, icons)
	b.WriteString(header)
	b.WriteString("\n")

	// If collapsed, don't show issues
	if p.Collapsed {
		return b.String()
	}

	// Separator
	sepStyle := lipgloss.NewStyle().Foreground(styles.Colors.BorderMuted)
	b.WriteString(sepStyle.Render(strings.Repeat("─", p.Width-4)))
	b.WriteString("\n")

	// Issues
	visibleCount := minInt(len(p.Issues), p.MaxVisible)
	for i := 0; i < visibleCount; i++ {
		issue := p.Issues[i]
		isSelected := i == p.SelectedIdx

		line := p.renderIssue(issue, isSelected, styles, icons)
		b.WriteString(line)
		b.WriteString("\n")

		// Show expanded details
		if issue.Expanded && isSelected {
			details := p.renderIssueDetails(issue, styles)
			b.WriteString(details)
			b.WriteString("\n")
		}
	}

	// Show "and X more" if there are more issues
	if len(p.Issues) > p.MaxVisible {
		moreStyle := lipgloss.NewStyle().
			Foreground(styles.Colors.TextMuted).
			Italic(true)
		more := moreStyle.Render(fmt.Sprintf("  ... and %d more", len(p.Issues)-p.MaxVisible))
		b.WriteString(more)
		b.WriteString("\n")
	}

	return strings.TrimSuffix(b.String(), "\n")
}

// renderHeader renders the issues panel header
func (p IssuesPanel) renderHeader(styles Styles, icons Icons) string {
	// Collapse indicator
	collapseIcon := icons.ChevronDown
	if p.Collapsed {
		collapseIcon = icons.ChevronRight
	}

	// Title with counts
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Colors.Text)

	countStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextMuted)

	title := titleStyle.Render("ISSUES")

	errorCount := p.ErrorCount()
	warningCount := p.WarningCount()

	counts := ""
	if errorCount > 0 {
		errStyle := styles.StatusStyle("error")
		counts += errStyle.Render(fmt.Sprintf("%d errors", errorCount))
	}
	if warningCount > 0 {
		if counts != "" {
			counts += countStyle.Render(", ")
		}
		warnStyle := styles.StatusStyle("warning")
		counts += warnStyle.Render(fmt.Sprintf("%d warnings", warningCount))
	}

	return collapseIcon + " " + title + " " + countStyle.Render("(") + counts + countStyle.Render(")")
}

// renderIssue renders a single issue line
func (p IssuesPanel) renderIssue(issue Issue, selected bool, styles Styles, icons Icons) string {
	// Type icon
	icon := icons.Warning
	iconStyle := styles.StatusStyle("warning")
	if issue.Type == IssueError {
		icon = icons.Error
		iconStyle = styles.StatusStyle("error")
	}

	// Selection indicator
	selectIndicator := "  "
	if selected {
		selectIndicator = icons.ChevronRight + " "
	}

	// File location
	locStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Accent)
	location := ""
	if issue.FilePath != "" {
		// Shorten path to last 2 segments
		location = shortenPath(issue.FilePath)
		if issue.Line > 0 {
			location += fmt.Sprintf(":%d", issue.Line)
		}
		location = locStyle.Render(location) + " "
	}

	// Message
	msgStyle := lipgloss.NewStyle().Foreground(styles.Colors.Text)
	if selected {
		msgStyle = msgStyle.Bold(true)
	}
	message := truncateString(issue.Message, p.Width-30)

	return selectIndicator + iconStyle.Render(icon) + " " + location + msgStyle.Render(message)
}

// renderIssueDetails renders expanded issue details
func (p IssuesPanel) renderIssueDetails(issue Issue, styles Styles) string {
	detailStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextMuted).
		PaddingLeft(4)

	var lines []string

	// Full file path
	if issue.FilePath != "" {
		lines = append(lines, detailStyle.Render("File: "+issue.FilePath))
	}

	// Full message if truncated
	if len(issue.Message) > p.Width-30 {
		lines = append(lines, detailStyle.Render(issue.Message))
	}

	// Context if available
	if issue.Context != "" {
		lines = append(lines, detailStyle.Render(issue.Context))
	}

	return strings.Join(lines, "\n")
}

// =============================================================================
// Helpers
// =============================================================================

// parseIssueLocation extracts file:line:column from an error/warning line
func parseIssueLocation(line string) (string, int, int) {
	// Common pattern: /path/to/file.swift:42:10: error: message
	parts := strings.Split(line, ":")

	if len(parts) < 2 {
		return "", 0, 0
	}

	// Find path-like segment
	for i := 0; i < len(parts); i++ {
		part := strings.TrimSpace(parts[i])
		if strings.Contains(part, "/") || strings.HasSuffix(part, ".swift") ||
			strings.HasSuffix(part, ".m") || strings.HasSuffix(part, ".h") {

			filePath := part
			lineNum := 0
			column := 0

			// Try to get line number
			if i+1 < len(parts) {
				fmt.Sscanf(strings.TrimSpace(parts[i+1]), "%d", &lineNum)
			}
			// Try to get column
			if i+2 < len(parts) {
				fmt.Sscanf(strings.TrimSpace(parts[i+2]), "%d", &column)
			}

			return filePath, lineNum, column
		}
	}

	return "", 0, 0
}

// extractIssueMessage extracts the message part from an error/warning line
func extractIssueMessage(line string) string {
	// Try to find "error:" or "warning:" and take everything after
	lower := strings.ToLower(line)

	if idx := strings.Index(lower, "error:"); idx != -1 {
		msg := strings.TrimSpace(line[idx+6:])
		// Remove leading "error: " if it appears again
		msg = strings.TrimPrefix(strings.TrimPrefix(msg, "error:"), "Error:")
		return strings.TrimSpace(msg)
	}

	if idx := strings.Index(lower, "warning:"); idx != -1 {
		msg := strings.TrimSpace(line[idx+8:])
		msg = strings.TrimPrefix(strings.TrimPrefix(msg, "warning:"), "Warning:")
		return strings.TrimSpace(msg)
	}

	return line
}

// shortenPath shortens a file path to the last 2 segments
func shortenPath(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) <= 2 {
		return path
	}
	return ".../" + strings.Join(parts[len(parts)-2:], "/")
}

// truncateString truncates a string to maxLen characters with ellipsis
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	if maxLen <= 3 {
		return s[:maxLen]
	}
	return s[:maxLen-3] + "..."
}
