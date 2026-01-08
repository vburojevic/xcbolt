package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// SummaryTab - Build results dashboard
// =============================================================================

// BuildStatus represents the current build state
type BuildStatus int

const (
	BuildStatusPending BuildStatus = iota
	BuildStatusRunning
	BuildStatusSuccess
	BuildStatusFailed
)

// SummaryTab displays build results and statistics
type SummaryTab struct {
	Status       BuildStatus
	Duration     string
	Phases       []PhaseResult
	ErrorCount   int
	WarningCount int

	// Additional stats
	FileCount   int
	TargetCount int

	// Scroll state
	ScrollPos   int
	VisibleRows int

	// Dimensions
	Width  int
	Height int
}

// NewSummaryTab creates a new SummaryTab
func NewSummaryTab() *SummaryTab {
	return &SummaryTab{
		Status: BuildStatusPending,
		Phases: make([]PhaseResult, 0),
	}
}

// SetSize updates dimensions
func (st *SummaryTab) SetSize(width, height int) {
	st.Width = width
	st.Height = height
	st.VisibleRows = height
}

// Clear resets the summary
func (st *SummaryTab) Clear() {
	st.Status = BuildStatusPending
	st.Duration = ""
	st.Phases = st.Phases[:0]
	st.ErrorCount = 0
	st.WarningCount = 0
	st.FileCount = 0
	st.TargetCount = 0
	st.ScrollPos = 0
}

// SetResult updates the summary with build results
func (st *SummaryTab) SetResult(success bool, duration string, phases []PhaseResult, errors, warnings int) {
	if success {
		st.Status = BuildStatusSuccess
	} else {
		st.Status = BuildStatusFailed
	}
	st.Duration = duration
	st.Phases = phases
	st.ErrorCount = errors
	st.WarningCount = warnings

	// Count files from phases
	for _, p := range phases {
		st.FileCount += p.Count
	}
}

// SetRunning marks the build as in progress
func (st *SummaryTab) SetRunning() {
	st.Status = BuildStatusRunning
}

// =============================================================================
// Scrolling
// =============================================================================

func (st *SummaryTab) maxScrollPos() int {
	// Estimate content height
	contentHeight := 10 + len(st.Phases)*2
	max := contentHeight - st.VisibleRows
	if max < 0 {
		return 0
	}
	return max
}

// ScrollUp scrolls up
func (st *SummaryTab) ScrollUp(n int) {
	st.ScrollPos -= n
	if st.ScrollPos < 0 {
		st.ScrollPos = 0
	}
}

// ScrollDown scrolls down
func (st *SummaryTab) ScrollDown(n int) {
	st.ScrollPos += n
	max := st.maxScrollPos()
	if st.ScrollPos > max {
		st.ScrollPos = max
	}
}

// GotoTop scrolls to top
func (st *SummaryTab) GotoTop() {
	st.ScrollPos = 0
}

// GotoBottom scrolls to bottom
func (st *SummaryTab) GotoBottom() {
	st.ScrollPos = st.maxScrollPos()
}

// =============================================================================
// View Rendering
// =============================================================================

// View renders the summary tab content
func (st *SummaryTab) View(styles Styles) string {
	switch st.Status {
	case BuildStatusPending:
		return st.pendingView(styles)
	case BuildStatusRunning:
		return st.runningView(styles)
	case BuildStatusSuccess:
		return st.successView(styles)
	case BuildStatusFailed:
		return st.failedView(styles)
	default:
		return st.pendingView(styles)
	}
}

