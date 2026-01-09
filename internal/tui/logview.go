package tui

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Build Phase - Collapsible section representing a build stage
// =============================================================================

// BuildPhase represents a collapsible build phase (Compiling, Linking, etc.)
type BuildPhase struct {
	Name      string      // Phase name: "Compiling", "Linking", etc.
	Lines     []LogLine   // Log lines in this phase
	Collapsed bool        // Whether phase is collapsed
	Status    PhaseStatus // Current status
	FileCount int         // Number of files processed (for display)
	HasError  bool        // Quick flag for error presence
}

// PhaseStatus represents the build phase status
type PhaseStatus int

const (
	PhaseRunning PhaseStatus = iota
	PhaseSuccess
	PhaseWarning
	PhaseError
)

// LogLine represents a single log entry with categorization
type LogLine struct {
	Text     string
	Type     LogLineType
	FilePath string // Source file path for errors/warnings
	LineNum  int    // Line number for errors/warnings
	TestName string // Test name for test results
}

// LogLineType categorizes log lines for styling
type LogLineType int

const (
	LogLineNormal   LogLineType = iota
	LogLineInfo                 // Build step info (▸ arrows)
	LogLineSuccess              // Success message or passed test
	LogLineWarning              // Compiler warning
	LogLineError                // Compiler error or test failure
	LogLineTestPass             // Passed test
	LogLineTestFail             // Failed test
)

// =============================================================================
// Phase View - Main grouped log view component
// =============================================================================

// PhaseView manages grouped log display with collapsible phases
type PhaseView struct {
	Phases    []BuildPhase
	FlatLines []string // Raw lines for fallback/raw mode

	// Scroll state
	ScrollPos       int // Scroll position in visible lines
	VisibleRows     int // Number of visible rows
	AutoFollow      bool
	StickyHeader    bool
	ShowErrorsOnly  bool
	Width           int
	RenderMode      PhaseRenderMode
	LastRenderLines int

	// Selection
	SelectedPhase int  // Currently selected phase (-1 for none)
	PhaseMode     bool // Whether we're in phase selection mode

	// Settings
	SmartCollapse bool // Auto-collapse clean phases, expand errors
	ShowRawMode   bool // Show raw logs instead of grouped

	// Search state
	SearchQuery   string          // Current search query
	SearchMatches []SearchMatch   // Search match positions
	HighlightLine map[string]bool // Lines to highlight (phase:line key)
}

// SearchMatch represents a search result position
type SearchMatch struct {
	Phase int
	Line  int
}

// PhaseRenderMode controls how phases are rendered.
type PhaseRenderMode int

const (
	PhaseRenderGrouped PhaseRenderMode = iota
	PhaseRenderCards
)

// NewPhaseView creates a new phase view
func NewPhaseView() PhaseView {
	return PhaseView{
		Phases:        []BuildPhase{},
		FlatLines:     []string{},
		SmartCollapse: true,
		SelectedPhase: -1,
		AutoFollow:    true,
		StickyHeader:  true,
		RenderMode:    PhaseRenderGrouped,
	}
}

// Clear resets the phase view for a new build
func (v *PhaseView) Clear() {
	v.Phases = []BuildPhase{}
	v.FlatLines = []string{}
	v.ScrollPos = 0
	v.SelectedPhase = -1
	v.AutoFollow = true
	v.LastRenderLines = 0
}

// SetSize sets the visible dimensions
func (v *PhaseView) SetSize(width, height int) {
	v.Width = width
	v.VisibleRows = height
}

// =============================================================================
// Adding Content
// =============================================================================

// AddLine processes and adds a log line
func (v *PhaseView) AddLine(line string) {
	if strings.TrimSpace(line) == "" {
		return
	}

	// Always add to flat lines
	v.FlatLines = append(v.FlatLines, line)
	if len(v.FlatLines) > 2000 {
		v.FlatLines = v.FlatLines[len(v.FlatLines)-2000:]
	}

	// Detect if this is a phase change
	phaseName := detectBuildPhase(line)
	if phaseName != "" {
		v.ensurePhase(phaseName)
	}

	// Add to current phase (or create default)
	logLine := categorizeLogLine(line)
	v.addToCurrentPhase(logLine)

	// Auto-scroll if at bottom
	v.autoScroll()
}

