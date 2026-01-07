package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Status Bar - Top bar showing project info and running status
// =============================================================================

// StatusBar renders the top status bar
type StatusBar struct {
	// Project info
	ProjectName string
	GitBranch   string
	Scheme      string
	Destination string
	DestOS      string

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

	// Error/warning counts (shown when not running and issues exist)
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

// View renders the status bar content
func (s StatusBar) View(width int, styles Styles) string {
	isCompact := width < 100
	icons := styles.Icons

	// Brand
	brandStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Colors.Accent)
	brand := brandStyle.Render("xcbolt")

	// Project name (if available)
	projectPart := ""
	if s.ProjectName != "" {
		projectStyle := lipgloss.NewStyle().
			Foreground(styles.Colors.Text)
		projectPart = projectStyle.Render(s.ProjectName)
	}

	// Git branch (if available)
	branchPart := ""
	if s.GitBranch != "" {
		branchStyle := lipgloss.NewStyle().
			Foreground(styles.Colors.TextMuted)
		branchPart = branchStyle.Render(icons.Branch + " " + s.GitBranch)
	}

	// Scheme
	schemeText := "No scheme"
	if s.Scheme != "" {
		schemeText = s.Scheme
	}
	schemeStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Text)
	schemePart := schemeStyle.Render(schemeText)

	// Destination
	destText := "No destination"
	if s.Destination != "" {
		destText = s.Destination
		if !isCompact && s.DestOS != "" {
			destText += " (" + s.DestOS + ")"
		}
	}
	destStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Text)
	destPart := destStyle.Render(destText)

	// Separator
	sepStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextSubtle)
	sep := sepStyle.Render(" · ")

	// Build left side
	var leftParts []string
	leftParts = append(leftParts, brand)

	if projectPart != "" {
		leftParts = append(leftParts, sep, projectPart)
	}
	if branchPart != "" {
		leftParts = append(leftParts, sep, branchPart)
	}
	leftParts = append(leftParts, sep, schemePart)
	leftParts = append(leftParts, sep, destPart)

	leftContent := lipgloss.JoinHorizontal(lipgloss.Center, leftParts...)

	// Status (right side)
	status := s.renderStatus(styles, icons)

	// Calculate spacing
	leftWidth := lipgloss.Width(leftContent)
	statusWidth := lipgloss.Width(status)
	spacerWidth := maxInt(1, width-leftWidth-statusWidth-4)
	spacer := strings.Repeat(" ", spacerWidth)

	return lipgloss.JoinHorizontal(lipgloss.Center, leftContent, spacer, status)
}

// renderStatus renders the status indicator on the right
func (s StatusBar) renderStatus(styles Styles, icons Icons) string {
	if s.Running {
		// Running status with spinner
		icon := styles.StatusStyle("running").Render(s.Spinner.View())

		var labelParts []string
		labelParts = append(labelParts, strings.ToUpper(s.RunningCmd))

		if s.Stage != "" {
			labelParts = append(labelParts, s.Stage)
		}
		if s.Progress != "" {
			labelParts = append(labelParts, s.Progress)
		}

		label := strings.Join(labelParts, " ")
		labelStyle := lipgloss.NewStyle().
			Foreground(styles.Colors.TextMuted)

		return icon + " " + labelStyle.Render(label)
	}

	// Build result + idle status
	var parts []string

	// Error/warning counts: [✗ 3  ⚠ 2]
	if s.ErrorCount > 0 || s.WarningCount > 0 {
		bracketStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextSubtle)
		var countParts []string

		if s.ErrorCount > 0 {
			errStyle := styles.StatusStyle("error")
			countParts = append(countParts, errStyle.Render(icons.Error+" "+itoa(s.ErrorCount)))
		}
		if s.WarningCount > 0 {
			warnStyle := styles.StatusStyle("warning")
			countParts = append(countParts, warnStyle.Render(icons.Warning+" "+itoa(s.WarningCount)))
		}

		counts := strings.Join(countParts, " ")
		parts = append(parts, bracketStyle.Render("[")+counts+bracketStyle.Render("]"))
	} else if s.HasLastResult {
		// Last result indicator: [✓ 2.3s] or [✗] (only if no error counts)
		resultIcon := icons.Success
		resultStatus := "success"
		if !s.LastResultSuccess {
			resultIcon = icons.Error
			resultStatus = "error"
		}

		resultStyle := styles.StatusStyle(resultStatus)
		resultText := resultStyle.Render(resultIcon)
		if s.LastResultTime != "" {
			resultText += " " + lipgloss.NewStyle().Foreground(styles.Colors.TextMuted).Render(s.LastResultTime)
		}

		bracketStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextSubtle)
		parts = append(parts, bracketStyle.Render("[")+resultText+bracketStyle.Render("]"))
	}

	// Idle indicator
	idleIcon := styles.StatusStyle("idle").Render(icons.Idle)
	parts = append(parts, idleIcon)

	return strings.Join(parts, " ")
}

