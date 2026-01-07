package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Log Group - Collapsible section of log output
// =============================================================================

// LogGroup represents a collapsible group of log lines
type LogGroup struct {
	ID        string
	Title     string
	Stage     string // Build stage: Compiling, Linking, etc.
	Lines     []LogLine
	Collapsed bool
	Status    LogGroupStatus
}

// LogGroupStatus represents the status of a log group
type LogGroupStatus int

const (
	LogGroupPending LogGroupStatus = iota
	LogGroupRunning
	LogGroupSuccess
	LogGroupError
	LogGroupWarning
)

// LogLine represents a single log entry
type LogLine struct {
	Text     string
	Type     LogLineType
	FilePath string // For error/warning: source file path
	Line     int    // For error/warning: line number
}

// LogLineType categorizes log lines
type LogLineType int

const (
	LogLineNormal LogLineType = iota
	LogLineInfo
	LogLineSuccess
	LogLineWarning
	LogLineError
)

// =============================================================================
// Log View - Manages grouped log display
// =============================================================================

// LogView manages grouped/collapsible log display
type LogView struct {
	Groups       []LogGroup
	FlatLines    []string // Raw flat lines for backwards compatibility
	SelectedIdx  int      // Selected group index for expansion
	AutoCollapse bool     // Auto-collapse completed groups
	Width        int
	Height       int
}

// NewLogView creates a new log view
func NewLogView() LogView {
	return LogView{
		Groups:       []LogGroup{},
		FlatLines:    []string{},
		AutoCollapse: true,
	}
}

// Clear resets the log view
func (v *LogView) Clear() {
	v.Groups = []LogGroup{}
	v.FlatLines = []string{}
	v.SelectedIdx = 0
}

// AddLine adds a raw line (backwards compatible)
func (v *LogView) AddLine(line string) {
	v.FlatLines = append(v.FlatLines, line)
	// Also try to add to current group or create new one
	v.processLine(line)
}

// processLine categorizes and adds line to appropriate group
func (v *LogView) processLine(line string) {
	// Detect stage changes to create new groups
	stage := detectStage(line)
	if stage != "" {
		// Check if we already have this stage
		found := false
		for i := range v.Groups {
			if v.Groups[i].Stage == stage {
				v.Groups[i].Status = LogGroupRunning
				found = true
				break
			}
		}
		if !found {
			// Create new group
			group := LogGroup{
				ID:        stage,
				Title:     stage,
				Stage:     stage,
				Lines:     []LogLine{},
				Collapsed: false,
				Status:    LogGroupRunning,
			}
			v.Groups = append(v.Groups, group)
		}
	}

	// Add line to current (last running) group
	if len(v.Groups) > 0 {
		logLine := categorizeLogLine(line)
		lastIdx := len(v.Groups) - 1
		for i := lastIdx; i >= 0; i-- {
			if v.Groups[i].Status == LogGroupRunning {
				v.Groups[i].Lines = append(v.Groups[i].Lines, logLine)

				// Update group status based on line type
				if logLine.Type == LogLineError {
					v.Groups[i].Status = LogGroupError
				} else if logLine.Type == LogLineWarning && v.Groups[i].Status != LogGroupError {
					v.Groups[i].Status = LogGroupWarning
				}
				break
			}
		}
	}
}

// MarkGroupComplete marks the current running group as complete
func (v *LogView) MarkGroupComplete(success bool) {
	for i := range v.Groups {
		if v.Groups[i].Status == LogGroupRunning {
			if success && v.Groups[i].Status != LogGroupError && v.Groups[i].Status != LogGroupWarning {
				v.Groups[i].Status = LogGroupSuccess
			}
			if v.AutoCollapse {
				v.Groups[i].Collapsed = true
			}
		}
	}
}

// ToggleGroup toggles the collapsed state of a group
func (v *LogView) ToggleGroup(idx int) {
	if idx >= 0 && idx < len(v.Groups) {
		v.Groups[idx].Collapsed = !v.Groups[idx].Collapsed
	}
}

// MoveUp moves selection up
func (v *LogView) MoveUp() {
	if v.SelectedIdx > 0 {
		v.SelectedIdx--
	}
}

// MoveDown moves selection down
func (v *LogView) MoveDown() {
	if v.SelectedIdx < len(v.Groups)-1 {
		v.SelectedIdx++
	}
}

// =============================================================================
// Rendering
// =============================================================================

// View renders the log view
func (v LogView) View(styles Styles) string {
	// If no groups, use flat view
	if len(v.Groups) == 0 {
		return v.flatView(styles)
	}

	return v.groupedView(styles)
}

// flatView renders logs as flat lines (backwards compatible)
func (v LogView) flatView(styles Styles) string {
	if len(v.FlatLines) == 0 {
		return ""
	}

	// Limit visible lines to height
	startIdx := 0
	if len(v.FlatLines) > v.Height {
		startIdx = len(v.FlatLines) - v.Height
	}

	var lines []string
	for i := startIdx; i < len(v.FlatLines); i++ {
		lines = append(lines, v.FlatLines[i])
	}

	return strings.Join(lines, "\n")
}

