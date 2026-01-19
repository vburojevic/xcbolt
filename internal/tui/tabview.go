package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Tab Types
// =============================================================================

// Tab represents which tab is active
type Tab int

const (
	TabDashboard Tab = iota
	TabStream
	TabIssues
)

// String returns the display name for a tab
func (t Tab) String() string {
	switch t {
	case TabDashboard:
		return "Dashboard"
	case TabStream:
		return "Logs"
	case TabIssues:
		return "Issues"
	default:
		return "Unknown"
	}
}

// =============================================================================
// TabCounts - Live counts for tab badges
// =============================================================================

// TabCounts holds the badge counts for each tab
type TabCounts struct {
	StreamLines  int // Total lines in logs
	ErrorCount   int // Number of errors
	WarningCount int // Number of warnings
}

// IssueTotal returns total issues (errors + warnings)
func (c TabCounts) IssueTotal() int {
	return c.ErrorCount + c.WarningCount
}

// =============================================================================
// TabView - Main container for all tabs
// =============================================================================

// TabView manages the 3-tab log display system
type TabView struct {
	ActiveTab Tab
	Counts    TabCounts

	// Individual tab components
	StreamTab  *StreamTab
	IssuesTab  *IssuesTab
	SummaryTab *SummaryTab

	// Dimensions
	Width  int
	Height int

	// Settings
	ShowLineNumbers bool
	ShowTimestamps  bool
}

// NewTabView creates a new TabView with all tabs initialized
func NewTabView() *TabView {
	return &TabView{
		ActiveTab:       TabDashboard,
		StreamTab:       NewStreamTab(),
		IssuesTab:       NewIssuesTab(),
		SummaryTab:      NewSummaryTab(),
		ShowLineNumbers: true,
		ShowTimestamps:  false,
	}
}

// SetSize updates dimensions for all tabs
func (tv *TabView) SetSize(width, height int) {
	tv.Width = width
	// Subtract 3 lines for tab bar (2 content lines + 1 border line from Container style)
	contentHeight := height - 3
	if contentHeight < 0 {
		contentHeight = 0
	}
	tv.Height = contentHeight

	tv.StreamTab.SetSize(width, contentHeight)
	tv.IssuesTab.SetSize(width, contentHeight)
	tv.SummaryTab.SetSize(width, contentHeight)
}

// Clear resets all tabs for a new build
func (tv *TabView) Clear() {
	tv.StreamTab.Clear()
	tv.IssuesTab.Clear()
	tv.SummaryTab.Clear()
	tv.Counts = TabCounts{}
}

// SetActiveTab changes the active tab
func (tv *TabView) SetActiveTab(tab Tab) {
	tv.ActiveTab = tab
}

// NextTab cycles to the next tab
func (tv *TabView) NextTab() {
	tv.ActiveTab = (tv.ActiveTab + 1) % 3
}

// PrevTab cycles to the previous tab
func (tv *TabView) PrevTab() {
	tv.ActiveTab = (tv.ActiveTab + 2) % 3
}

// =============================================================================
// Event Routing
// =============================================================================

// AddLine routes a log line to appropriate tabs
func (tv *TabView) AddLine(line string, lineType TabLineType) {
	tv.StreamTab.AddLine(line, lineType)
	tv.Counts.StreamLines++

	// Route to issues tab if it's an error/warning (notes stay in stream only)
	switch lineType {
	case TabLineTypeError:
		tv.IssuesTab.AddIssue(IssueTypeError, line)
		tv.Counts.ErrorCount++
	case TabLineTypeWarning:
		tv.IssuesTab.AddIssue(IssueTypeWarning, line)
		tv.Counts.WarningCount++
	}
}

// AddRawLine adds a raw line to the stream tab
func (tv *TabView) AddRawLine(line string) {
	lineType := classifyTabLogLine(line)
	tv.AddLine(line, lineType)
}

// SetBuildResult updates the summary tab with build results
func (tv *TabView) SetBuildResult(status BuildStatus, duration string, phases []PhaseResult) {
	tv.SummaryTab.SetResult(status, duration, phases, tv.Counts.ErrorCount, tv.Counts.WarningCount)
}

// =============================================================================
// Scrolling
// =============================================================================