// ensurePhase creates or activates a phase
func (v *PhaseView) ensurePhase(name string) {
	// Check if phase already exists
	for i := range v.Phases {
		if v.Phases[i].Name == name {
			// Mark any running phases as complete first
			for j := range v.Phases {
				if j != i && v.Phases[j].Status == PhaseRunning {
					v.completePhase(j)
				}
			}
			// Re-activate this phase
			v.Phases[i].Status = PhaseRunning
			v.Phases[i].Collapsed = false
			return
		}
	}

	// Mark all running phases as complete
	for i := range v.Phases {
		if v.Phases[i].Status == PhaseRunning {
			v.completePhase(i)
		}
	}

	// Create new phase
	phase := BuildPhase{
		Name:      name,
		Lines:     []LogLine{},
		Collapsed: false,
		Status:    PhaseRunning,
	}
	v.Phases = append(v.Phases, phase)
}

// completePhase marks a phase as complete with smart collapse
func (v *PhaseView) completePhase(idx int) {
	if idx < 0 || idx >= len(v.Phases) {
		return
	}

	phase := &v.Phases[idx]
	if phase.Status == PhaseRunning {
		// Determine final status based on contents
		if phase.HasError {
			phase.Status = PhaseError
		} else {
			hasWarning := false
			for _, line := range phase.Lines {
				if line.Type == LogLineWarning {
					hasWarning = true
					break
				}
			}
			if hasWarning {
				phase.Status = PhaseWarning
			} else {
				phase.Status = PhaseSuccess
			}
		}

		// Smart collapse: collapse clean phases, expand errors
		if v.SmartCollapse {
			phase.Collapsed = phase.Status == PhaseSuccess
		}
	}
}

// addToCurrentPhase adds a line to the current (running) phase
func (v *PhaseView) addToCurrentPhase(line LogLine) {
	// Find running phase
	for i := range v.Phases {
		if v.Phases[i].Status == PhaseRunning {
			v.Phases[i].Lines = append(v.Phases[i].Lines, line)

			// Update phase status based on line type
			if line.Type == LogLineError || line.Type == LogLineTestFail {
				v.Phases[i].HasError = true
				v.Phases[i].Status = PhaseError
				v.Phases[i].Collapsed = false // Always show errors
			}

			// Count files for "Compiling X files" display
			if strings.HasPrefix(line.Text, "Compiling") ||
				strings.HasPrefix(line.Text, "▸ Compiling") {
				v.Phases[i].FileCount++
			}
			return
		}
	}

	// No running phase - create a default "Build" phase
	if len(v.Phases) == 0 {
		v.Phases = append(v.Phases, BuildPhase{
			Name:   "Build",
			Lines:  []LogLine{line},
			Status: PhaseRunning,
		})
	}
}

// MarkBuildComplete marks all phases complete
func (v *PhaseView) MarkBuildComplete(success bool) {
	for i := range v.Phases {
		if v.Phases[i].Status == PhaseRunning {
			v.completePhase(i)
		}
	}
}

// =============================================================================
// Navigation
// =============================================================================

// ScrollUp scrolls the view up
func (v *PhaseView) ScrollUp(lines int) {
	v.ScrollPos -= lines
	if v.ScrollPos < 0 {
		v.ScrollPos = 0
	}
	v.AutoFollow = false
}

// ScrollDown scrolls the view down
func (v *PhaseView) ScrollDown(lines int) {
	maxScroll := v.totalLines() - v.visibleRowsForContent()
	if maxScroll < 0 {
		maxScroll = 0
	}
	v.ScrollPos += lines
	if v.ScrollPos > maxScroll {
		v.ScrollPos = maxScroll
	}
	v.AutoFollow = v.ScrollPos == maxScroll
}

// GotoTop scrolls to the top
func (v *PhaseView) GotoTop() {
	v.ScrollPos = 0
	v.AutoFollow = false
}

// GotoBottom scrolls to the bottom
func (v *PhaseView) GotoBottom() {
	maxScroll := v.totalLines() - v.visibleRowsForContent()
	if maxScroll < 0 {
		maxScroll = 0
	}
	v.ScrollPos = maxScroll
	v.AutoFollow = true
}

