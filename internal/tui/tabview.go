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
		return "Stream"
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
	StreamLines  int // Total lines in stream
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
func (tv *TabView) SetBuildResult(success bool, duration string, phases []PhaseResult) {
	tv.SummaryTab.SetResult(success, duration, phases, tv.Counts.ErrorCount, tv.Counts.WarningCount)
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

// renderTabBar renders the 2-line tab bar
// NOTE: Background is pushed into each cell to prevent ANSI reset "holes"
func (tv *TabView) renderTabBar(styles Styles) string {
	s := styles.TabBar
	icons := styles.Icons
	bgColor := styles.Colors.Surface

	// Build each tab (order: Dashboard, Stream, Issues)
	tabs := []struct {
		tab      Tab
		icon     string
		label    string
		subtitle string
		badge    string
	}{
		{
			tab:      TabDashboard,
			icon:     icons.TabSummary, // Dashboard icon
			label:    "Dashboard",
			subtitle: "Live stats",
			badge:    "",
		},
		{
			tab:      TabStream,
			icon:     icons.TabStream,
			label:    "Stream",
			subtitle: "Live output",
			badge:    "",
		},
		{
			tab:      TabIssues,
			icon:     icons.TabIssues,
			label:    "Issues",
			subtitle: tv.issuesSubtitle(),
			badge:    tv.issuesBadge(),
		},
	}

	// Calculate tab width - distribute evenly
	tabWidth := tv.Width / 3
	if tabWidth < 15 {
		tabWidth = 15
	}
	// Last tab gets remaining width to fill exactly
	lastTabWidth := tv.Width - (tabWidth * 2)

	var line1Parts []string
	var line2Parts []string

	for i, t := range tabs {
		isActive := tv.ActiveTab == t.tab
		cellWidth := tabWidth
		if i == 2 {
			cellWidth = lastTabWidth
		}

		// Base cell style with background - key fix for color consistency
		cellBase := lipgloss.NewStyle().
			Background(bgColor).
			Width(cellWidth).
			MaxWidth(cellWidth).
			MaxHeight(1)

		// Build line 1 content: [num] icon label (badge)
		keyHint := lipgloss.NewStyle().Background(bgColor).Foreground(styles.Colors.TextSubtle).Render(fmt.Sprintf("[%d]", i+1))

		var iconStr, labelStr string
		if isActive {
			iconStr = lipgloss.NewStyle().Background(bgColor).Foreground(styles.Colors.Accent).Render(t.icon)
			labelStr = lipgloss.NewStyle().Background(bgColor).Foreground(styles.Colors.Accent).Bold(true).Render(t.label)
		} else {
			iconStr = lipgloss.NewStyle().Background(bgColor).Foreground(styles.Colors.TextSubtle).Render(t.icon)
			labelStr = lipgloss.NewStyle().Background(bgColor).Foreground(styles.Colors.TextMuted).Render(t.label)
		}

		line1Content := " " + keyHint + " " + iconStr + " " + labelStr
		if t.badge != "" {
			badgeStyle := s.TabBadge.Background(bgColor)
			if tv.Counts.ErrorCount > 0 && t.tab == TabIssues {
				badgeStyle = s.TabBadgeError.Background(bgColor)
			}
			line1Content += " " + badgeStyle.Render(t.badge)
		}

		// Build line 2 content: subtitle (indented)
		var subtitleStr string
		if isActive {
			subtitleStr = lipgloss.NewStyle().Background(bgColor).Foreground(styles.Colors.TextMuted).Render(t.subtitle)
		} else {
			subtitleStr = lipgloss.NewStyle().Background(bgColor).Foreground(styles.Colors.TextSubtle).Render(t.subtitle)
		}
		line2Content := "     " + subtitleStr

		// Render cells with background
		line1Parts = append(line1Parts, cellBase.Render(line1Content))
		line2Parts = append(line2Parts, cellBase.Render(line2Content))
	}

	line1 := lipgloss.JoinHorizontal(lipgloss.Top, line1Parts...)
	line2 := lipgloss.JoinHorizontal(lipgloss.Top, line2Parts...)

	// Render border manually with background
	border := lipgloss.NewStyle().
		Background(bgColor).
		Foreground(styles.Colors.Border).
		Width(tv.Width).
		Render(strings.Repeat("─", tv.Width))

	return lipgloss.JoinVertical(lipgloss.Left, line1, line2, border)
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

	// Error patterns
	if strings.Contains(lower, "error:") ||
		strings.Contains(lower, "fatal error") ||
		strings.Contains(lower, "build failed") ||
		strings.Contains(lower, "❌") ||
		strings.Contains(line, "✗") {
		return TabLineTypeError
	}

	// Warning patterns
	if strings.Contains(lower, "warning:") ||
		strings.Contains(lower, "⚠") {
		return TabLineTypeWarning
	}

	// Note patterns
	if strings.Contains(lower, "note:") ||
		strings.Contains(lower, "remark:") {
		return TabLineTypeNote
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
