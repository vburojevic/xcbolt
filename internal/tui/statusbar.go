package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Status Bar - Top bar with 3-section layout
// Left: Project + Branch | Center: Scheme + Destination | Right: Status
// =============================================================================

// StatusBar renders the top status bar
type StatusBar struct {
	// Project info
	ProjectName   string
	GitBranch     string
	Scheme        string
	Configuration string
	Destination   string
	DestOS        string
	DryRun        bool

	// Running state
	Running    bool
	RunningCmd string
	Stage      string
	Progress   string

	// Last result (shown when not running)
	HasLastResult     bool
	LastResultSuccess bool
	LastResultOp      string
	LastResultTime    string

	// Error/warning counts
	ErrorCount   int
	WarningCount int

	// Spinner for animation
	Spinner spinner.Model
}

// NewStatusBar creates a new status bar
func NewStatusBar() StatusBar {
	sp := spinner.New()
	sp.Spinner = spinner.Dot

	return StatusBar{
		Spinner: sp,
	}
}

// View renders the status bar content with 3-section layout
func (s StatusBar) View(width int, styles Styles) string {
	return s.ViewWithMinimal(width, styles, false)
}

// ViewMinimal renders a compact single-line status for small terminals
func (s StatusBar) ViewMinimal(width int, styles Styles) string {
	return s.ViewWithMinimal(width, styles, true)
}

// ViewWithMinimal renders the status bar, optionally in minimal mode
func (s StatusBar) ViewWithMinimal(width int, styles Styles, minimal bool) string {
	icons := styles.Icons

	if minimal {
		return s.renderMinimalView(width, styles, icons)
	}

	// === LEFT SECTION: Project + Brand ===
	leftContent := s.renderLeftSection(styles, icons)

	// === CENTER SECTION: Scheme + Destination ===
	centerContent := s.renderCenterSection(styles)

	// === RIGHT SECTION: Status ===
	rightContent := s.renderRightSection(styles, icons)

	// Calculate widths and spacing
	leftWidth := lipgloss.Width(leftContent)
	centerWidth := lipgloss.Width(centerContent)
	rightWidth := lipgloss.Width(rightContent)

	// Distribute space: try to center the center section
	totalContentWidth := leftWidth + centerWidth + rightWidth
	availableSpace := width - totalContentWidth - 4 // padding

	var content string
	if availableSpace > 0 {
		// Distribute space evenly on both sides of center
		leftSpace := availableSpace / 2
		rightSpace := availableSpace - leftSpace

		leftSpacer := strings.Repeat(" ", maxInt(1, leftSpace))
		rightSpacer := strings.Repeat(" ", maxInt(1, rightSpace))

		content = lipgloss.JoinHorizontal(lipgloss.Center,
			leftContent, leftSpacer, centerContent, rightSpacer, rightContent)
	} else {
		// Not enough space - just join with minimal spacing
		content = lipgloss.JoinHorizontal(lipgloss.Center,
			leftContent, " ", centerContent, " ", rightContent)
	}

	// Ensure we always return something visible
	if strings.TrimSpace(content) == "" {
		brandStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Colors.Accent)
		content = brandStyle.Render(icons.Bolt + " xcbolt")
	}

	return content
}

