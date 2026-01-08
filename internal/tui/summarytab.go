package tui

import (
	"fmt"
	"strings"
	"time"

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

	// Live activity (during build)
	CurrentPhase    string    // "Compiling", "Linking", etc.
	CurrentFile     string    // File being processed
	FileProgress    int       // Current file number
	FilesTotal      int       // Total files to process
	StartTime       time.Time // For elapsed timer
	RecentErrors    []string  // Last 3 errors (compact)
	PhaseList       []string  // All phases for timeline
	CompletedPhases []string  // Completed phase names

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
		Status:          BuildStatusPending,
		Phases:          make([]PhaseResult, 0),
		RecentErrors:    make([]string, 0, 3),
		PhaseList:       make([]string, 0),
		CompletedPhases: make([]string, 0),
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

	// Reset live activity
	st.CurrentPhase = ""
	st.CurrentFile = ""
	st.FileProgress = 0
	st.FilesTotal = 0
	st.StartTime = time.Time{}
	st.RecentErrors = st.RecentErrors[:0]
	st.PhaseList = st.PhaseList[:0]
	st.CompletedPhases = st.CompletedPhases[:0]
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
	st.StartTime = time.Now()
}

// =============================================================================
// Live Activity Updates
// =============================================================================

// UpdateProgress updates live build progress
func (st *SummaryTab) UpdateProgress(phase string, file string, current, total int) {
	st.CurrentPhase = phase
	st.CurrentFile = file
	st.FileProgress = current
	st.FilesTotal = total
}

// AddRecentError adds an error to the recent errors list (keeps last 3)
func (st *SummaryTab) AddRecentError(msg string) {
	// Truncate message if too long
	if len(msg) > 60 {
		msg = msg[:57] + "..."
	}
	st.RecentErrors = append(st.RecentErrors, msg)
	// Keep only last 3
	if len(st.RecentErrors) > 3 {
		st.RecentErrors = st.RecentErrors[len(st.RecentErrors)-3:]
	}
}

// SetPhases sets the list of all phases for the timeline
func (st *SummaryTab) SetPhases(phases []string) {
	st.PhaseList = phases
	st.CompletedPhases = st.CompletedPhases[:0]
}

// CompletePhase marks a phase as completed
func (st *SummaryTab) CompletePhase(phase string) {
	// Check if already completed
	for _, p := range st.CompletedPhases {
		if p == phase {
			return
		}
	}
	st.CompletedPhases = append(st.CompletedPhases, phase)
}