// autoScroll scrolls to bottom if near the end
func (v *PhaseView) autoScroll() {
	if !v.AutoFollow {
		return
	}
	total := v.totalLines()
	if total <= v.visibleRowsForContent() {
		v.ScrollPos = 0
		return
	}

	maxScroll := total - v.visibleRowsForContent()
	// Auto-follow if within 5 lines of bottom
	if v.ScrollPos >= maxScroll-5 {
		v.ScrollPos = maxScroll
	}
}

// totalLines returns the total line count in current view mode
func (v *PhaseView) totalLines() int {
	if v.ShowRawMode {
		return len(v.FlatLines)
	}
	if v.RenderMode == PhaseRenderCards && v.LastRenderLines > 0 {
		return v.LastRenderLines
	}

	count := 0
	for _, phase := range v.Phases {
		if v.ShowErrorsOnly {
			matchCount := v.matchingLineCount(phase)
			if matchCount == 0 {
				continue
			}
			count++ // Phase header
			count += matchCount
			continue
		}

		count++ // Phase header
		if !phase.Collapsed {
			count += len(phase.Lines)
		}
	}
	return count
}

func (v *PhaseView) visibleRowsForContent() int {
	rows := v.VisibleRows
	if rows <= 0 {
		return 0
	}
	if v.RenderMode == PhaseRenderCards {
		return rows
	}
	if v.StickyHeader && !v.ShowRawMode && rows > 1 && len(v.Phases) > 0 {
		return rows - 1
	}
	return rows
}

func (v *PhaseView) matchingLineCount(phase BuildPhase) int {
	count := 0
	for _, line := range phase.Lines {
		if line.Type == LogLineError || line.Type == LogLineWarning || line.Type == LogLineTestFail {
			count++
		}
	}
	return count
}

// SelectNextPhase moves to next phase
func (v *PhaseView) SelectNextPhase() {
	if len(v.Phases) == 0 {
		return
	}
	v.PhaseMode = true
	v.SelectedPhase++
	if v.SelectedPhase >= len(v.Phases) {
		v.SelectedPhase = 0
	}
}

// SelectPrevPhase moves to previous phase
func (v *PhaseView) SelectPrevPhase() {
	if len(v.Phases) == 0 {
		return
	}
	v.PhaseMode = true
	v.SelectedPhase--
	if v.SelectedPhase < 0 {
		v.SelectedPhase = len(v.Phases) - 1
	}
}

// ToggleSelectedPhase toggles collapse on selected phase
func (v *PhaseView) ToggleSelectedPhase() {
	if v.SelectedPhase >= 0 && v.SelectedPhase < len(v.Phases) {
		v.Phases[v.SelectedPhase].Collapsed = !v.Phases[v.SelectedPhase].Collapsed
	}
}

// ExpandAll expands all phases
func (v *PhaseView) ExpandAll() {
	for i := range v.Phases {
		v.Phases[i].Collapsed = false
	}
}

// CollapseAll collapses all phases
func (v *PhaseView) CollapseAll() {
	for i := range v.Phases {
		v.Phases[i].Collapsed = true
	}
}

// ToggleRawMode toggles between grouped and raw log view
func (v *PhaseView) ToggleRawMode() {
	v.ShowRawMode = !v.ShowRawMode
}

// =============================================================================
// Error/Match Navigation
// =============================================================================

// FindNextError finds the next error line and returns its phase and line index
func (v *PhaseView) FindNextError(currentPhase, currentLine int) (int, int) {
	// Start searching from current position
	for p := currentPhase; p < len(v.Phases); p++ {
		startLine := 0
		if p == currentPhase {
			startLine = currentLine + 1
		}
		for l := startLine; l < len(v.Phases[p].Lines); l++ {
			if v.Phases[p].Lines[l].Type == LogLineError ||
				v.Phases[p].Lines[l].Type == LogLineTestFail {
				// Expand this phase
				v.Phases[p].Collapsed = false
				return p, l
			}
		}
	}

	// Wrap around to beginning
	for p := 0; p <= currentPhase; p++ {
		endLine := len(v.Phases[p].Lines)
		if p == currentPhase {
			endLine = currentLine
		}
		for l := 0; l < endLine; l++ {
			if v.Phases[p].Lines[l].Type == LogLineError ||
				v.Phases[p].Lines[l].Type == LogLineTestFail {
				v.Phases[p].Collapsed = false
				return p, l
			}
		}
	}

	return -1, -1
}

