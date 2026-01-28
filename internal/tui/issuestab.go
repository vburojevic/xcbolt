package tui

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Issue Types
// =============================================================================

// IssueType categorizes the severity of an issue
type IssueType int

const (
	IssueTypeError IssueType = iota
	IssueTypeWarning
	IssueTypeNote
)

// String returns the display name for an issue type
func (t IssueType) String() string {
	switch t {
	case IssueTypeError:
		return "error"
	case IssueTypeWarning:
		return "warning"
	case IssueTypeNote:
		return "note"
	default:
		return "unknown"
	}
}

// =============================================================================
// Issue
// =============================================================================

// Issue represents a build error, warning, or note
type Issue struct {
	Type     IssueType
	Message  string
	File     string
	Line     int
	Column   int
	FullText string // Complete multi-line message
	Expanded bool   // Whether to show full message
}

// =============================================================================
// IssuesTab
// =============================================================================

// IssuesTab displays issues sorted by severity
type IssuesTab struct {
	Issues       []Issue
	Selected     int // Currently selected issue
	Running      bool
	SpinnerFrame int

	// Scroll state
	ScrollPos   int
	VisibleRows int

	// Dimensions
	Width  int
	Height int

	// Regex for parsing error locations
	locationRegex *regexp.Regexp
}

// NewIssuesTab creates a new IssuesTab
func NewIssuesTab() *IssuesTab {
	return &IssuesTab{
		Issues:        make([]Issue, 0, 100),
		locationRegex: regexp.MustCompile(`([^\s:]+):(\d+):(\d+):`),
	}
}

// SetRunning updates the running state for empty view hints.
func (it *IssuesTab) SetRunning(running bool) {
	it.Running = running
}

// AdvanceSpinner advances the idle spinner frame.
func (it *IssuesTab) AdvanceSpinner() {
	it.SpinnerFrame = (it.SpinnerFrame + 1) % len(spinnerFrames)
}

// SetSize updates dimensions
func (it *IssuesTab) SetSize(width, height int) {
	it.Width = width
	it.Height = height
	// Reserve lines for header and analysis section
	it.VisibleRows = height - 4
	if it.VisibleRows < 1 {
		it.VisibleRows = 1
	}
}

// Clear resets the issues list
func (it *IssuesTab) Clear() {
	it.Issues = it.Issues[:0]
	it.Selected = 0
	it.ScrollPos = 0
}

// AddIssue adds a new issue from a log line
func (it *IssuesTab) AddIssue(issueType IssueType, line string) {
	issue := it.parseIssue(issueType, line)
	it.Issues = append(it.Issues, issue)
	it.sortIssues()
	if maxIssues > 0 && len(it.Issues) > maxIssues {
		it.Issues = it.Issues[:maxIssues]
		if it.Selected >= len(it.Issues) {
			it.Selected = len(it.Issues) - 1
			if it.Selected < 0 {
				it.Selected = 0
			}
		}
		if it.ScrollPos > it.Selected {
			it.ScrollPos = it.Selected
		}
	}
}

// parseIssue extracts issue details from a log line
func (it *IssuesTab) parseIssue(issueType IssueType, line string) Issue {
	issue := Issue{
		Type:     issueType,
		Message:  line,
		FullText: line,
	}

	// Try to extract file:line:column
	matches := it.locationRegex.FindStringSubmatch(line)
	if len(matches) >= 4 {
		issue.File = matches[1]
		fmt.Sscanf(matches[2], "%d", &issue.Line)
		fmt.Sscanf(matches[3], "%d", &issue.Column)

		// Extract just the message part (after the location)
		idx := strings.Index(line, matches[0])
		if idx >= 0 {
			rest := line[idx+len(matches[0]):]
			// Remove "error:" or "warning:" prefix
			rest = strings.TrimPrefix(rest, " error: ")
			rest = strings.TrimPrefix(rest, " warning: ")
			rest = strings.TrimPrefix(rest, " note: ")
			rest = strings.TrimSpace(rest)
			if rest != "" {
				issue.Message = rest
			}
		}
	}

	return issue
}

