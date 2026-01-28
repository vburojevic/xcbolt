package tui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// SummaryTab - Dashboard with card-based layout
// =============================================================================

// BuildStatus represents the current build state
type BuildStatus int

const (
	BuildStatusPending BuildStatus = iota
	BuildStatusRunning
	BuildStatusSuccess
	BuildStatusFailed
	BuildStatusCanceled
)

// Spinner frames for animation
var spinnerFrames = []string{"◐", "◓", "◑", "◒"}

// SummaryTab displays the dashboard with project info, build status, and actions
type SummaryTab struct {
	Status BuildStatus

	// Current action being performed (build, run, test, clean)
	ActionType string

	// Project Info (idle state)
	ProjectName  string
	SchemeName   string
	TargetDevice string
	BuildConfig  string
	BundleID     string

	// System Info (idle state)
	XcodeVersion    string
	SimulatorStatus string
	DeviceConnected bool

	// Build Progress
	CurrentFile  string // Filename only
	CurrentStage string // Current build stage (Compile, Link, Sign, etc.)
	LogIdle      time.Duration
	FileProgress int       // Current file number
	FilesTotal   int       // Total files to process
	StartTime    time.Time // For elapsed timer
	SpinnerFrame int       // 0-3 for animation

	// Results
	Duration     string
	ErrorCount   int
	WarningCount int

	// Last Build (for idle state)
	LastBuildSuccess  bool
	LastBuildDuration string
	LastBuildErrors   int
	LastBuildWarnings int
	HasLastBuild      bool

	// Legacy fields (kept for compatibility)
	Phases    []PhaseResult
	FileCount int

	// Context loading state
	ContextLoaded bool // Set to true when context discovery completes

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

// Clear resets for a new build (preserves project/system info and last build)
func (st *SummaryTab) Clear() {
	// Save last build info before clearing
	if st.Status == BuildStatusSuccess || st.Status == BuildStatusFailed {
		st.HasLastBuild = true
		st.LastBuildSuccess = st.Status == BuildStatusSuccess
		st.LastBuildDuration = st.Duration
		st.LastBuildErrors = st.ErrorCount
		st.LastBuildWarnings = st.WarningCount
	}

	st.Status = BuildStatusPending
	st.Duration = ""
	st.ErrorCount = 0
	st.WarningCount = 0
	st.FileCount = 0
	st.ScrollPos = 0

	// Reset build progress
	st.CurrentFile = ""
	st.FileProgress = 0
	st.FilesTotal = 0
	st.StartTime = time.Time{}
	st.SpinnerFrame = 0
	st.Phases = st.Phases[:0]
}

// =============================================================================
// Data Setters
// =============================================================================

// SetProjectInfo sets project information for the idle state
func (st *SummaryTab) SetProjectInfo(name, scheme, target, config string) {
	st.ProjectName = name
	st.SchemeName = scheme
	st.TargetDevice = target
	st.BuildConfig = config
}

func (st *SummaryTab) SetAppInfo(bundleID string) {
	st.BundleID = bundleID
}

// SetSystemInfo sets system information for the idle state
func (st *SummaryTab) SetSystemInfo(xcode, simulator string, deviceConnected bool) {
	st.XcodeVersion = xcode
	st.SimulatorStatus = simulator
	st.DeviceConnected = deviceConnected
}

// SetContextLoaded marks context discovery as complete
func (st *SummaryTab) SetContextLoaded(loaded bool) {
	st.ContextLoaded = loaded
}

// SetRunning marks an action as in progress
func (st *SummaryTab) SetRunning(actionType string) {
	st.Status = BuildStatusRunning
	st.ActionType = actionType
	st.StartTime = time.Now()
	st.SpinnerFrame = 0
	st.ErrorCount = 0
	st.WarningCount = 0
}

// UpdateProgress updates live build progress
func (st *SummaryTab) UpdateProgress(file string, current, total int, stage string) {
	// Extract just the filename
	if file != "" {
		st.CurrentFile = filepath.Base(file)
	} else {
		st.CurrentFile = ""
	}
	st.FileProgress = current
	st.FilesTotal = total
	if stage != "" {
		st.CurrentStage = stage
	} else {
		st.CurrentStage = ""
	}
}

// SetLogIdle updates the idle duration since the last log line.
func (st *SummaryTab) SetLogIdle(d time.Duration) {
	st.LogIdle = d
}

// IncrementErrors increments the error count
func (st *SummaryTab) IncrementErrors() {
	st.ErrorCount++
}

// IncrementWarnings increments the warning count
func (st *SummaryTab) IncrementWarnings() {
	st.WarningCount++
}

// SetResult updates with final build results
func (st *SummaryTab) SetResult(status BuildStatus, duration string, phases []PhaseResult, errors, warnings int) {
	switch status {
	case BuildStatusSuccess:
		st.Status = BuildStatusSuccess
	case BuildStatusCanceled:
		st.Status = BuildStatusCanceled
	default:
		st.Status = BuildStatusFailed
	}
	st.Duration = duration
	st.Phases = phases
	st.ErrorCount = errors
	st.WarningCount = warnings

	if st.Status == BuildStatusSuccess || st.Status == BuildStatusFailed {
		st.LastBuildSuccess = st.Status == BuildStatusSuccess
	}

	// Count files from phases
	st.FileCount = 0
	for _, p := range phases {
		st.FileCount += p.Count
	}
}

// AdvanceSpinner advances the spinner animation frame
func (st *SummaryTab) AdvanceSpinner() {
	st.SpinnerFrame = (st.SpinnerFrame + 1) % len(spinnerFrames)
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
	contentHeight := 20
	max := contentHeight - st.VisibleRows
	if max < 0 {
		return 0
	}
	return max
}

func (st *SummaryTab) ScrollUp(n int) {
	st.ScrollPos -= n
	if st.ScrollPos < 0 {
		st.ScrollPos = 0
	}
}

func (st *SummaryTab) ScrollDown(n int) {
	st.ScrollPos += n
	max := st.maxScrollPos()
	if st.ScrollPos > max {
		st.ScrollPos = max
	}
}

func (st *SummaryTab) GotoTop() {
	st.ScrollPos = 0
}

func (st *SummaryTab) GotoBottom() {
	st.ScrollPos = st.maxScrollPos()
}

// =============================================================================
// Card Rendering
// =============================================================================

// renderCard draws a bordered card with title
func (st *SummaryTab) renderCard(title string, content []string, width int, styles Styles) string {
	if width < 20 {
		width = 60
	}
	innerWidth := width - 4 // Account for borders and padding

	// Title line: ┌─ Title ─────────────────────┐
	titleStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)
	borderStyle := lipgloss.NewStyle().Foreground(styles.Colors.Border)

	titleText := titleStyle.Render(title)
	// Use lipgloss.Width for visual width (excludes ANSI codes)
	titleVisualLen := lipgloss.Width(title)
	dashesAfter := innerWidth - titleVisualLen - 1
	if dashesAfter < 0 {
		dashesAfter = 0
	}

	topLine := borderStyle.Render("┌─ ") +
		titleText +
		borderStyle.Render(" "+strings.Repeat("─", dashesAfter)+"┐")

	// Content lines: │ content                    │
	var lines []string
	lines = append(lines, topLine)

	for _, line := range content {
		// Use lipgloss.Width for visual width (excludes ANSI codes)
		visualWidth := lipgloss.Width(line)
		// Don't truncate - just pad to fill width
		padding := innerWidth - visualWidth
		if padding < 0 {
			padding = 0
		}
		contentLine := borderStyle.Render("│ ") +
			line +
			strings.Repeat(" ", padding) +
			borderStyle.Render(" │")
		lines = append(lines, contentLine)
	}

	// Bottom line: └────────────────────────────┘
	bottomLine := borderStyle.Render("└" + strings.Repeat("─", innerWidth+2) + "┘")
	lines = append(lines, bottomLine)

	return strings.Join(lines, "\n")
}