// =============================================================================
// Progress Bar - Visual progress indicator
// =============================================================================

// ProgressBar renders a visual progress bar
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

// View renders the progress bar
func (p ProgressBar) View(width int, styles Styles) string {
	if !p.Visible {
		return ""
	}

	icons := styles.Icons

	// Calculate progress percentage
	percent := 0.0
	if p.Total > 0 {
		percent = float64(p.Current) / float64(p.Total)
	}

	// Progress bar width (leave room for label)
	barWidth := maxInt(20, width-40)

	// Build the bar
	filledWidth := int(float64(barWidth) * percent)
	emptyWidth := barWidth - filledWidth

	filledStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Accent)
	emptyStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.BorderMuted)

	bar := filledStyle.Render(strings.Repeat("█", filledWidth)) +
		emptyStyle.Render(strings.Repeat("░", emptyWidth))

	// Stage label
	stageStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Text)
	stagePart := ""
	if p.Stage != "" {
		stagePart = stageStyle.Render(icons.ChevronRight + " " + p.Stage)
	}

	// Progress count
	countStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextMuted)
	countPart := ""
	if p.Total > 0 {
		countPart = countStyle.Render("(" + itoa(p.Current) + "/" + itoa(p.Total) + ")")
	}

	// Combine
	return lipgloss.JoinHorizontal(lipgloss.Center,
		stagePart, "  ", bar, "  ", countPart)
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
		{Key: "d", Desc: "dest"},
		{Key: "^K", Desc: "commands"},
		{Key: "?", Desc: "help"},
	}
}

// IssuesFocusedHints returns hints when issues panel is focused
func IssuesFocusedHints() []HintItem {
	return []HintItem{
		{Key: "j/k", Desc: "navigate"},
		{Key: "⏎", Desc: "expand"},
		{Key: "o", Desc: "open"},
		{Key: "tab", Desc: "logs"},
		{Key: "e", Desc: "hide"},
		{Key: "?", Desc: "help"},
	}
}

// NormalWithIssuesHints returns hints for normal mode when issues exist
func NormalWithIssuesHints() []HintItem {
	return []HintItem{
		{Key: "b", Desc: "build"},
		{Key: "r", Desc: "run"},
		{Key: "t", Desc: "test"},
		{Key: "e", Desc: "errors"},
		{Key: "^K", Desc: "commands"},
		{Key: "?", Desc: "help"},
	}
}

// View renders the hints bar
func (h HintsBar) View(width int, styles Styles) string {
	return h.renderHints(h.Hints, styles)
}

// ViewWithContext renders hints based on context
func (h HintsBar) ViewWithContext(width int, styles Styles, issuesFocused bool, hasIssues bool) string {
	var hints []HintItem

	if issuesFocused {
		hints = IssuesFocusedHints()
	} else if hasIssues {
		hints = NormalWithIssuesHints()
	} else {
		hints = DefaultHints()
	}

	return h.renderHints(hints, styles)
}

func (h HintsBar) renderHints(hints []HintItem, styles Styles) string {
	var parts []string

	keyStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Accent)
	descStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextSubtle)
	sepStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.BorderMuted)

	for i, hint := range hints {
		part := keyStyle.Render(hint.Key) + ":" + descStyle.Render(hint.Desc)
		parts = append(parts, part)

		if i < len(hints)-1 {
			parts = append(parts, sepStyle.Render("  "))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}