// sortIssues sorts issues by severity (errors first, then warnings, then notes)
func (it *IssuesTab) sortIssues() {
	sort.SliceStable(it.Issues, func(i, j int) bool {
		return it.Issues[i].Type < it.Issues[j].Type
	})
}

// =============================================================================
// Scrolling and Selection
// =============================================================================

func (it *IssuesTab) maxScrollPos() int {
	max := len(it.Issues) - it.VisibleRows
	if max < 0 {
		return 0
	}
	return max
}

// ScrollUp scrolls up by n lines
func (it *IssuesTab) ScrollUp(n int) {
	it.Selected -= n
	if it.Selected < 0 {
		it.Selected = 0
	}
	// Adjust scroll to keep selection visible
	if it.Selected < it.ScrollPos {
		it.ScrollPos = it.Selected
	}
}

// ScrollDown scrolls down by n lines
func (it *IssuesTab) ScrollDown(n int) {
	it.Selected += n
	if it.Selected >= len(it.Issues) {
		it.Selected = len(it.Issues) - 1
	}
	if it.Selected < 0 {
		it.Selected = 0
	}
	// Adjust scroll to keep selection visible
	if it.Selected >= it.ScrollPos+it.VisibleRows {
		it.ScrollPos = it.Selected - it.VisibleRows + 1
	}
}

// GotoTop goes to the first issue
func (it *IssuesTab) GotoTop() {
	it.Selected = 0
	it.ScrollPos = 0
}

// GotoBottom goes to the last issue
func (it *IssuesTab) GotoBottom() {
	if len(it.Issues) > 0 {
		it.Selected = len(it.Issues) - 1
		it.ScrollPos = it.maxScrollPos()
	}
}

// ToggleExpand toggles the expanded state of the selected issue
func (it *IssuesTab) ToggleExpand() {
	if it.Selected >= 0 && it.Selected < len(it.Issues) {
		it.Issues[it.Selected].Expanded = !it.Issues[it.Selected].Expanded
	}
}

// =============================================================================
// View Rendering
// =============================================================================

// View renders the issues tab content
func (it *IssuesTab) View(styles Styles) string {
	if len(it.Issues) == 0 {
		return it.emptyView(styles)
	}

	barWidth := scrollbarWidth
	if it.Width-barWidth < 1 {
		barWidth = 0
	}
	contentWidth := it.Width - barWidth
	if contentWidth < 1 {
		contentWidth = it.Width
	}

	emptyBar := strings.Repeat(" ", barWidth)
	pad := lipgloss.NewStyle().Width(contentWidth)

	var lines []string

	// Header with counts
	header := it.renderHeader(styles)
	if header != "" {
		for _, line := range strings.Split(header, "\n") {
			lines = append(lines, pad.Render(line)+emptyBar)
		}
	}

	// Issue list with scrollbar
	listLines := it.renderIssueListLines(styles, contentWidth)
	barLines := renderScrollbarLines(it.VisibleRows, len(it.Issues), it.ScrollPos, styles)
	if len(barLines) != it.VisibleRows {
		barLines = make([]string, it.VisibleRows)
		for i := range barLines {
			barLines[i] = emptyBar
		}
	}
	for i, line := range listLines {
		lines = append(lines, pad.Render(line)+barLines[i])
	}

	// Analysis section (if there are errors)
	errorCount := it.countByType(IssueTypeError)
	if errorCount > 0 {
		analysis := it.renderAnalysis(styles)
		for _, line := range strings.Split(analysis, "\n") {
			lines = append(lines, pad.Render(line)+emptyBar)
		}
	}

	return strings.Join(lines, "\n")
}

// renderHeader renders the issue count header
func (it *IssuesTab) renderHeader(styles Styles) string {
	errorCount := it.countByType(IssueTypeError)
	warnCount := it.countByType(IssueTypeWarning)

	icons := styles.Icons

	var parts []string

	if errorCount > 0 {
		errorStyle := lipgloss.NewStyle().Foreground(styles.Colors.Error)
		parts = append(parts, errorStyle.Render(fmt.Sprintf("%s %d errors", icons.Error, errorCount)))
	}

	if warnCount > 0 {
		warnStyle := lipgloss.NewStyle().Foreground(styles.Colors.Warning)
		parts = append(parts, warnStyle.Render(fmt.Sprintf("%s %d warnings", icons.Warning, warnCount)))
	}

	if len(parts) == 0 {
		return ""
	}

	header := strings.Join(parts, "  ")
	return lipgloss.NewStyle().
		Foreground(styles.Colors.Text).
		Padding(0, 1).
		Render(header)
}

