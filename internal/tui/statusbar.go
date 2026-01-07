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
	Scheme      string
	Destination string
	DestOS      string

	// Running state
	Running    bool
	RunningCmd string
	Stage      string
	Progress   string

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

	// Idle status
	icon := styles.StatusStyle("idle").Render(icons.Idle)
	labelStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextMuted)

	return icon + " " + labelStyle.Render("idle")
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
		Hints: []HintItem{
			{Key: "Tab", Desc: "focus"},
			{Key: "^B", Desc: "sidebar"},
			{Key: "^K", Desc: "commands"},
			{Key: "?", Desc: "help"},
			{Key: "q", Desc: "quit"},
		},
	}
}

// View renders the hints bar
func (h HintsBar) View(width int, styles Styles) string {
	var parts []string

	keyStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.Accent)
	descStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextSubtle)
	sepStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.BorderMuted)

	for i, hint := range h.Hints {
		part := keyStyle.Render(hint.Key) + ":" + descStyle.Render(hint.Desc)
		parts = append(parts, part)

		if i < len(h.Hints)-1 {
			parts = append(parts, sepStyle.Render("  "))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

// =============================================================================
// Summary Card - Shows operation result briefly
// =============================================================================

// SummaryCard displays a brief result summary
type SummaryCard struct {
	Visible   bool
	Operation string
	Success   bool
	Duration  string
	Message   string
}

// NewSummaryCard creates a new summary card
func NewSummaryCard() SummaryCard {
	return SummaryCard{Visible: false}
}

// Show displays the summary card with the given result
func (c *SummaryCard) Show(operation string, success bool, duration, message string) {
	c.Visible = true
	c.Operation = operation
	c.Success = success
	c.Duration = duration
	c.Message = message
}

// Hide hides the summary card
func (c *SummaryCard) Hide() {
	c.Visible = false
}

// View renders the summary card
func (c SummaryCard) View(styles Styles) string {
	if !c.Visible {
		return ""
	}

	icons := styles.Icons

	// Status icon
	status := "success"
	icon := icons.Success
	if !c.Success {
		status = "error"
		icon = icons.Error
	}

	iconPart := styles.StatusStyle(status).Render(icon)

	// Operation name
	opStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(styles.Colors.Text)
	opPart := opStyle.Render(c.Operation)

	// Duration
	durStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextMuted)
	durPart := ""
	if c.Duration != "" {
		durPart = durStyle.Render(" · " + c.Duration)
	}

	// Message
	msgStyle := lipgloss.NewStyle().
		Foreground(styles.Colors.TextMuted)
	msgPart := ""
	if c.Message != "" {
		msgPart = msgStyle.Render(" · " + c.Message)
	}

	content := lipgloss.JoinHorizontal(lipgloss.Center,
		iconPart, " ", opPart, durPart, msgPart)

	// Card container
	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Colors.Border).
		Padding(0, 2)

	return cardStyle.Render(content)
}