// FindPrevError finds the previous error line
func (v *PhaseView) FindPrevError(currentPhase, currentLine int) (int, int) {
	// Search backwards from current position
	for p := currentPhase; p >= 0; p-- {
		endLine := len(v.Phases[p].Lines) - 1
		if p == currentPhase {
			endLine = currentLine - 1
		}
		for l := endLine; l >= 0; l-- {
			if v.Phases[p].Lines[l].Type == LogLineError ||
				v.Phases[p].Lines[l].Type == LogLineTestFail {
				v.Phases[p].Collapsed = false
				return p, l
			}
		}
	}

	// Wrap around to end
	for p := len(v.Phases) - 1; p >= currentPhase; p-- {
		startLine := 0
		if p == currentPhase {
			startLine = currentLine + 1
		}
		for l := len(v.Phases[p].Lines) - 1; l >= startLine; l-- {
			if v.Phases[p].Lines[l].Type == LogLineError ||
				v.Phases[p].Lines[l].Type == LogLineTestFail {
				v.Phases[p].Collapsed = false
				return p, l
			}
		}
	}

	return -1, -1
}

// ErrorCount returns total error count
func (v *PhaseView) ErrorCount() int {
	count := 0
	for _, phase := range v.Phases {
		for _, line := range phase.Lines {
			if line.Type == LogLineError || line.Type == LogLineTestFail {
				count++
			}
		}
	}
	return count
}

// WarningCount returns total warning count
func (v *PhaseView) WarningCount() int {
	count := 0
	for _, phase := range v.Phases {
		for _, line := range phase.Lines {
			if line.Type == LogLineWarning {
				count++
			}
		}
	}
	return count
}

// =============================================================================
// Search
// =============================================================================

// Search performs a search and returns matches
func (v *PhaseView) Search(query string) []SearchMatch {
	v.SearchQuery = query
	v.SearchMatches = nil
	v.HighlightLine = make(map[string]bool)

	if query == "" {
		return nil
	}

	lowerQuery := strings.ToLower(query)

	for p, phase := range v.Phases {
		for l, line := range phase.Lines {
			if strings.Contains(strings.ToLower(line.Text), lowerQuery) {
				match := SearchMatch{Phase: p, Line: l}
				v.SearchMatches = append(v.SearchMatches, match)
				v.HighlightLine[fmt.Sprintf("%d:%d", p, l)] = true
			}
		}
	}

	return v.SearchMatches
}

// ClearSearch clears the search state
func (v *PhaseView) ClearSearch() {
	v.SearchQuery = ""
	v.SearchMatches = nil
	v.HighlightLine = nil
}

// JumpToMatch scrolls to show a specific match
func (v *PhaseView) JumpToMatch(phase, line int) {
	if phase < 0 || phase >= len(v.Phases) {
		return
	}

	// Expand the phase containing the match
	v.Phases[phase].Collapsed = false

	// Calculate the line position in the rendered view
	linePos := 0
	for p := 0; p < phase; p++ {
		linePos++ // Phase header
		if !v.Phases[p].Collapsed {
			linePos += len(v.Phases[p].Lines)
		}
	}
	linePos++ // Current phase header
	linePos += line

	// Scroll to show the match (centered if possible)
	rows := v.visibleRowsForContent()
	targetPos := linePos - rows/2
	if targetPos < 0 {
		targetPos = 0
	}
	maxScroll := v.totalLines() - rows
	if maxScroll < 0 {
		maxScroll = 0
	}
	if targetPos > maxScroll {
		targetPos = maxScroll
	}
	v.ScrollPos = targetPos
}

// IsHighlighted returns whether a specific line should be highlighted
func (v *PhaseView) IsHighlighted(phase, line int) bool {
	if v.HighlightLine == nil {
		return false
	}
	return v.HighlightLine[fmt.Sprintf("%d:%d", phase, line)]
}

// =============================================================================
// Rendering
// =============================================================================

// View renders the log view
func (v *PhaseView) View(styles Styles) string {
	if v.ShowRawMode {
		return v.renderRaw(styles)
	}

	if len(v.Phases) == 0 {
		return v.renderRaw(styles)
	}

	if v.RenderMode == PhaseRenderCards {
		return v.renderCards(styles)
	}

	return v.renderGrouped(styles)
}