// renderIssueList renders the list of issues
func (it *IssuesTab) renderIssueListLines(styles Styles, maxWidth int) []string {
	if len(it.Issues) == 0 {
		return padLines(nil, it.VisibleRows)
	}

	// Calculate visible range
	start := it.ScrollPos
	end := start + it.VisibleRows
	if end > len(it.Issues) {
		end = len(it.Issues)
	}

	var lines []string
	for i := start; i < end; i++ {
		issue := it.Issues[i]
		isSelected := i == it.Selected
		line := it.renderIssue(issue, isSelected, styles, maxWidth)
		lines = append(lines, line)
	}

	return padLines(lines, it.VisibleRows)
}

// renderIssue renders a single issue line
func (it *IssuesTab) renderIssue(issue Issue, selected bool, styles Styles, maxWidth int) string {
	icons := styles.Icons
	lineWidth := maxWidth - 4
	if lineWidth < 10 {
		lineWidth = 10
	}

	// Icon based on type
	var icon string
	var iconStyle lipgloss.Style
	switch issue.Type {
	case IssueTypeError:
		icon = icons.Error
		iconStyle = lipgloss.NewStyle().Foreground(styles.Colors.Error)
	case IssueTypeWarning:
		icon = icons.Warning
		iconStyle = lipgloss.NewStyle().Foreground(styles.Colors.Warning)
	case IssueTypeNote:
		icon = icons.Note
		iconStyle = lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)
	}

	// Message (truncate if needed)
	message := issue.Message
	if !issue.Expanded {
		maxLen := lineWidth - 20
		if maxLen < 10 {
			maxLen = lineWidth
		}
		if maxLen > 3 && len(message) > maxLen {
			cut := maxLen - 3
			message = message[:cut] + "..."
		}
	}

	// File location (shortened)
	var location string
	if issue.File != "" {
		file := issue.File
		// Shorten path
		parts := strings.Split(file, "/")
		if len(parts) > 2 {
			file = ".../" + strings.Join(parts[len(parts)-2:], "/")
		}
		if issue.Line > 0 {
			location = fmt.Sprintf("%s:%d", file, issue.Line)
		} else {
			location = file
		}
	}

	// Build the line
	iconRendered := iconStyle.Render(icon)

	messageStyle := lipgloss.NewStyle().Foreground(styles.Colors.Text)
	if selected {
		messageStyle = messageStyle.Bold(true).Foreground(styles.Colors.Accent)
	}
	messageRendered := messageStyle.Render(message)

	locationStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextSubtle)
	locationRendered := ""
	if location != "" {
		locationRendered = "  " + locationStyle.Render(location)
	}

	// Selection indicator
	prefix := "  "
	if selected {
		prefix = lipgloss.NewStyle().Foreground(styles.Colors.Accent).Render("> ")
	}

	line := prefix + iconRendered + " " + messageRendered + locationRendered

	// If expanded, show full text on next lines
	if issue.Expanded && issue.FullText != issue.Message {
		fullStyle := lipgloss.NewStyle().
			Foreground(styles.Colors.TextMuted).
			PaddingLeft(4)
		line += "\n" + fullStyle.Render(issue.FullText)
	}

	return line
}

// renderAnalysis renders the AI-style analysis section
func (it *IssuesTab) renderAnalysis(styles Styles) string {
	errors := it.getByType(IssueTypeError)
	if len(errors) == 0 {
		return ""
	}

	// Generate analysis based on error patterns
	analysis := it.generateAnalysis(errors)
	if analysis == "" {
		return ""
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextMuted).
		Bold(true)

	contentStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Text).
		PaddingLeft(2)

	divider := lipgloss.NewStyle().
		Foreground(styles.Colors.Border).
		Render(strings.Repeat("─", it.Width-4))

	return lipgloss.JoinVertical(lipgloss.Left,
		"",
		divider,
		headerStyle.Render("Analysis"),
		contentStyle.Render(analysis),
	)
}