// renderMinimalView renders compact single-line status: ⚡xcbolt | Scheme | Device | Status
func (s StatusBar) renderMinimalView(width int, styles Styles, icons Icons) string {
	sep := " | "

	// Brand with bolt icon
	brandStyle := lipgloss.NewStyle().Bold(true).Foreground(styles.Colors.Accent)
	brand := brandStyle.Render(icons.Bolt + " xcbolt")

	// Scheme (truncated if needed)
	scheme := s.Scheme
	if scheme == "" {
		scheme = "?"
	}
	if s.Configuration != "" {
		scheme = scheme + ":" + s.Configuration
	}
	if len(scheme) > 15 {
		scheme = scheme[:12] + "..."
	}
	schemeStyle := lipgloss.NewStyle().Foreground(styles.Colors.Text)

	// Device (truncated if needed)
	device := s.Destination
	if device == "" {
		device = "?"
	}
	if len(device) > 15 {
		device = device[:12] + "..."
	}
	deviceStyle := lipgloss.NewStyle().Foreground(styles.Colors.Text)

	// Status indicator
	var status string
	if s.Running {
		status = s.Spinner.View() + " " + s.Stage
	} else if s.HasLastResult {
		if s.LastResultSuccess {
			status = icons.Success
		} else {
			status = icons.Error
		}
	} else {
		status = icons.Idle
	}
	if s.DryRun {
		status = status + " DRY"
	}

	sepStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)

	return brand +
		sepStyle.Render(sep) +
		schemeStyle.Render(scheme) +
		sepStyle.Render(sep) +
		deviceStyle.Render(device) +
		sepStyle.Render(sep) +
		status
}

// renderLeftSection renders project name and git branch
func (s StatusBar) renderLeftSection(styles Styles, icons Icons) string {
	var parts []string

	// Brand with bolt icon
	brandStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Colors.Accent)
	parts = append(parts, brandStyle.Render(icons.Bolt+" xcbolt"))

	sepStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextSubtle)
	sep := sepStyle.Render(" · ")

	// Project name
	if s.ProjectName != "" {
		projectStyle := lipgloss.NewStyle().
			Foreground(styles.Colors.Text)
		parts = append(parts, sep, projectStyle.Render(s.ProjectName))
	}

	// Git branch
	if s.GitBranch != "" {
		branchStyle := lipgloss.NewStyle().
			Foreground(styles.Colors.TextMuted)
		parts = append(parts, sep, branchStyle.Render(icons.Branch+" "+s.GitBranch))
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
}

// renderCenterSection renders scheme and destination
func (s StatusBar) renderCenterSection(styles Styles) string {
	var parts []string

	sepStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextSubtle)
	sep := sepStyle.Render(" · ")

	// Scheme
	schemeText := "No scheme"
	schemeStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)
	if s.Scheme != "" {
		schemeText = s.Scheme
		schemeStyle = lipgloss.NewStyle().Foreground(styles.Colors.Text)
	}
	parts = append(parts, schemeStyle.Render(schemeText))

	// Configuration
	if s.Configuration != "" {
		confStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)
		parts = append(parts, sep, confStyle.Render(s.Configuration))
	}

	// Destination
	destText := "No destination"
	destStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)
	if s.Destination != "" {
		destText = s.Destination
		if s.DestOS != "" {
			destText += " (" + s.DestOS + ")"
		}
		destStyle = lipgloss.NewStyle().Foreground(styles.Colors.Text)
	}
	parts = append(parts, sep, destStyle.Render(destText))

	if s.DryRun {
		dryStyle := styles.StatusStyle("warning")
		parts = append(parts, sep, dryStyle.Render("DRY RUN"))
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, parts...)
}

// renderRightSection renders status indicator (spinner or result)
func (s StatusBar) renderRightSection(styles Styles, icons Icons) string {
	if s.Running {
		// Running: use a static play icon + phase name
		icon := styles.StatusStyle("running").Render(icons.Run)
		var labelParts []string
		if s.Stage != "" {
			labelParts = append(labelParts, s.Stage)
		} else {
			labelParts = append(labelParts, "RUNNING")
		}
		labelStyle := lipgloss.NewStyle().Foreground(styles.Colors.Accent)
		label := labelStyle.Render(strings.Join(labelParts, " "))
		return icon + " " + label
	}

	// Not running: show result or idle
	var parts []string

	// Error/warning counts if any
	if s.ErrorCount > 0 || s.WarningCount > 0 {
		var countParts []string
		if s.ErrorCount > 0 {
			errStyle := styles.StatusStyle("error")
			countParts = append(countParts, errStyle.Render(icons.Error+" "+itoa(s.ErrorCount)))
		}
		if s.WarningCount > 0 {
			warnStyle := styles.StatusStyle("warning")
			countParts = append(countParts, warnStyle.Render(icons.Warning+" "+itoa(s.WarningCount)))
		}
		parts = append(parts, strings.Join(countParts, " "))
	} else if s.HasLastResult {
		// Last result: icon + duration
		resultIcon := icons.Success
		resultStatus := "success"
		if !s.LastResultSuccess {
			resultIcon = icons.Error
			resultStatus = "error"
		}

		resultStyle := styles.StatusStyle(resultStatus)
		resultPart := resultStyle.Render(resultIcon)
		if s.LastResultTime != "" {
			timeStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)
			resultPart += " " + timeStyle.Render(s.LastResultTime)
		}
		parts = append(parts, resultPart)
	} else {
		// Idle indicator
		idleStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)
		parts = append(parts, idleStyle.Render(icons.Idle))
	}

	return strings.Join(parts, " ")
}