// ElapsedTime returns the elapsed time since build started
func (st *SummaryTab) ElapsedTime() string {
	if st.StartTime.IsZero() {
		return "0:00"
	}
	d := time.Since(st.StartTime)
	mins := int(d.Minutes())
	secs := int(d.Seconds()) % 60
	return fmt.Sprintf("%d:%02d", mins, secs)
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

// pendingView shows the waiting state with large bolt icon
func (st *SummaryTab) pendingView(styles Styles) string {
	icons := styles.Icons

	// Large bolt icon (5x)
	boltStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Accent).
		Bold(true).
		Padding(1, 0)
	bigBolt := boltStyle.Render(icons.Bolt + "  " + icons.Bolt + "  " + icons.Bolt + "  " + icons.Bolt + "  " + icons.Bolt)

	titleStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Text).
		Bold(true)
	title := titleStyle.Render(icons.Bolt + " xcbolt")

	msg := lipgloss.NewStyle().
		Foreground(styles.Colors.TextSubtle).
		Render("No build results yet")

	hint := lipgloss.NewStyle().
		Foreground(styles.Colors.TextSubtle).
		Render("Press b to build, r to run, t to test")

	content := lipgloss.JoinVertical(lipgloss.Center,
		bigBolt, "", title, "", msg, "", hint,
	)

	return lipgloss.Place(
		st.Width,
		st.Height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// runningView shows the in-progress state with live stats
func (st *SummaryTab) runningView(styles Styles) string {
	icons := styles.Icons
	var sections []string

	// === HEADER: BUILDING... + Elapsed Timer ===
	buildingIcon := lipgloss.NewStyle().
		Foreground(styles.Colors.Running).
		Bold(true).
		Render(icons.Running)

	buildingText := lipgloss.NewStyle().
		Foreground(styles.Colors.Running).
		Bold(true).
		Render("BUILDING...")

	timerStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextMuted)
	timer := timerStyle.Render("⏱ " + st.ElapsedTime())

	header := lipgloss.JoinHorizontal(lipgloss.Center,
		buildingIcon, " ", buildingText, "              ", timer)
	sections = append(sections, "", header, "")

	// === PROGRESS BAR: File Counter ===
	if st.FilesTotal > 0 {
		progressBar := st.renderProgressBar(styles)
		sections = append(sections, progressBar, "")
	}

	// === PHASE CARDS: Timeline of phases ===
	if len(st.PhaseList) > 0 {
		phaseCards := st.renderPhaseCards(styles)
		sections = append(sections, phaseCards, "")
	}

	// === CURRENT FILE ===
	if st.CurrentFile != "" {
		fileLabel := lipgloss.NewStyle().
			Foreground(styles.Colors.TextMuted).
			Render(st.CurrentPhase + ": ")
		fileName := st.CurrentFile
		// Shorten path if too long
		if len(fileName) > 40 {
			parts := strings.Split(fileName, "/")
			if len(parts) > 2 {
				fileName = ".../" + strings.Join(parts[len(parts)-2:], "/")
			}
		}
		fileStyle := lipgloss.NewStyle().
			Foreground(styles.Colors.Accent)
		fileLine := fileLabel + fileStyle.Render(fileName)
		sections = append(sections, fileLine, "")
	}

	// === RECENT ERRORS (last 3) ===
	if len(st.RecentErrors) > 0 {
		errorHeader := lipgloss.NewStyle().
			Foreground(styles.Colors.TextMuted).
			Render("Recent Issues:")
		sections = append(sections, errorHeader)

		for _, errMsg := range st.RecentErrors {
			errStyle := lipgloss.NewStyle().
				Foreground(styles.Colors.Error)
			errLine := errStyle.Render(icons.Error + " " + errMsg)
			sections = append(sections, errLine)
		}
	}

	content := lipgloss.JoinVertical(lipgloss.Center, sections...)

	return lipgloss.Place(
		st.Width,
		st.Height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// renderProgressBar renders a file progress bar
func (st *SummaryTab) renderProgressBar(styles Styles) string {
	width := 30
	if st.FilesTotal == 0 {
		return ""
	}

	// Calculate progress
	progress := float64(st.FileProgress) / float64(st.FilesTotal)
	filled := int(progress * float64(width))
	if filled > width {
		filled = width
	}

	// Build progress bar
	filledStyle := lipgloss.NewStyle().Foreground(styles.Colors.Accent)
	emptyStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextSubtle)

	bar := filledStyle.Render(strings.Repeat("═", filled)) +
		emptyStyle.Render(strings.Repeat("░", width-filled))

	// Counter
	counterStyle := lipgloss.NewStyle().Foreground(styles.Colors.Text)
	counter := counterStyle.Render(fmt.Sprintf(" %d/%d", st.FileProgress, st.FilesTotal))

	return bar + counter
}

// renderPhaseCards renders the phase timeline
func (st *SummaryTab) renderPhaseCards(styles Styles) string {
	icons := styles.Icons
	var parts []string

	for i, phase := range st.PhaseList {
		isCompleted := st.isPhaseCompleted(phase)
		isCurrent := phase == st.CurrentPhase

		var icon string
		var style lipgloss.Style

		if isCompleted {
			icon = icons.Success
			style = lipgloss.NewStyle().Foreground(styles.Colors.Success)
		} else if isCurrent {
			icon = icons.Running
			style = lipgloss.NewStyle().Foreground(styles.Colors.Accent).Bold(true)
		} else {
			icon = "○"
			style = lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)
		}

		card := style.Render(icon + " " + phase)
		parts = append(parts, card)

		// Add arrow separator (except after last)
		if i < len(st.PhaseList)-1 {
			arrowStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextSubtle)
			parts = append(parts, arrowStyle.Render(" → "))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
}

// isPhaseCompleted checks if a phase is completed
func (st *SummaryTab) isPhaseCompleted(phase string) bool {
	for _, p := range st.CompletedPhases {
		if p == phase {
			return true
		}
	}
	return false
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
	hint := hintStyle.Render("Press 3 to view Issues tab for details")
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