// renderRaw renders flat log lines
func (v PhaseView) renderRaw(styles Styles) string {
	if len(v.FlatLines) == 0 {
		return ""
	}

	contentRows := v.visibleRowsForContent()
	if contentRows <= 0 {
		return ""
	}

	// Apply scroll
	startIdx := v.ScrollPos
	endIdx := startIdx + contentRows
	if startIdx >= len(v.FlatLines) {
		startIdx = len(v.FlatLines) - 1
		if startIdx < 0 {
			startIdx = 0
		}
	}
	if endIdx > len(v.FlatLines) {
		endIdx = len(v.FlatLines)
	}

	var lines []string
	for i := startIdx; i < endIdx; i++ {
		lines = append(lines, v.FlatLines[i])
	}

	return strings.Join(lines, "\n")
}

// renderGrouped renders grouped phases
func (v PhaseView) renderGrouped(styles Styles) string {
	icons := styles.Icons
	var allLines []string
	var linePhases []int
	headerIndex := map[int]int{}

	for i, phase := range v.Phases {
		isSelected := v.PhaseMode && i == v.SelectedPhase

		if v.ShowErrorsOnly && v.matchingLineCount(phase) == 0 {
			continue
		}

		// Render phase header
		header := v.renderPhaseHeader(phase, isSelected, styles, icons)
		allLines = append(allLines, header)
		linePhases = append(linePhases, i)
		headerIndex[i] = len(allLines) - 1

		// Render phase content if expanded
		for j, line := range phase.Lines {
			if v.ShowErrorsOnly &&
				line.Type != LogLineError &&
				line.Type != LogLineWarning &&
				line.Type != LogLineTestFail {
				continue
			}
			if !v.ShowErrorsOnly && phase.Collapsed {
				continue
			}
			highlighted := v.IsHighlighted(i, j)
			rendered := v.renderLogLine(line, highlighted, styles, icons)
			allLines = append(allLines, "  "+rendered)
			linePhases = append(linePhases, i)
		}
	}

	if len(allLines) == 0 {
		return ""
	}

	contentRows := v.visibleRowsForContent()
	if contentRows <= 0 {
		return ""
	}

	// Apply scroll
	startIdx := v.ScrollPos
	if startIdx >= len(allLines) {
		startIdx = len(allLines) - 1
		if startIdx < 0 {
			startIdx = 0
		}
	}

	stickyHeader := ""
	if v.StickyHeader && len(linePhases) > 0 {
		lineIdx := startIdx
		if lineIdx >= len(linePhases) {
			lineIdx = len(linePhases) - 1
		}
		phaseIdx := linePhases[lineIdx]
		if phaseIdx >= 0 {
			if headerLine, ok := headerIndex[phaseIdx]; ok && headerLine >= 0 && headerLine < len(allLines) {
				stickyHeader = allLines[headerLine]
				if startIdx == headerLine {
					startIdx++
				}
			}
		}
	}

	endIdx := startIdx + contentRows
	if startIdx >= len(allLines) {
		startIdx = len(allLines) - 1
		if startIdx < 0 {
			startIdx = 0
		}
	}
	if endIdx > len(allLines) {
		endIdx = len(allLines)
	}

	visibleLines := allLines[startIdx:endIdx]
	if stickyHeader != "" {
		visibleLines = append([]string{stickyHeader}, visibleLines...)
	}

	return strings.Join(visibleLines, "\n")
}

// renderCards renders phase cards with borders and summaries.
func (v *PhaseView) renderCards(styles Styles) string {
	icons := styles.Icons
	var allLines []string

	for i, phase := range v.Phases {
		if v.ShowErrorsOnly && v.matchingLineCount(phase) == 0 {
			continue
		}

		card := v.renderPhaseCard(i, phase, styles, icons, v.Width)
		if card == "" {
			continue
		}
		allLines = append(allLines, strings.Split(card, "\n")...)
		allLines = append(allLines, "")
	}

	if len(allLines) == 0 {
		v.LastRenderLines = 0
		return ""
	}

	// Drop trailing spacer line.
	if len(allLines) > 0 && allLines[len(allLines)-1] == "" {
		allLines = allLines[:len(allLines)-1]
	}

	v.LastRenderLines = len(allLines)

	startIdx := v.ScrollPos
	endIdx := startIdx + v.VisibleRows
	if startIdx >= len(allLines) {
		startIdx = len(allLines) - 1
		if startIdx < 0 {
			startIdx = 0
		}
	}
	if endIdx > len(allLines) {
		endIdx = len(allLines)
	}

	return strings.Join(allLines[startIdx:endIdx], "\n")
}