// ScrollUp scrolls the active tab up
func (tv *TabView) ScrollUp(n int) {
	switch tv.ActiveTab {
	case TabStream:
		tv.StreamTab.ScrollUp(n)
	case TabIssues:
		tv.IssuesTab.ScrollUp(n)
	case TabDashboard:
		tv.SummaryTab.ScrollUp(n)
	}
}

// ScrollDown scrolls the active tab down
func (tv *TabView) ScrollDown(n int) {
	switch tv.ActiveTab {
	case TabStream:
		tv.StreamTab.ScrollDown(n)
	case TabIssues:
		tv.IssuesTab.ScrollDown(n)
	case TabDashboard:
		tv.SummaryTab.ScrollDown(n)
	}
}

// GotoTop scrolls the active tab to the top
func (tv *TabView) GotoTop() {
	switch tv.ActiveTab {
	case TabStream:
		tv.StreamTab.GotoTop()
	case TabIssues:
		tv.IssuesTab.GotoTop()
	case TabDashboard:
		tv.SummaryTab.GotoTop()
	}
}

// GotoBottom scrolls the active tab to the bottom
func (tv *TabView) GotoBottom() {
	switch tv.ActiveTab {
	case TabStream:
		tv.StreamTab.GotoBottom()
	case TabIssues:
		tv.IssuesTab.GotoBottom()
	case TabDashboard:
		tv.SummaryTab.GotoBottom()
	}
}

// =============================================================================
// View Rendering
// =============================================================================

// View renders the complete tab view (tab bar + content)
func (tv *TabView) View(styles Styles) string {
	tabBar := tv.renderTabBar(styles)
	content := tv.renderContent(styles)

	return lipgloss.JoinVertical(lipgloss.Left, tabBar, content)
}

// renderTabBar renders the single-line tab bar
func (tv *TabView) renderTabBar(styles Styles) string {
	s := styles.TabBar
	icons := styles.Icons

	// Build each tab (order: Dashboard, Logs, Issues)
	tabs := []struct {
		tab   Tab
		icon  string
		label string
		badge string
	}{
		{
			tab:   TabDashboard,
			icon:  icons.TabSummary,
			label: "Dashboard",
			badge: "",
		},
		{
			tab:   TabStream,
			icon:  icons.TabStream,
			label: "Logs",
			badge: "",
		},
		{
			tab:   TabIssues,
			icon:  icons.TabIssues,
			label: "Issues",
			badge: tv.issuesBadge(),
		},
	}

	// Calculate tab width - distribute evenly
	tabWidth := tv.Width / 3
	if tabWidth < 15 {
		tabWidth = 15
	}
	// Last tab gets remaining width to fill exactly
	lastTabWidth := tv.Width - (tabWidth * 2)

	var lineParts []string
	var underlineParts []string

	for i, t := range tabs {
		isActive := tv.ActiveTab == t.tab
		cellWidth := tabWidth
		if i == 2 {
			cellWidth = lastTabWidth
		}

		// Alignment: Dashboard left, Logs center, Issues right
		var align lipgloss.Position
		switch i {
		case 0:
			align = lipgloss.Left
		case 1:
			align = lipgloss.Center
		case 2:
			align = lipgloss.Right
		}

		// Build content: icon  label (badge) - with proper spacing
		var iconStr, labelStr string
		if isActive {
			iconStr = lipgloss.NewStyle().Foreground(styles.Colors.Accent).Render(t.icon)
			labelStr = lipgloss.NewStyle().Foreground(styles.Colors.Accent).Bold(true).Render(t.label)
		} else {
			iconStr = lipgloss.NewStyle().Foreground(styles.Colors.TextSubtle).Render(t.icon)
			labelStr = lipgloss.NewStyle().Foreground(styles.Colors.TextMuted).Render(t.label)
		}

		// Add space between icon and text
		lineContent := iconStr + "  " + labelStr
		if t.badge != "" {
			badgeStyle := s.TabBadge
			if tv.Counts.ErrorCount > 0 && t.tab == TabIssues {
				badgeStyle = s.TabBadgeError
			}
			lineContent += " " + badgeStyle.Render(t.badge)
		}

		// Base cell style with alignment
		cellBase := lipgloss.NewStyle().
			Width(cellWidth).
			MaxWidth(cellWidth).
			MaxHeight(1).
			Align(align).
			PaddingLeft(1).
			PaddingRight(1)

		// Render cell
		lineParts = append(lineParts, cellBase.Render(lineContent))

		// Build underline for this cell
		underlineStyle := lipgloss.NewStyle().
			Width(cellWidth).
			MaxWidth(cellWidth).
			Align(align).
			PaddingLeft(1).
			PaddingRight(1)

		if isActive {
			// Calculate underline width based on label length
			underlineLen := len(t.icon) + 2 + len(t.label) // icon + spacing + label
			if t.badge != "" {
				underlineLen += 1 + len(t.badge)
			}
			underline := lipgloss.NewStyle().
				Foreground(styles.Colors.Accent).
				Render(strings.Repeat("─", underlineLen))
			underlineParts = append(underlineParts, underlineStyle.Render(underline))
		} else {
			underlineParts = append(underlineParts, underlineStyle.Render(""))
		}
	}

	line := lipgloss.JoinHorizontal(lipgloss.Top, lineParts...)
	underline := lipgloss.JoinHorizontal(lipgloss.Top, underlineParts...)

	// Ensure lines fill full width
	lineContainer := lipgloss.NewStyle().
		Width(tv.Width).
		MaxWidth(tv.Width).
		MaxHeight(1).
		Render(line)

	underlineContainer := lipgloss.NewStyle().
		Width(tv.Width).
		MaxWidth(tv.Width).
		MaxHeight(1).
		Render(underline)

	return lipgloss.JoinVertical(lipgloss.Left, lineContainer, underlineContainer)
}