// =============================================================================
// Progress Bar - Simplified to just spinner + phase (rendered in status bar)
// =============================================================================

// ProgressBar is kept for backwards compatibility but mostly unused now
type ProgressBar struct {
	Visible bool
	Label   string
	Current int
	Total   int
	Stage   string
	Width   int
}

// NewProgressBar creates a new progress bar
func NewProgressBar() ProgressBar {
	return ProgressBar{
		Visible: false,
	}
}

// SetProgress updates the progress bar state
func (p *ProgressBar) SetProgress(current, total int, stage string) {
	p.Current = current
	p.Total = total
	p.Stage = stage
	p.Visible = true
}

// Hide hides the progress bar
func (p *ProgressBar) Hide() {
	p.Visible = false
}

// View renders nothing - progress is now shown in status bar
func (p ProgressBar) View(width int, styles Styles) string {
	// Progress is now integrated into the status bar as spinner + phase
	// This method returns empty string - we keep it for API compatibility
	return ""
}

// itoa converts int to string (simple version)
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	var digits []byte
	negative := n < 0
	if negative {
		n = -n
	}

	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	if negative {
		return "-" + string(digits)
	}
	return string(digits)
}

// =============================================================================
// Hints Bar - Bottom bar with keyboard shortcuts
// =============================================================================

// HintsBar renders keyboard hints at the bottom
type HintsBar struct {
	Hints []HintItem
}

// HintItem represents a single hint
type HintItem struct {
	Key  string
	Desc string
}

// NewHintsBar creates a new hints bar with default hints
func NewHintsBar() HintsBar {
	return HintsBar{
		Hints: DefaultHints(),
	}
}

// DefaultHints returns the default hints for normal mode
func DefaultHints() []HintItem {
	return []HintItem{
		{Key: "b", Desc: "build"},
		{Key: "r", Desc: "run"},
		{Key: "t", Desc: "test"},
		{Key: "s", Desc: "scheme"},
		{Key: "~", Desc: "build config"},
		{Key: "d", Desc: "dest"},
		{Key: "1-3", Desc: "tabs"},
		{Key: "/", Desc: "search"},
		{Key: "?", Desc: "help"},
		{Key: "q", Desc: "quit"},
	}
}

// View renders the hints bar
func (h HintsBar) View(width int, styles Styles) string {
	return h.renderHints(h.Hints, styles)
}

// ViewWithContext renders hints based on context (kept for API compatibility)
func (h HintsBar) ViewWithContext(width int, styles Styles, issuesFocused bool, hasIssues bool) string {
	// With issues panel removed, always use default hints
	return h.renderHints(DefaultHints(), styles)
}

func (h HintsBar) renderHints(hints []HintItem, styles Styles) string {
	var parts []string

	keyStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Accent).
		Bold(true)
	descStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextMuted)

	for i, hint := range hints {
		part := keyStyle.Render(hint.Key) + ":" + descStyle.Render(hint.Desc)
		parts = append(parts, part)

		if i < len(hints)-1 {
			parts = append(parts, "  ")
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}