func (v PhaseView) renderPhaseCard(phaseIndex int, phase BuildPhase, styles Styles, icons Icons, width int) string {
	linesCount, errCount, warnCount := phaseCounts(phase)
	if v.ShowErrorsOnly && errCount == 0 && warnCount == 0 {
		return ""
	}

	statusIcon, statusStyle := phaseStatusStyle(phase, styles, icons)
	nameStyle := lipgloss.NewStyle().Foreground(styles.Colors.Text).Bold(true)

	headerParts := []string{
		statusStyle.Render(statusIcon),
		nameStyle.Render(phase.Name),
	}

	if linesCount > 0 {
		metaStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)
		headerParts = append(headerParts, metaStyle.Render(fmt.Sprintf("%d lines", linesCount)))
	}
	if errCount > 0 {
		headerParts = append(headerParts, styles.StatusStyle("error").Render(fmt.Sprintf("%s %d", icons.Error, errCount)))
	}
	if warnCount > 0 {
		headerParts = append(headerParts, styles.StatusStyle("warning").Render(fmt.Sprintf("%s %d", icons.Warning, warnCount)))
	}

	var body []string
	if v.ShowErrorsOnly {
		for i, line := range phase.Lines {
			if line.Type != LogLineError && line.Type != LogLineWarning && line.Type != LogLineTestFail {
				continue
			}
			body = append(body, "  "+v.renderLogLine(line, v.IsHighlighted(phaseIndex, i), styles, icons))
		}
	} else if phase.Collapsed {
		muted := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)
		body = append(body, muted.Render(fmt.Sprintf("%d lines hidden", linesCount)))
	} else {
		for i, line := range phase.Lines {
			body = append(body, "  "+v.renderLogLine(line, v.IsHighlighted(phaseIndex, i), styles, icons))
		}
	}

	content := strings.Join(append([]string{strings.Join(headerParts, " ")}, body...), "\n")

	cardStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.Colors.Border).
		Padding(0, 1)
	if width > 0 {
		cardStyle = cardStyle.Width(width)
	}

	return cardStyle.Render(content)
}

func phaseStatusStyle(phase BuildPhase, styles Styles, icons Icons) (string, lipgloss.Style) {
	switch phase.Status {
	case PhaseRunning:
		return icons.Running, styles.StatusStyle("running")
	case PhaseSuccess:
		return icons.Success, styles.StatusStyle("success")
	case PhaseWarning:
		return icons.Warning, styles.StatusStyle("warning")
	case PhaseError:
		return icons.Error, styles.StatusStyle("error")
	default:
		return icons.Idle, lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)
	}
}

func phaseCounts(phase BuildPhase) (lines, errors, warnings int) {
	for _, line := range phase.Lines {
		switch line.Type {
		case LogLineError, LogLineTestFail:
			errors++
		case LogLineWarning:
			warnings++
		}
	}
	return len(phase.Lines), errors, warnings
}

// renderPhaseHeader renders a phase header line
func (v PhaseView) renderPhaseHeader(phase BuildPhase, selected bool, styles Styles, icons Icons) string {
	// Collapse indicator
	collapseIcon := icons.ChevronDown
	if phase.Collapsed {
		collapseIcon = icons.ChevronRight
	}
	collapseStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)

	// Status icon
	var statusIcon string
	var statusStyle lipgloss.Style
	switch phase.Status {
	case PhaseRunning:
		statusIcon = icons.Running
		statusStyle = styles.StatusStyle("running")
	case PhaseSuccess:
		statusIcon = icons.Success
		statusStyle = styles.StatusStyle("success")
	case PhaseWarning:
		statusIcon = icons.Warning
		statusStyle = styles.StatusStyle("warning")
	case PhaseError:
		statusIcon = icons.Error
		statusStyle = styles.StatusStyle("error")
	}

	// Phase name
	nameStyle := lipgloss.NewStyle().Foreground(styles.Colors.Text)
	if selected {
		nameStyle = nameStyle.Bold(true).Foreground(styles.Colors.Accent)
	}

	// Count/summary
	countStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)
	var countText string
	if phase.Collapsed {
		// Show summary when collapsed
		lineCount := len(phase.Lines)
		if lineCount == 1 {
			countText = "(1 line)"
		} else {
			countText = fmt.Sprintf("(%d lines)", lineCount)
		}
	}

	parts := []string{
		collapseStyle.Render(collapseIcon),
		statusStyle.Render(statusIcon),
		nameStyle.Render(phase.Name),
	}
	if countText != "" {
		parts = append(parts, countStyle.Render(countText))
	}

	return strings.Join(parts, " ")
}