// generateAnalysis creates heuristic-based analysis for common errors
func (it *IssuesTab) generateAnalysis(errors []Issue) string {
	if len(errors) == 0 {
		return ""
	}

	var suggestions []string

	for _, err := range errors {
		msg := strings.ToLower(err.Message)

		// Module not found
		if strings.Contains(msg, "no such module") {
			suggestions = append(suggestions, "Missing module - try 'pod install' or resolve SPM packages")
		}

		// Type mismatch
		if strings.Contains(msg, "cannot convert") || strings.Contains(msg, "type mismatch") {
			suggestions = append(suggestions, "Type conversion issue - check function signatures and return types")
		}

		// Missing member
		if strings.Contains(msg, "has no member") {
			suggestions = append(suggestions, "Missing member - check spelling or import statements")
		}

		// Unresolved identifier
		if strings.Contains(msg, "cannot find") || strings.Contains(msg, "unresolved identifier") {
			suggestions = append(suggestions, "Unresolved symbol - check imports and variable declarations")
		}

		// Concurrency issues
		if strings.Contains(msg, "sendable") || strings.Contains(msg, "@mainactor") {
			suggestions = append(suggestions, "Swift concurrency issue - review actor isolation and Sendable conformance")
		}
	}

	// Deduplicate
	seen := make(map[string]bool)
	var unique []string
	for _, s := range suggestions {
		if !seen[s] {
			seen[s] = true
			unique = append(unique, s)
		}
	}

	if len(unique) == 0 {
		return fmt.Sprintf("Found %d error(s). Review the messages above for details.", len(errors))
	}

	return strings.Join(unique, "\n")
}

// =============================================================================
// Helper Methods
// =============================================================================

// countByType counts issues of a specific type
func (it *IssuesTab) countByType(issueType IssueType) int {
	count := 0
	for _, issue := range it.Issues {
		if issue.Type == issueType {
			count++
		}
	}
	return count
}

// getByType returns issues of a specific type
func (it *IssuesTab) getByType(issueType IssueType) []Issue {
	var result []Issue
	for _, issue := range it.Issues {
		if issue.Type == issueType {
			result = append(result, issue)
		}
	}
	return result
}

// emptyView renders the empty state
func (it *IssuesTab) emptyView(styles Styles) string {
	icons := styles.Icons

	if it.Running {
		spinner := spinnerFrames[it.SpinnerFrame]
		spinStyle := lipgloss.NewStyle().Foreground(styles.Colors.Accent).Bold(true)
		msg := lipgloss.NewStyle().
			Foreground(styles.Colors.TextSubtle).
			Render("Scanning for issues...")
		hint := lipgloss.NewStyle().
			Foreground(styles.Colors.TextMuted).
			Render("Build in progress")

		content := lipgloss.JoinVertical(lipgloss.Center, spinStyle.Render(spinner), "", msg, hint)
		return lipgloss.Place(
			it.Width,
			it.Height,
			lipgloss.Center,
			lipgloss.Center,
			content,
		)
	}

	// Large icon
	iconStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Success).
		Bold(true)
	bigIcon := iconStyle.Render(icons.Success)

	msg := lipgloss.NewStyle().
		Foreground(styles.Colors.TextSubtle).
		Render("No issues found!")

	hint := lipgloss.NewStyle().
		Foreground(styles.Colors.TextSubtle).
		Render("Build completed without errors or warnings")

	content := lipgloss.JoinVertical(lipgloss.Center, bigIcon, "", msg, hint)

	return lipgloss.Place(
		it.Width,
		it.Height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// GetSelectedIssue returns the currently selected issue
func (it *IssuesTab) GetSelectedIssue() *Issue {
	if it.Selected >= 0 && it.Selected < len(it.Issues) {
		return &it.Issues[it.Selected]
	}
	return nil
}