// groupedView renders logs with collapsible groups
func (v LogView) groupedView(styles Styles) string {
	var b strings.Builder
	icons := styles.Icons

	for i, group := range v.Groups {
		isSelected := i == v.SelectedIdx

		// Group header
		header := v.renderGroupHeader(group, isSelected, styles, icons)
		b.WriteString(header)
		b.WriteString("\n")

		// Group content (if expanded)
		if !group.Collapsed {
			for _, line := range group.Lines {
				lineStr := v.renderLogLine(line, styles, icons)
				b.WriteString("  " + lineStr)
				b.WriteString("\n")
			}
		}
	}

	return strings.TrimSuffix(b.String(), "\n")
}

// renderGroupHeader renders a group header with status and collapse indicator
func (v LogView) renderGroupHeader(group LogGroup, selected bool, styles Styles, icons Icons) string {
	// Collapse indicator
	collapseIcon := icons.ChevronDown
	if group.Collapsed {
		collapseIcon = icons.ChevronRight
	}

	// Status icon
	statusIcon := icons.Idle
	statusStyle := styles.StatusStyle("idle")
	switch group.Status {
	case LogGroupRunning:
		statusIcon = icons.Running
		statusStyle = styles.StatusStyle("running")
	case LogGroupSuccess:
		statusIcon = icons.Success
		statusStyle = styles.StatusStyle("success")
	case LogGroupError:
		statusIcon = icons.Error
		statusStyle = styles.StatusStyle("error")
	case LogGroupWarning:
		statusIcon = icons.Warning
		statusStyle = styles.StatusStyle("warning")
	}

	// Title style
	titleStyle := lipgloss.NewStyle().Foreground(styles.Colors.Text)
	if selected {
		titleStyle = titleStyle.Bold(true)
	}

	// Count
	countStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)
	count := countStyle.Render("(" + itoa(len(group.Lines)) + ")")

	return collapseIcon + " " + statusStyle.Render(statusIcon) + " " + titleStyle.Render(group.Title) + " " + count
}

// renderLogLine renders a single log line
func (v LogView) renderLogLine(line LogLine, styles Styles, icons Icons) string {
	var prefix string
	lineStyle := lipgloss.NewStyle().Foreground(styles.Colors.Text)

	switch line.Type {
	case LogLineError:
		prefix = styles.StatusStyle("error").Render(icons.Error) + " "
		lineStyle = lineStyle.Foreground(styles.Colors.Error)
	case LogLineWarning:
		prefix = styles.StatusStyle("warning").Render(icons.Warning) + " "
		lineStyle = lineStyle.Foreground(styles.Colors.Warning)
	case LogLineSuccess:
		prefix = styles.StatusStyle("success").Render(icons.Success) + " "
	case LogLineInfo:
		prefix = styles.StatusStyle("running").Render(icons.ChevronRight) + " "
	default:
		prefix = "  "
	}

	return prefix + lineStyle.Render(line.Text)
}

// =============================================================================
// Helpers
// =============================================================================

// detectStage detects build stage from log line
func detectStage(line string) string {
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "compiling"):
		return "Compiling"
	case strings.Contains(lower, "linking"):
		return "Linking"
	case strings.Contains(lower, "signing"):
		return "Signing"
	case strings.Contains(lower, "processing"):
		return "Processing"
	case strings.Contains(lower, "copying"):
		return "Copying"
	case strings.Contains(lower, "running"):
		return "Running"
	case strings.Contains(lower, "testing"):
		return "Testing"
	case strings.Contains(lower, "analyzing"):
		return "Analyzing"
	case strings.Contains(lower, "archiving"):
		return "Archiving"
	}
	return ""
}

// categorizeLogLine determines the type of a log line
func categorizeLogLine(line string) LogLine {
	lower := strings.ToLower(line)

	logLine := LogLine{Text: line, Type: LogLineNormal}

	// Detect error patterns
	if strings.Contains(lower, "error:") ||
		strings.Contains(lower, "❌") ||
		strings.Contains(lower, "failed") {
		logLine.Type = LogLineError
		// Try to extract file path and line number
		logLine.FilePath, logLine.Line = extractFileLocation(line)
	} else if strings.Contains(lower, "warning:") ||
		strings.Contains(lower, "⚠") {
		logLine.Type = LogLineWarning
		logLine.FilePath, logLine.Line = extractFileLocation(line)
	} else if strings.Contains(lower, "success") ||
		strings.Contains(lower, "✓") ||
		strings.Contains(lower, "passed") {
		logLine.Type = LogLineSuccess
	} else if strings.Contains(line, "▸") ||
		strings.Contains(line, "→") {
		logLine.Type = LogLineInfo
	}

	return logLine
}

// extractFileLocation tries to extract file:line from a log line
func extractFileLocation(line string) (string, int) {
	// Common patterns: /path/to/file.swift:42:10: error:
	// This is a simple implementation - could be made more robust
	parts := strings.Split(line, ":")
	if len(parts) >= 2 {
		// Try to find a path-like part followed by a number
		for i := 0; i < len(parts)-1; i++ {
			if strings.Contains(parts[i], "/") || strings.Contains(parts[i], ".swift") || strings.Contains(parts[i], ".m") {
				var lineNum int
				if i+1 < len(parts) {
					_, _ = fmt.Sscanf(parts[i+1], "%d", &lineNum)
				}
				return strings.TrimSpace(parts[i]), lineNum
			}
		}
	}
	return "", 0
}