// renderLogLine renders a single log line with appropriate styling
func (v PhaseView) renderLogLine(line LogLine, highlighted bool, styles Styles, icons Icons) string {
	var prefix string
	lineStyle := lipgloss.NewStyle().Foreground(styles.Colors.Text)

	switch line.Type {
	case LogLineError:
		prefix = styles.StatusStyle("error").Render(icons.Error) + " "
		lineStyle = lineStyle.Foreground(styles.Colors.Error)
	case LogLineWarning:
		prefix = styles.StatusStyle("warning").Render(icons.Warning) + " "
		lineStyle = lineStyle.Foreground(styles.Colors.Warning)
	case LogLineSuccess, LogLineTestPass:
		prefix = styles.StatusStyle("success").Render(icons.Success) + " "
		lineStyle = lineStyle.Foreground(styles.Colors.Success)
	case LogLineTestFail:
		prefix = styles.StatusStyle("error").Render(icons.Error) + " "
		lineStyle = lineStyle.Foreground(styles.Colors.Error)
	case LogLineInfo:
		prefix = lipgloss.NewStyle().Foreground(styles.Colors.TextMuted).Render("  ")
	default:
		prefix = "  "
		lineStyle = lineStyle.Foreground(styles.Colors.TextMuted)
	}

	// Apply search highlight
	if highlighted {
		lineStyle = lineStyle.Background(styles.Colors.Accent).Foreground(styles.Colors.Background)
	}

	return prefix + lineStyle.Render(line.Text)
}

// =============================================================================
// Phase Detection - Comprehensive xcodebuild pattern matching
// =============================================================================

// Phase patterns for xcodebuild output
var phasePatterns = []struct {
	pattern *regexp.Regexp
	name    string
}{
	// xcpretty/xcbeautify formatted output
	{regexp.MustCompile(`^▸ Compiling`), "Compiling"},
	{regexp.MustCompile(`^▸ Linking`), "Linking"},
	{regexp.MustCompile(`^▸ Signing`), "Signing"},
	{regexp.MustCompile(`^▸ Processing`), "Processing"},
	{regexp.MustCompile(`^▸ Copying`), "Copying"},
	{regexp.MustCompile(`^▸ Building`), "Building"},
	{regexp.MustCompile(`^▸ Running`), "Running"},
	{regexp.MustCompile(`^▸ Testing`), "Testing"},
	{regexp.MustCompile(`^▸ Analyzing`), "Analyzing"},

	// Raw xcodebuild output
	{regexp.MustCompile(`^CompileSwiftSources`), "Compiling Swift"},
	{regexp.MustCompile(`^CompileC `), "Compiling C/ObjC"},
	{regexp.MustCompile(`^Ld `), "Linking"},
	{regexp.MustCompile(`^CodeSign `), "Signing"},
	{regexp.MustCompile(`^ProcessInfoPlistFile`), "Processing Info.plist"},
	{regexp.MustCompile(`^CpResource`), "Copying Resources"},
	{regexp.MustCompile(`^PBXCp`), "Copying"},
	{regexp.MustCompile(`^PhaseScriptExecution`), "Running Script"},
	{regexp.MustCompile(`^CreateUniversalBinary`), "Creating Universal Binary"},
	{regexp.MustCompile(`^Touch `), "Touching"},
	{regexp.MustCompile(`^RegisterExecutionPolicyException`), "Registering"},

	// Build phases
	{regexp.MustCompile(`(?i)^=== BUILD TARGET`), "Building Target"},
	{regexp.MustCompile(`(?i)^=== TEST TARGET`), "Testing Target"},
	{regexp.MustCompile(`(?i)^=== ANALYZE TARGET`), "Analyzing Target"},

	// Test phases
	{regexp.MustCompile(`^Test Suite '.*' started`), "Running Tests"},
	{regexp.MustCompile(`^Test Case '.*' started`), "Running Tests"},

	// Archive/Export
	{regexp.MustCompile(`^Archive`), "Archiving"},
	{regexp.MustCompile(`^Export`), "Exporting"},
}