// issuesSubtitle returns the subtitle for the Issues tab
func (tv *TabView) issuesSubtitle() string {
	if tv.Counts.ErrorCount > 0 {
		return fmt.Sprintf("%d errors", tv.Counts.ErrorCount)
	}
	if tv.Counts.WarningCount > 0 {
		return fmt.Sprintf("%d warnings", tv.Counts.WarningCount)
	}
	return "No issues"
}

// issuesBadge returns the badge text for the Issues tab
func (tv *TabView) issuesBadge() string {
	total := tv.Counts.IssueTotal()
	if total == 0 {
		return ""
	}
	return fmt.Sprintf("(%d)", total)
}

// renderContent renders the content of the active tab
func (tv *TabView) renderContent(styles Styles) string {
	switch tv.ActiveTab {
	case TabStream:
		return tv.StreamTab.View(styles)
	case TabIssues:
		return tv.IssuesTab.View(styles)
	case TabDashboard:
		return tv.SummaryTab.View(styles)
	default:
		return ""
	}
}

// =============================================================================
// Line Classification
// =============================================================================

// TabLineType categorizes log lines for styling
type TabLineType int

const (
	TabLineTypeNormal TabLineType = iota
	TabLineTypeError
	TabLineTypeWarning
	TabLineTypeNote
	TabLineTypeVerbose
	TabLineTypeProgress
	TabLineTypePhaseHeader
)

// classifyTabLogLine determines the type of a log line
func classifyTabLogLine(line string) TabLineType {
	lower := strings.ToLower(line)

	// Errors/warnings/notes first (use shared heuristics)
	if severity := issueSeverity(line); severity != TabLineTypeNormal {
		return severity
	}

	// Progress patterns
	if strings.Contains(lower, " of ") && strings.Contains(lower, "task") {
		return TabLineTypeProgress
	}

	// Phase header patterns
	if strings.HasPrefix(line, "===") ||
		strings.Contains(lower, "compiling") ||
		strings.Contains(lower, "linking") ||
		strings.Contains(lower, "signing") {
		return TabLineTypePhaseHeader
	}

	// Verbose patterns (less important output)
	if strings.HasPrefix(line, "    ") ||
		strings.Contains(lower, "creating") ||
		strings.Contains(lower, "copying") {
		return TabLineTypeVerbose
	}

	return TabLineTypeNormal
}

// =============================================================================
// PhaseResult for summary
// =============================================================================

// PhaseResult holds timing info for a build phase
type PhaseResult struct {
	Name     string
	Duration string
	Count    int // e.g., file count for compile phase
}