// =============================================================================
// View Rendering
// =============================================================================

// View renders the dashboard content
func (st *SummaryTab) View(styles Styles) string {
	switch st.Status {
	case BuildStatusPending:
		return st.idleView(styles)
	case BuildStatusRunning:
		return st.buildingView(styles)
	case BuildStatusSuccess:
		return st.successView(styles)
	case BuildStatusFailed:
		return st.failedView(styles)
	case BuildStatusCanceled:
		return st.canceledView(styles)
	default:
		return st.idleView(styles)
	}
}

// idleView shows project info, system status, last build, and quick actions
func (st *SummaryTab) idleView(styles Styles) string {
	// Show loading state if context hasn't loaded yet
	if !st.ContextLoaded {
		return st.loadingView(styles)
	}

	cardWidth := st.Width - 8
	if cardWidth > 70 {
		cardWidth = 70
	}
	if cardWidth < 40 {
		cardWidth = 40
	}

	var cards []string

	// Project Card
	projectContent := []string{}
	if st.ProjectName != "" {
		projectContent = append(projectContent, st.ProjectName)
	} else {
		projectContent = append(projectContent, "No project configured")
	}
	if st.SchemeName != "" {
		projectContent = append(projectContent, "Scheme: "+st.SchemeName)
	}
	if st.BuildConfig != "" {
		projectContent = append(projectContent, "Config: "+st.BuildConfig)
	}
	if st.TargetDevice != "" {
		projectContent = append(projectContent, "Target: "+st.TargetDevice)
	}
	if st.BundleID != "" {
		projectContent = append(projectContent, "Bundle: "+st.BundleID)
	}
	if len(projectContent) == 0 {
		projectContent = append(projectContent, "No project info available")
	}
	cards = append(cards, st.renderCard("Project", projectContent, cardWidth, styles))

	// System Card
	systemContent := []string{}
	line1 := ""
	if st.XcodeVersion != "" {
		line1 += st.XcodeVersion
	} else {
		line1 += "Xcode: Unknown"
	}
	if st.SimulatorStatus != "" {
		line1 += "   Simulator: " + st.SimulatorStatus
	}
	systemContent = append(systemContent, line1)

	deviceStatus := "Device: "
	if st.DeviceConnected {
		deviceStatus += "Connected"
	} else {
		deviceStatus += "Not connected"
	}
	systemContent = append(systemContent, deviceStatus)
	cards = append(cards, st.renderCard("System", systemContent, cardWidth, styles))

	// Last Build Card (if available)
	if st.HasLastBuild {
		lastBuildContent := []string{}
		var statusIcon, statusText string
		if st.LastBuildSuccess {
			statusIcon = styles.Icons.Success
			statusText = "Succeeded"
		} else {
			statusIcon = styles.Icons.Error
			statusText = "Failed"
		}
		summary := fmt.Sprintf("%s %s · %s · %d errors, %d warnings",
			statusIcon, statusText, st.LastBuildDuration,
			st.LastBuildErrors, st.LastBuildWarnings)
		lastBuildContent = append(lastBuildContent, summary)
		cards = append(cards, st.renderCard("Last Build", lastBuildContent, cardWidth, styles))
	}

	// Quick Actions
	actionStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Accent).
		Bold(true)
	keyStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextMuted)

	actions := lipgloss.JoinHorizontal(lipgloss.Center,
		actionStyle.Render("[B]"), keyStyle.Render(" Build   "),
		actionStyle.Render("[R]"), keyStyle.Render(" Run   "),
		actionStyle.Render("[T]"), keyStyle.Render(" Test   "),
		actionStyle.Render("[C]"), keyStyle.Render(" Clean"),
	)

	// Combine all cards
	content := lipgloss.JoinVertical(lipgloss.Center,
		"",
		strings.Join(cards, "\n\n"),
		"",
		actions,
	)

	return lipgloss.Place(
		st.Width,
		st.Height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// loadingView shows a loading indicator while context is being discovered
func (st *SummaryTab) loadingView(styles Styles) string {
	spinner := spinnerFrames[st.SpinnerFrame]
	spinnerStyle := lipgloss.NewStyle().Foreground(styles.Colors.Accent).Bold(true)
	textStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)

	content := lipgloss.JoinVertical(lipgloss.Center,
		"",
		spinnerStyle.Render(spinner),
		"",
		textStyle.Render("Loading project..."),
		"",
	)

	return lipgloss.Place(
		st.Width,
		st.Height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// buildingView shows live progress with spinner
func (st *SummaryTab) buildingView(styles Styles) string {
	cardWidth := st.Width - 8
	if cardWidth > 70 {
		cardWidth = 70
	}
	if cardWidth < 40 {
		cardWidth = 40
	}

	var cards []string

	// Determine action label based on ActionType
	actionLabel := "BUILDING"
	cardTitle := "Building"
	switch st.ActionType {
	case "clean":
		actionLabel = "CLEANING"
		cardTitle = "Cleaning"
	case "test":
		actionLabel = "TESTING"
		cardTitle = "Testing"
	case "run":
		actionLabel = "RUNNING"
		cardTitle = "Running"
	}

	// Main Progress Card
	buildContent := []string{}

	// Spinner + ACTION... + Timer (properly spaced)
	spinner := spinnerFrames[st.SpinnerFrame]
	spinnerStyle := lipgloss.NewStyle().Foreground(styles.Colors.Running).Bold(true)
	timerStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)

	actionText := spinnerStyle.Render(spinner + " " + actionLabel + "...")
	timerText := timerStyle.Render(st.ElapsedTime())
	actionWidth := lipgloss.Width(spinner+" "+actionLabel+"...") + lipgloss.Width(st.ElapsedTime())
	padding := cardWidth - 4 - actionWidth
	if padding < 1 {
		padding = 1
	}
	headerLine := actionText + strings.Repeat(" ", padding) + timerText
	buildContent = append(buildContent, headerLine)
	buildContent = append(buildContent, "")

	// Progress bar with dots
	if st.FilesTotal > 0 {
		progressBar := st.renderDotProgress(cardWidth-10, styles)
		buildContent = append(buildContent, progressBar)
		buildContent = append(buildContent, "")
	}

	// Current activity - show stage + file or just stage
	var activityLine string
	fileStyle := lipgloss.NewStyle().Foreground(styles.Colors.Accent)
	labelStyle := lipgloss.NewStyle().Foreground(styles.Colors.Text)

	if st.CurrentFile != "" {
		// We're compiling a specific file
		label := "Compiling: "
		if st.CurrentStage == "Link" {
			label = "Linking: "
		} else if st.CurrentStage == "Sign" {
			label = "Signing: "
		}
		activityLine = labelStyle.Render(label) + fileStyle.Render(st.CurrentFile)
	} else if st.CurrentStage != "" {
		// We have a stage but no specific file
		stageText := st.CurrentStage
		switch st.CurrentStage {
		case "Resolve":
			stageText = "Resolving dependencies..."
		case "Compile":
			stageText = "Preparing to compile..."
		case "Link":
			stageText = "Linking..."
		case "Sign":
			stageText = "Signing..."
		default:
			stageText = st.CurrentStage + "..."
		}
		activityLine = fileStyle.Render(stageText)
	} else {
		// Initial state - no stage or file yet
		switch st.ActionType {
		case "clean":
			activityLine = fileStyle.Render("Cleaning derived data...")
		case "test":
			activityLine = fileStyle.Render("Preparing tests...")
		case "run":
			activityLine = fileStyle.Render("Preparing to run...")
		default:
			activityLine = fileStyle.Render("Preparing build...")
		}
	}
	buildContent = append(buildContent, activityLine)

	cards = append(cards, st.renderCard(cardTitle, buildContent, cardWidth, styles))

	// Issues Card (only if errors or warnings)
	if st.ErrorCount > 0 || st.WarningCount > 0 {
		issuesContent := []string{}
		var parts []string
		if st.ErrorCount > 0 {
			errStyle := lipgloss.NewStyle().Foreground(styles.Colors.Error)
			parts = append(parts, errStyle.Render(fmt.Sprintf("%s %d errors", styles.Icons.Error, st.ErrorCount)))
		}
		if st.WarningCount > 0 {
			warnStyle := lipgloss.NewStyle().Foreground(styles.Colors.Warning)
			parts = append(parts, warnStyle.Render(fmt.Sprintf("%s %d warnings", styles.Icons.Warning, st.WarningCount)))
		}
		issuesContent = append(issuesContent, strings.Join(parts, "   "))
		cards = append(cards, st.renderCard("Issues", issuesContent, cardWidth, styles))
	}

	content := lipgloss.JoinVertical(lipgloss.Center,
		"",
		strings.Join(cards, "\n\n"),
	)

	return lipgloss.Place(
		st.Width,
		st.Height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// renderDotProgress renders the dot-style progress bar
func (st *SummaryTab) renderDotProgress(width int, styles Styles) string {
	if st.FilesTotal == 0 {
		return ""
	}

	progress := float64(st.FileProgress) / float64(st.FilesTotal)
	filled := int(progress * float64(width))
	if filled > width {
		filled = width
	}

	filledStyle := lipgloss.NewStyle().Foreground(styles.Colors.Accent)
	emptyStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextSubtle)
	counterStyle := lipgloss.NewStyle().Foreground(styles.Colors.Text)

	bar := filledStyle.Render(strings.Repeat("●", filled)) +
		emptyStyle.Render(strings.Repeat("○", width-filled))

	counter := counterStyle.Render(fmt.Sprintf("  %d/%d", st.FileProgress, st.FilesTotal))

	return bar + counter
}

// successView shows build success
func (st *SummaryTab) successView(styles Styles) string {
	cardWidth := st.Width - 8
	if cardWidth > 70 {
		cardWidth = 70
	}
	if cardWidth < 40 {
		cardWidth = 40
	}

	var cards []string

	// Success Card
	successContent := []string{""}
	successIcon := lipgloss.NewStyle().Foreground(styles.Colors.Success).Bold(true)
	successText := successIcon.Render(styles.Icons.Success + " BUILD SUCCEEDED")
	durationStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)
	durationText := durationStyle.Render(st.Duration)

	// Center text using lipgloss width calculations
	innerWidth := cardWidth - 4 // Account for card borders
	successContent = append(successContent, lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, successText))
	successContent = append(successContent, lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, durationText))
	successContent = append(successContent, "")

	cards = append(cards, st.renderCard("Build Succeeded", successContent, cardWidth, styles))

	// Summary Card
	summaryContent := []string{}
	var parts []string
	if st.ErrorCount > 0 {
		errStyle := lipgloss.NewStyle().Foreground(styles.Colors.Error)
		parts = append(parts, errStyle.Render(fmt.Sprintf("%s %d errors", styles.Icons.Error, st.ErrorCount)))
	} else {
		textStyle := lipgloss.NewStyle().Foreground(styles.Colors.Success)
		parts = append(parts, textStyle.Render(fmt.Sprintf("%s 0 errors", styles.Icons.Success)))
	}
	if st.WarningCount > 0 {
		warnStyle := lipgloss.NewStyle().Foreground(styles.Colors.Warning)
		parts = append(parts, warnStyle.Render(fmt.Sprintf("%s %d warnings", styles.Icons.Warning, st.WarningCount)))
	} else {
		textStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)
		parts = append(parts, textStyle.Render("0 warnings"))
	}
	summaryContent = append(summaryContent, strings.Join(parts, "   "))
	cards = append(cards, st.renderCard("Summary", summaryContent, cardWidth, styles))

	// Quick Actions
	actionStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Accent).
		Bold(true)
	keyStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextMuted)

	actions := lipgloss.JoinHorizontal(lipgloss.Center,
		actionStyle.Render("[B]"), keyStyle.Render(" Rebuild   "),
		actionStyle.Render("[R]"), keyStyle.Render(" Run   "),
		actionStyle.Render("[C]"), keyStyle.Render(" Clean"),
	)

	content := lipgloss.JoinVertical(lipgloss.Center,
		"",
		strings.Join(cards, "\n\n"),
		"",
		actions,
	)

	return lipgloss.Place(
		st.Width,
		st.Height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// failedView shows build failure
func (st *SummaryTab) failedView(styles Styles) string {
	cardWidth := st.Width - 8
	if cardWidth > 70 {
		cardWidth = 70
	}
	if cardWidth < 40 {
		cardWidth = 40
	}

	var cards []string

	// Failed Card
	failedContent := []string{""}
	failedIcon := lipgloss.NewStyle().Foreground(styles.Colors.Error).Bold(true)
	failedText := failedIcon.Render(styles.Icons.Error + " BUILD FAILED")
	durationStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)
	durationText := durationStyle.Render(st.Duration)

	// Center text using lipgloss width calculations
	innerWidth := cardWidth - 4 // Account for card borders
	failedContent = append(failedContent, lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, failedText))
	failedContent = append(failedContent, lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, durationText))
	failedContent = append(failedContent, "")

	cards = append(cards, st.renderCard("Build Failed", failedContent, cardWidth, styles))

	// Summary Card
	summaryContent := []string{}
	var parts []string
	if st.ErrorCount > 0 {
		errStyle := lipgloss.NewStyle().Foreground(styles.Colors.Error)
		parts = append(parts, errStyle.Render(fmt.Sprintf("%s %d errors", styles.Icons.Error, st.ErrorCount)))
	}
	if st.WarningCount > 0 {
		warnStyle := lipgloss.NewStyle().Foreground(styles.Colors.Warning)
		parts = append(parts, warnStyle.Render(fmt.Sprintf("%s %d warnings", styles.Icons.Warning, st.WarningCount)))
	}
	summaryContent = append(summaryContent, strings.Join(parts, "   "))

	hintStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextSubtle)
	summaryContent = append(summaryContent, hintStyle.Render("Press 2 to view Issues"))
	cards = append(cards, st.renderCard("Summary", summaryContent, cardWidth, styles))

	// Quick Actions
	actionStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Accent).
		Bold(true)
	keyStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextMuted)

	actions := lipgloss.JoinHorizontal(lipgloss.Center,
		actionStyle.Render("[B]"), keyStyle.Render(" Build   "),
		actionStyle.Render("[R]"), keyStyle.Render(" Run   "),
		actionStyle.Render("[T]"), keyStyle.Render(" Test   "),
		actionStyle.Render("[C]"), keyStyle.Render(" Clean"),
	)

	content := lipgloss.JoinVertical(lipgloss.Center,
		"",
		strings.Join(cards, "\n\n"),
		"",
		actions,
	)

	return lipgloss.Place(
		st.Width,
		st.Height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}