// detectBuildPhase detects the build phase from a log line
func detectBuildPhase(line string) string {
	for _, p := range phasePatterns {
		if p.pattern.MatchString(line) {
			return p.name
		}
	}

	// Fallback: simple string matching for common patterns
	lower := strings.ToLower(line)
	switch {
	case strings.Contains(lower, "compiling"):
		return "Compiling"
	case strings.Contains(lower, "linking"):
		return "Linking"
	case strings.Contains(lower, "signing"):
		return "Signing"
	case strings.Contains(lower, "testing"):
		return "Testing"
	case strings.Contains(lower, "analyzing"):
		return "Analyzing"
	}

	return ""
}

// categorizeLogLine categorizes a log line by type
func categorizeLogLine(line string) LogLine {
	logLine := LogLine{Text: line, Type: LogLineNormal}
	lower := strings.ToLower(line)

	// Test results
	if strings.Contains(line, "Test Case") || strings.Contains(line, "test case") {
		if strings.Contains(lower, "passed") {
			logLine.Type = LogLineTestPass
			logLine.TestName = extractTestName(line)
			return logLine
		}
		if strings.Contains(lower, "failed") {
			logLine.Type = LogLineTestFail
			logLine.TestName = extractTestName(line)
			return logLine
		}
	}

	// Errors/warnings/notes (shared heuristics)
	switch issueSeverity(line) {
	case TabLineTypeError:
		logLine.Type = LogLineError
		logLine.FilePath, logLine.LineNum = extractFileLocation(line)
		return logLine
	case TabLineTypeWarning:
		logLine.Type = LogLineWarning
		logLine.FilePath, logLine.LineNum = extractFileLocation(line)
		return logLine
	case TabLineTypeNote:
		logLine.Type = LogLineInfo
		return logLine
	}

	// Success indicators
	if strings.Contains(lower, "succeeded") ||
		strings.Contains(line, "✓") ||
		strings.Contains(line, "✔") ||
		strings.Contains(lower, "build succeeded") {
		logLine.Type = LogLineSuccess
		return logLine
	}

	// Info lines (build step output)
	if strings.HasPrefix(line, "▸") ||
		strings.HasPrefix(line, "→") ||
		strings.HasPrefix(line, "•") {
		logLine.Type = LogLineInfo
		return logLine
	}

	return logLine
}

// extractTestName extracts test name from test output line
func extractTestName(line string) string {
	// Pattern: Test Case '-[ClassName testMethod]' started/passed/failed
	start := strings.Index(line, "'")
	end := strings.LastIndex(line, "'")
	if start != -1 && end != -1 && end > start {
		return line[start+1 : end]
	}
	return ""
}

// extractFileLocation tries to extract file:line from error/warning
func extractFileLocation(line string) (string, int) {
	// Common pattern: /path/to/file.swift:42:10: error: message
	// Also: /path/to/file.swift:42: error: message
	parts := strings.Split(line, ":")
	if len(parts) >= 3 {
		// Find path-like part followed by numbers
		for i := 0; i < len(parts)-1; i++ {
			part := strings.TrimSpace(parts[i])
			if strings.HasPrefix(part, "/") ||
				strings.HasSuffix(part, ".swift") ||
				strings.HasSuffix(part, ".m") ||
				strings.HasSuffix(part, ".mm") ||
				strings.HasSuffix(part, ".c") ||
				strings.HasSuffix(part, ".cpp") ||
				strings.HasSuffix(part, ".h") {
				var lineNum int
				if i+1 < len(parts) {
					fmt.Sscanf(strings.TrimSpace(parts[i+1]), "%d", &lineNum)
				}
				return part, lineNum
			}
		}
	}
	return "", 0
}
