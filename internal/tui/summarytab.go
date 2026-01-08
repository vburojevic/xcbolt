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
)

// Spinner frames for animation
var spinnerFrames = []string{"◐", "◓", "◑", "◒"}

// SummaryTab displays the dashboard with project info, build status, and actions
type SummaryTab struct {
	Status BuildStatus

	// Project Info (idle state)
	ProjectName  string
	SchemeName   string
	TargetDevice string
	BuildConfig  string

	// System Info (idle state)
	XcodeVersion    string
	SimulatorStatus string
	DeviceConnected bool

	// Build Progress
	CurrentFile  string    // Filename only
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

// SetSystemInfo sets system information for the idle state
func (st *SummaryTab) SetSystemInfo(xcode, simulator string, deviceConnected bool) {
	st.XcodeVersion = xcode
	st.SimulatorStatus = simulator
	st.DeviceConnected = deviceConnected
}

// SetRunning marks the build as in progress
func (st *SummaryTab) SetRunning() {
	st.Status = BuildStatusRunning
	st.StartTime = time.Now()
	st.SpinnerFrame = 0
	st.ErrorCount = 0
	st.WarningCount = 0
}

// UpdateProgress updates live build progress
func (st *SummaryTab) UpdateProgress(file string, current, total int) {
	// Extract just the filename
	if file != "" {
		st.CurrentFile = filepath.Base(file)
	}
	st.FileProgress = current
	st.FilesTotal = total
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
	default:
		return st.idleView(styles)
	}
}

// idleView shows project info, system status, last build, and quick actions
func (st *SummaryTab) idleView(styles Styles) string {
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
		projectContent = append(projectContent, "No project loaded")
	}
	if st.SchemeName != "" || st.BuildConfig != "" {
		line := ""
		if st.SchemeName != "" {
			line += "Scheme: " + st.SchemeName
		}
		if st.BuildConfig != "" {
			if line != "" {
				line += "   "
			}
			line += "Config: " + st.BuildConfig
		}
		projectContent = append(projectContent, line)
	}
	if st.TargetDevice != "" {
		projectContent = append(projectContent, "Target: "+st.TargetDevice)
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

// buildingView shows live build progress with spinner
func (st *SummaryTab) buildingView(styles Styles) string {
	cardWidth := st.Width - 8
	if cardWidth > 70 {
		cardWidth = 70
	}
	if cardWidth < 40 {
		cardWidth = 40
	}

	var cards []string

	// Main Building Card
	buildContent := []string{}

	// Spinner + BUILDING... + Timer
	spinner := spinnerFrames[st.SpinnerFrame]
	spinnerStyle := lipgloss.NewStyle().Foreground(styles.Colors.Running).Bold(true)
	timerStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)

	headerLine := spinnerStyle.Render(spinner+" BUILDING...") +
		strings.Repeat(" ", cardWidth-30) +
		timerStyle.Render(st.ElapsedTime())
	buildContent = append(buildContent, headerLine)
	buildContent = append(buildContent, "")

	// Progress bar with dots
	if st.FilesTotal > 0 {
		progressBar := st.renderDotProgress(cardWidth-10, styles)
		buildContent = append(buildContent, progressBar)
		buildContent = append(buildContent, "")
	}

	// Current file
	fileLabel := "Compiling: "
	fileName := st.CurrentFile
	if fileName == "" {
		fileName = "..."
	}
	fileStyle := lipgloss.NewStyle().Foreground(styles.Colors.Accent)
	fileLine := fileLabel + fileStyle.Render(fileName)
	buildContent = append(buildContent, fileLine)

	cards = append(cards, st.renderCard("Building", buildContent, cardWidth, styles))

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

	// Center the text
	padding := (cardWidth - 20) / 2
	if padding < 0 {
		padding = 0
	}
	successContent = append(successContent, strings.Repeat(" ", padding)+successText)
	successContent = append(successContent, strings.Repeat(" ", padding+5)+st.Duration)
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

	// Center the text
	padding := (cardWidth - 18) / 2
	if padding < 0 {
		padding = 0
	}
	failedContent = append(failedContent, strings.Repeat(" ", padding)+failedText)
	failedContent = append(failedContent, strings.Repeat(" ", padding+5)+st.Duration)
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
	summaryContent = append(summaryContent, hintStyle.Render("Press 3 to view Issues"))
	cards = append(cards, st.renderCard("Summary", summaryContent, cardWidth, styles))

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