// pendingView shows the waiting state
func (st *SummaryTab) pendingView(styles Styles) string {
	icons := styles.Icons

	iconStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextMuted)

	msgStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextSubtle)

	hintStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextSubtle)

	icon := iconStyle.Render(icons.TabSummary)
	msg := msgStyle.Render("No build results yet")
	hint := hintStyle.Render("Press b to build, r to run, t to test")

	content := lipgloss.JoinVertical(lipgloss.Center,
		"", "", icon, "", msg, "", hint,
	)

	return lipgloss.Place(
		st.Width,
		st.Height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// runningView shows the in-progress state
func (st *SummaryTab) runningView(styles Styles) string {
	icons := styles.Icons

	iconStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Running)

	msgStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Text)

	icon := iconStyle.Render(icons.Running)
	msg := msgStyle.Render("Build in progress...")

	content := lipgloss.JoinVertical(lipgloss.Center,
		"", "", icon, "", msg,
	)

	return lipgloss.Place(
		st.Width,
		st.Height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// successView shows the success dashboard
func (st *SummaryTab) successView(styles Styles) string {
	icons := styles.Icons
	var sections []string

	// Big success indicator
	successIcon := lipgloss.NewStyle().
		Foreground(styles.Colors.Success).
		Bold(true).
		Render(icons.Success)

	successText := lipgloss.NewStyle().
		Foreground(styles.Colors.Success).
		Bold(true).
		Render("BUILD SUCCEEDED")

	header := lipgloss.JoinHorizontal(lipgloss.Center, successIcon, " ", successText)
	sections = append(sections, "", header, "")

	// Duration
	if st.Duration != "" {
		durationStyle := lipgloss.NewStyle().
			Foreground(styles.Colors.Text)
		durationLabel := lipgloss.NewStyle().
			Foreground(styles.Colors.TextMuted).
			Render("Duration: ")
		duration := durationStyle.Render(st.Duration)
		sections = append(sections, durationLabel+duration, "")
	}

	// Phase breakdown
	if len(st.Phases) > 0 {
		phaseHeader := lipgloss.NewStyle().
			Foreground(styles.Colors.TextMuted).
			Bold(true).
			Render("Phase Breakdown")
		sections = append(sections, phaseHeader)

		for _, phase := range st.Phases {
			phaseLine := st.renderPhase(phase, styles)
			sections = append(sections, phaseLine)
		}
		sections = append(sections, "")
	}

	// Stats
	statsSection := st.renderStats(styles)
	if statsSection != "" {
		sections = append(sections, statsSection)
	}

	// Warnings (if any)
	if st.WarningCount > 0 {
		warnStyle := lipgloss.NewStyle().
			Foreground(styles.Colors.Warning)
		warnLine := warnStyle.Render(fmt.Sprintf("%s %d warning(s)", icons.Warning, st.WarningCount))
		sections = append(sections, "", warnLine)
	}

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Center vertically
	return lipgloss.Place(
		st.Width,
		st.Height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// failedView shows the failure dashboard
func (st *SummaryTab) failedView(styles Styles) string {
	icons := styles.Icons
	var sections []string

	// Big failure indicator
	failIcon := lipgloss.NewStyle().
		Foreground(styles.Colors.Error).
		Bold(true).
		Render(icons.Error)

	failText := lipgloss.NewStyle().
		Foreground(styles.Colors.Error).
		Bold(true).
		Render("BUILD FAILED")

	header := lipgloss.JoinHorizontal(lipgloss.Center, failIcon, " ", failText)
	sections = append(sections, "", header, "")

	// Duration (if any)
	if st.Duration != "" {
		durationStyle := lipgloss.NewStyle().
			Foreground(styles.Colors.TextMuted)
		durationLabel := lipgloss.NewStyle().
			Foreground(styles.Colors.TextSubtle).
			Render("Duration: ")
		duration := durationStyle.Render(st.Duration)
		sections = append(sections, durationLabel+duration, "")
	}

	// Error count
	if st.ErrorCount > 0 {
		errorStyle := lipgloss.NewStyle().
			Foreground(styles.Colors.Error)
		errorLine := errorStyle.Render(fmt.Sprintf("%s %d error(s)", icons.Error, st.ErrorCount))
		sections = append(sections, errorLine)
	}

	// Warning count
	if st.WarningCount > 0 {
		warnStyle := lipgloss.NewStyle().
			Foreground(styles.Colors.Warning)
		warnLine := warnStyle.Render(fmt.Sprintf("%s %d warning(s)", icons.Warning, st.WarningCount))
		sections = append(sections, warnLine)
	}

	sections = append(sections, "")

	// Hint to check Issues tab
	hintStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextSubtle)
	hint := hintStyle.Render("Press 2 to view Issues tab for details")
	sections = append(sections, hint)

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	// Center vertically
	return lipgloss.Place(
		st.Width,
		st.Height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// renderPhase renders a single phase result line
func (st *SummaryTab) renderPhase(phase PhaseResult, styles Styles) string {
	nameStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Text).
		Width(15)

	durationStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextMuted).
		Width(10)

	countStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextSubtle)

	name := nameStyle.Render(phase.Name)
	duration := durationStyle.Render(phase.Duration)

	var count string
	if phase.Count > 0 {
		count = countStyle.Render(fmt.Sprintf("(%d files)", phase.Count))
	}

	return "  " + name + duration + count
}

// renderStats renders the stats section
func (st *SummaryTab) renderStats(styles Styles) string {
	if st.FileCount == 0 && st.TargetCount == 0 {
		return ""
	}

	var parts []string

	labelStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextMuted)

	valueStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Text)

	if st.FileCount > 0 {
		parts = append(parts, labelStyle.Render("Files compiled: ")+valueStyle.Render(fmt.Sprintf("%d", st.FileCount)))
	}

	if st.TargetCount > 0 {
		parts = append(parts, labelStyle.Render("Targets built: ")+valueStyle.Render(fmt.Sprintf("%d", st.TargetCount)))
	}

	return strings.Join(parts, "  ")
}