// canceledView shows a canceled build (user-initiated)
func (st *SummaryTab) canceledView(styles Styles) string {
	cardWidth := st.Width - 8
	if cardWidth > 70 {
		cardWidth = 70
	}
	if cardWidth < 40 {
		cardWidth = 40
	}

	var cards []string

	// Canceled Card
	canceledContent := []string{""}
	canceledIcon := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted).Bold(true)
	canceledText := canceledIcon.Render(styles.Icons.Paused + " BUILD CANCELED")
	durationStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)
	durationText := durationStyle.Render(st.Duration)

	// Center text using lipgloss width calculations
	innerWidth := cardWidth - 4 // Account for card borders
	canceledContent = append(canceledContent, lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, canceledText))
	if st.Duration != "" {
		canceledContent = append(canceledContent, lipgloss.PlaceHorizontal(innerWidth, lipgloss.Center, durationText))
	}
	canceledContent = append(canceledContent, "")

	cards = append(cards, st.renderCard("Build Canceled", canceledContent, cardWidth, styles))

	// Summary Card
	summaryContent := []string{}
	var parts []string
	if st.ErrorCount > 0 {
		errStyle := lipgloss.NewStyle().Foreground(styles.Colors.Error)
		parts = append(parts, errStyle.Render(fmt.Sprintf("%s %d errors", styles.Icons.Error, st.ErrorCount)))
	}
	if st.WarningCount > 0 {
		warnStyle := lipgloss.NewStyle().Foreground(styles.Colors.Warning)
		parts = append(parts, warnStyle.Render(fmt.Sprintf("%s %d warnings", styles.Icons.Warning, st.WarningCount)))
	}
	if len(parts) > 0 {
		summaryContent = append(summaryContent, strings.Join(parts, "   "))
	}

	hintStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextSubtle)
	if st.ErrorCount > 0 || st.WarningCount > 0 {
		summaryContent = append(summaryContent, hintStyle.Render("Press 2 to view Issues"))
	}

	summaryTitle := "Summary (canceled)"
	if st.ErrorCount > 0 {
		summaryTitle = "Summary"
	}
	cards = append(cards, st.renderCard(summaryTitle, summaryContent, cardWidth, styles))

	// Quick Actions
	actionStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Accent).
		Bold(true)
	keyStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextMuted)

	actions := lipgloss.JoinHorizontal(lipgloss.Center,
		actionStyle.Render("[B]"), keyStyle.Render(" Rebuild   "),
		actionStyle.Render("[C]"), keyStyle.Render(" Clean"),
	)

	content := lipgloss.JoinVertical(lipgloss.Center,
		"",
		strings.Join(cards, "\n\n"),
		"",
		actions,
	)

	return lipgloss.Place(
		st.Width,
		st.Height,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)
}
