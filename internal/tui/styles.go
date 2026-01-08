package tui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Color Palette - Monochrome + Blue Accent
// =============================================================================

// Colors defines the complete color palette with light/dark mode support
type Colors struct {
	// Primary accent color - Blue
	Accent      lipgloss.AdaptiveColor
	AccentMuted lipgloss.AdaptiveColor

	// Semantic colors
	Success lipgloss.AdaptiveColor
	Warning lipgloss.AdaptiveColor
	Error   lipgloss.AdaptiveColor
	Running lipgloss.AdaptiveColor

	// Background colors
	Background lipgloss.AdaptiveColor
	Surface    lipgloss.AdaptiveColor
	Overlay    lipgloss.AdaptiveColor

	// Text colors - Monochrome grays
	Text       lipgloss.AdaptiveColor
	TextMuted  lipgloss.AdaptiveColor
	TextSubtle lipgloss.AdaptiveColor

	// Border colors
	Border      lipgloss.AdaptiveColor
	BorderMuted lipgloss.AdaptiveColor
}

// DefaultColors returns the pastel color palette (auto dark/light)
func DefaultColors() Colors {
	return PastelColors()
}

// =============================================================================
// Icons - Nerd Font with Unicode Fallback
// =============================================================================

// Icons holds all icon glyphs with fallback support
type Icons struct {
	// Status icons
	Success string
	Error   string
	Warning string
	Running string
	Idle    string
	Paused  string

	// Action icons
	Build   string
	Run     string
	Test    string
	Clean   string
	Stop    string
	Archive string

	// Navigation icons
	ChevronDown  string
	ChevronRight string
	ArrowRight   string
	Collapsed    string
	Expanded     string

	// UI icons
	Search   string
	Settings string
	Help     string
	Command  string

	// Git
	Branch string

	// Misc
	Dot       string
	Check     string
	Cross     string
	Spinner   string
	Separator string

	// Log phases
	Compile string
	Link    string
	Sign    string
	Copy    string

	// Tab icons (NEW)
	TabStream  string
	TabIssues  string
	TabSummary string

	// Status bar icons (NEW)
	Project   string
	Scheme    string
	Device    string
	Simulator string
	Clock     string

	// Build phase icons (NEW)
	Swift   string
	Process string
	Script  string
	Package string

	// Action icons (NEW)
	Expand   string
	Collapse string
	Export   string
	Filter   string
	Clear    string
	Xcode    string
	Editor   string
	Quit     string

	// Log line icons (NEW)
	File      string
	SwiftFile string
	ObjCFile  string
	LineNum   string
	Info      string
	Note      string
}

// NerdFontIcons returns icons using Nerd Font glyphs
func NerdFontIcons() Icons {
	return Icons{
		// Status
		Success: "\uf00c", //  (nf-fa-check)
		Error:   "\uf00d", //  (nf-fa-times)
		Warning: "\uf071", //  (nf-fa-exclamation_triangle)
		Running: "\uf110", //  (nf-fa-spinner) - use with animation
		Idle:    "\uf111", //  (nf-fa-circle)
		Paused:  "\uf28b", //  (nf-fa-pause_circle)

		// Actions
		Build:   "\uf0ad", //  (nf-fa-wrench)
		Run:     "\uf04b", //  (nf-fa-play)
		Test:    "\uf0c3", //  (nf-fa-flask)
		Clean:   "\uf1f8", //  (nf-fa-trash)
		Stop:    "\uf04d", //  (nf-fa-stop)
		Archive: "\uf187", //  (nf-fa-archive)

		// Navigation
		ChevronDown:  "\uf078", //  (nf-fa-chevron_down)
		ChevronRight: "\uf054", //  (nf-fa-chevron_right)
		ArrowRight:   "\uf061", //  (nf-fa-arrow_right)
		Collapsed:    "\uf054", //  (nf-fa-chevron_right)
		Expanded:     "\uf078", //  (nf-fa-chevron_down)

		// UI
		Search:   "\uf002", //  (nf-fa-search)
		Settings: "\uf013", //  (nf-fa-cog)
		Help:     "\uf059", //  (nf-fa-question_circle)
		Command:  "\uf120", //  (nf-fa-terminal)

		// Git
		Branch: "\ue725", //  (nf-dev-git_branch)

		// Misc
		Dot:       "\uf111", //
		Check:     "\uf00c", //
		Cross:     "\uf00d", //
		Spinner:   "⠋",      // Braille spinner frame
		Separator: "│",

		// Log phases
		Compile: "\uf121", //  (nf-fa-code)
		Link:    "\uf0c1", //  (nf-fa-link)
		Sign:    "\uf023", //  (nf-fa-lock)
		Copy:    "\uf0c5", //  (nf-fa-copy)

		// Tab icons
		TabStream:  "\uf1de", //  (nf-fa-sliders)
		TabIssues:  "\uf188", //  (nf-fa-bug)
		TabSummary: "\uf46d", //  (nf-oct-graph)

		// Status bar icons
		Project:   "\uf07b", //  (nf-fa-folder)
		Scheme:    "\uf013", //  (nf-fa-cog)
		Device:    "\uf10a", //  (nf-fa-mobile)
		Simulator: "\uf3fa", //  (nf-fa-mobile_alt)
		Clock:     "\uf017", //  (nf-fa-clock)

		// Build phase icons
		Swift:   "\ue755", //  (nf-seti-swift)
		Process: "\uf085", //  (nf-fa-gears)
		Script:  "\uf489", //  (nf-oct-terminal)
		Package: "\uf1c6", //  (nf-fa-file_archive)

		// Action icons
		Expand:   "\uf065", //  (nf-fa-expand)
		Collapse: "\uf066", //  (nf-fa-compress)
		Export:   "\uf56e", //  (nf-fa-file_export)
		Filter:   "\uf0b0", //  (nf-fa-filter)
		Clear:    "\uf1f8", //  (nf-fa-trash)
		Xcode:    "\ue711", //  (nf-dev-apple)
		Editor:   "\ue7c5", //  (nf-dev-vim)
		Quit:     "\uf011", //  (nf-fa-power_off)

		// Log line icons
		File:      "\uf15b", //  (nf-fa-file)
		SwiftFile: "\ue755", //  (nf-seti-swift)
		ObjCFile:  "\ue61e", //  (nf-seti-c)
		LineNum:   "\uf292", //  (nf-fa-hashtag)
		Info:      "\uf05a", //  (nf-fa-info_circle)
		Note:      "\uf249", //  (nf-fa-sticky_note)
	}
}

// UnicodeIcons returns icons using standard Unicode (fallback)
func UnicodeIcons() Icons {
	return Icons{
		// Status
		Success: "✓",
		Error:   "✗",
		Warning: "⚠",
		Running: "●",
		Idle:    "○",
		Paused:  "◉",

		// Actions
		Build:   "⚒",
		Run:     "▶",
		Test:    "⚗",
		Clean:   "⌫",
		Stop:    "■",
		Archive: "▣",

		// Navigation
		ChevronDown:  "▾",
		ChevronRight: "▸",
		ArrowRight:   "→",
		Collapsed:    "▸",
		Expanded:     "▾",

		// UI
		Search:   "⌕",
		Settings: "⚙",
		Help:     "?",
		Command:  "⌘",

		// Git
		Branch: "",

		// Misc
		Dot:       "●",
		Check:     "✓",
		Cross:     "✗",
		Spinner:   "◐",
		Separator: "│",

		// Log phases
		Compile: "⟨⟩",
		Link:    "⛓",
		Sign:    "🔒",
		Copy:    "⧉",

		// Tab icons
		TabStream:  "≡",
		TabIssues:  "!",
		TabSummary: "◈",

		// Status bar icons
		Project:   "◫",
		Scheme:    "⚙",
		Device:    "◧",
		Simulator: "◧",
		Clock:     "◔",

		// Build phase icons
		Swift:   "◇",
		Process: "⚙",
		Script:  "▷",
		Package: "◫",

		// Action icons
		Expand:   "⊞",
		Collapse: "⊟",
		Export:   "↧",
		Filter:   "◇",
		Clear:    "⌫",
		Xcode:    "◈",
		Editor:   "◪",
		Quit:     "◉",

		// Log line icons
		File:      "◫",
		SwiftFile: "◇",
		ObjCFile:  "◆",
		LineNum:   "#",
		Info:      "ℹ",
		Note:      "◫",
	}
}

// DetectNerdFont checks if Nerd Font is likely available
func DetectNerdFont() bool {
	// Check common Nerd Font environment indicators
	term := os.Getenv("TERM")
	termProgram := os.Getenv("TERM_PROGRAM")

	// Common terminals that often have Nerd Fonts
	nerdFontTerminals := []string{
		"iTerm",
		"WezTerm",
		"Alacritty",
		"kitty",
		"Hyper",
		"Ghostty",
	}

	for _, t := range nerdFontTerminals {
		if strings.Contains(termProgram, t) || strings.Contains(term, t) {
			return true
		}
	}

	// Check for explicit user preference
	if os.Getenv("XCBOLT_NERD_FONT") == "1" {
		return true
	}
	if os.Getenv("XCBOLT_NERD_FONT") == "0" {
		return false
	}

	// Default to Nerd Font - most modern terminals support it
	return true
}

// GetIcons returns the appropriate icon set based on environment
func GetIcons() Icons {
	if DetectNerdFont() {
		return NerdFontIcons()
	}
	return UnicodeIcons()
}

// =============================================================================
// Component Styles
// =============================================================================

// Styles holds all component styles for the TUI
type Styles struct {
	Colors Colors
	Icons  Icons

	// Layout components
	StatusBar StatusBarStyles
	Logs      LogsStyles
	HintsBar  HintsBarStyles
	Popup     PopupStyles
	Selector  SelectorStyles
	Help      HelpStyles
	Search    SearchStyles
	TabBar    TabBarStyles
	Syntax    SyntaxColors
}

// StatusBarStyles for the top status bar
type StatusBarStyles struct {
	Container   lipgloss.Style
	Section     lipgloss.Style
	Label       lipgloss.Style
	Value       lipgloss.Style
	ValueMuted  lipgloss.Style
	Separator   lipgloss.Style
	StatusIcon  lipgloss.Style
	SpinnerText lipgloss.Style
}

// LogsStyles for the scrollable log viewport
type LogsStyles struct {
	Container lipgloss.Style

	// Phase groups
	PhaseHeader          lipgloss.Style
	PhaseHeaderCollapsed lipgloss.Style
	PhaseIcon            lipgloss.Style
	PhaseCount           lipgloss.Style

	// Log lines
	Line       lipgloss.Style
	LineError  lipgloss.Style
	LineWarn   lipgloss.Style
	LineNumber lipgloss.Style

	// Inline errors/warnings
	ErrorIcon   lipgloss.Style
	ErrorText   lipgloss.Style
	WarningIcon lipgloss.Style
	WarningText lipgloss.Style

	// Test results
	TestPass lipgloss.Style
	TestFail lipgloss.Style
	TestSkip lipgloss.Style

	// Search highlighting
	SearchMatch lipgloss.Style

	// Empty state
	EmptyState lipgloss.Style
}

// HintsBarStyles for the bottom hints bar
type HintsBarStyles struct {
	Container lipgloss.Style
	Key       lipgloss.Style
	Desc      lipgloss.Style
	Separator lipgloss.Style
}

// PopupStyles for centered modal popups
type PopupStyles struct {
	Container lipgloss.Style
	Title     lipgloss.Style
	Content   lipgloss.Style
}

// SelectorStyles for fuzzy search selectors
type SelectorStyles struct {
	Container    lipgloss.Style
	Title        lipgloss.Style
	Input        lipgloss.Style
	InputCursor  lipgloss.Style
	Item         lipgloss.Style
	ItemSelected lipgloss.Style
	ItemMeta     lipgloss.Style
	Divider      lipgloss.Style
	Hint         lipgloss.Style
}

// HelpStyles for help overlay
type HelpStyles struct {
	Container lipgloss.Style
	Title     lipgloss.Style
	Group     lipgloss.Style
	GroupName lipgloss.Style
	Key       lipgloss.Style
	Desc      lipgloss.Style
	Separator lipgloss.Style
}

// SearchStyles for log search mode
type SearchStyles struct {
	Container   lipgloss.Style
	Prompt      lipgloss.Style
	Input       lipgloss.Style
	InputCursor lipgloss.Style
	Match       lipgloss.Style
	MatchCount  lipgloss.Style
	NoMatch     lipgloss.Style
}

// TabBarStyles for the tab navigation bar
type TabBarStyles struct {
	Container     lipgloss.Style
	Tab           lipgloss.Style
	TabActive     lipgloss.Style
	TabInactive   lipgloss.Style
	TabIcon       lipgloss.Style
	TabLabel      lipgloss.Style
	TabSubtitle   lipgloss.Style
	TabBadge      lipgloss.Style
	TabBadgeError lipgloss.Style
	TabBadgeWarn  lipgloss.Style
	Separator     lipgloss.Style
}

// DefaultStyles returns the complete style configuration
func DefaultStyles() Styles {
	colors := DefaultColors()
	icons := GetIcons()

	return Styles{
		Colors: colors,
		Icons:  icons,

		StatusBar: defaultStatusBarStyles(colors),
		Logs:      defaultLogsStyles(colors),
		HintsBar:  defaultHintsBarStyles(colors),
		Popup:     defaultPopupStyles(colors),
		Selector:  defaultSelectorStyles(colors),
		Help:      defaultHelpStyles(colors),
		Search:    defaultSearchStyles(colors),
		TabBar:    defaultTabBarStyles(colors),
		Syntax:    DefaultSyntaxColors(),
	}
}

func defaultStatusBarStyles(c Colors) StatusBarStyles {
	return StatusBarStyles{
		Container: lipgloss.NewStyle().
			Padding(0, 1).
			Background(c.Surface).
			BorderStyle(lipgloss.Border{Bottom: "─"}).
			BorderForeground(c.Border).
			BorderBottom(true),

		Section: lipgloss.NewStyle().
			Padding(0, 1),

		Label: lipgloss.NewStyle().
			Foreground(c.TextMuted),

		Value: lipgloss.NewStyle().
			Foreground(c.Text),

		ValueMuted: lipgloss.NewStyle().
			Foreground(c.TextMuted),

		Separator: lipgloss.NewStyle().
			Foreground(c.Border).
			SetString("│").
			Padding(0, 1),

		StatusIcon: lipgloss.NewStyle(),

		SpinnerText: lipgloss.NewStyle().
			Foreground(c.Accent),
	}
}

func defaultLogsStyles(c Colors) LogsStyles {
	return LogsStyles{
		Container: lipgloss.NewStyle().
			Padding(0, 1),

		// Phase headers
		PhaseHeader: lipgloss.NewStyle().
			Foreground(c.Text).
			Bold(true).
			Padding(0, 0, 0, 0),

		PhaseHeaderCollapsed: lipgloss.NewStyle().
			Foreground(c.TextMuted).
			Padding(0, 0, 0, 0),

		PhaseIcon: lipgloss.NewStyle().
			Foreground(c.Accent).
			MarginRight(1),

		PhaseCount: lipgloss.NewStyle().
			Foreground(c.TextMuted),

		// Log lines
		Line: lipgloss.NewStyle().
			Foreground(c.Text),

		LineError: lipgloss.NewStyle().
			Foreground(c.Error),

		LineWarn: lipgloss.NewStyle().
			Foreground(c.Warning),

		LineNumber: lipgloss.NewStyle().
			Foreground(c.TextSubtle).
			Width(4).
			Align(lipgloss.Right),

		// Inline errors/warnings
		ErrorIcon: lipgloss.NewStyle().
			Foreground(c.Error).
			Bold(true),

		ErrorText: lipgloss.NewStyle().
			Foreground(c.Error),

		WarningIcon: lipgloss.NewStyle().
			Foreground(c.Warning).
			Bold(true),

		WarningText: lipgloss.NewStyle().
			Foreground(c.Warning),

		// Test results
		TestPass: lipgloss.NewStyle().
			Foreground(c.Success),

		TestFail: lipgloss.NewStyle().
			Foreground(c.Error),

		TestSkip: lipgloss.NewStyle().
			Foreground(c.TextMuted),

		// Search highlighting
		SearchMatch: lipgloss.NewStyle().
			Background(c.Accent).
			Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#FFFFFF"}),

		// Empty state
		EmptyState: lipgloss.NewStyle().
			Foreground(c.TextMuted).
			Align(lipgloss.Center),
	}
}

func defaultHintsBarStyles(c Colors) HintsBarStyles {
	return HintsBarStyles{
		Container: lipgloss.NewStyle().
			Padding(0, 1).
			BorderStyle(lipgloss.Border{Top: "─"}).
			BorderForeground(c.Border).
			BorderTop(true),

		Key: lipgloss.NewStyle().
			Foreground(c.Accent).
			Bold(true),

		Desc: lipgloss.NewStyle().
			Foreground(c.TextMuted),

		Separator: lipgloss.NewStyle().
			Foreground(c.Border).
			SetString("  "),
	}
}

func defaultPopupStyles(c Colors) PopupStyles {
	return PopupStyles{
		Container: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(c.Border).
			Padding(1, 2),

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(c.Text).
			MarginBottom(1),

		Content: lipgloss.NewStyle().
			Foreground(c.Text),
	}
}

func defaultSelectorStyles(c Colors) SelectorStyles {
	return SelectorStyles{
		Container: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(c.Border).
			Padding(1, 2),

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(c.Text).
			MarginBottom(1),

		Input: lipgloss.NewStyle().
			Foreground(c.Text).
			Padding(0, 1).
			MarginBottom(1),

		InputCursor: lipgloss.NewStyle().
			Foreground(c.Accent),

		Item: lipgloss.NewStyle().
			Foreground(c.Text).
			Padding(0, 1),

		ItemSelected: lipgloss.NewStyle().
			Foreground(c.Accent).
			Bold(true).
			Padding(0, 1),

		ItemMeta: lipgloss.NewStyle().
			Foreground(c.TextMuted),

		Divider: lipgloss.NewStyle().
			Foreground(c.BorderMuted),

		Hint: lipgloss.NewStyle().
			Foreground(c.TextSubtle).
			MarginTop(1),
	}
}

func defaultHelpStyles(c Colors) HelpStyles {
	return HelpStyles{
		Container: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(c.Border).
			Padding(1, 2),

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(c.Text).
			MarginBottom(1),

		Group: lipgloss.NewStyle().
			MarginBottom(1),

		GroupName: lipgloss.NewStyle().
			Foreground(c.TextMuted).
			Bold(true).
			MarginBottom(0),

		Key: lipgloss.NewStyle().
			Foreground(c.Accent).
			Bold(true).
			Width(12),

		Desc: lipgloss.NewStyle().
			Foreground(c.TextMuted),

		Separator: lipgloss.NewStyle().
			Foreground(c.BorderMuted),
	}
}

func defaultSearchStyles(c Colors) SearchStyles {
	return SearchStyles{
		Container: lipgloss.NewStyle().
			Padding(0, 1).
			BorderStyle(lipgloss.Border{Top: "─"}).
			BorderForeground(c.Accent).
			BorderTop(true),

		Prompt: lipgloss.NewStyle().
			Foreground(c.Accent).
			Bold(true),

		Input: lipgloss.NewStyle().
			Foreground(c.Text),

		InputCursor: lipgloss.NewStyle().
			Foreground(c.Accent),

		Match: lipgloss.NewStyle().
			Background(c.Accent).
			Foreground(lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#FFFFFF"}),

		MatchCount: lipgloss.NewStyle().
			Foreground(c.TextMuted),

		NoMatch: lipgloss.NewStyle().
			Foreground(c.Error),
	}
}

func defaultTabBarStyles(c Colors) TabBarStyles {
	return TabBarStyles{
		Container: lipgloss.NewStyle().
			Padding(0, 1).
			Background(c.Surface).
			BorderStyle(lipgloss.Border{Bottom: "─"}).
			BorderForeground(c.Border).
			BorderBottom(true),

		Tab: lipgloss.NewStyle().
			Padding(0, 2),

		TabActive: lipgloss.NewStyle().
			Foreground(c.Accent).
			Bold(true).
			Padding(0, 2),

		TabInactive: lipgloss.NewStyle().
			Foreground(c.TextMuted).
			Padding(0, 2),

		TabIcon: lipgloss.NewStyle().
			Foreground(c.Accent).
			MarginRight(1),

		TabLabel: lipgloss.NewStyle().
			Foreground(c.Text),

		TabSubtitle: lipgloss.NewStyle().
			Foreground(c.TextSubtle),

		TabBadge: lipgloss.NewStyle().
			Foreground(c.TextMuted),

		TabBadgeError: lipgloss.NewStyle().
			Foreground(c.Error).
			Bold(true),

		TabBadgeWarn: lipgloss.NewStyle().
			Foreground(c.Warning),

		Separator: lipgloss.NewStyle().
			Foreground(c.Border).
			SetString("│").
			Padding(0, 1),
	}
}

// =============================================================================
// Style Helpers
// =============================================================================

// StatusStyle returns the appropriate style for a status
func (s Styles) StatusStyle(status string) lipgloss.Style {
	switch status {
	case "success", "ok", "passed":
		return lipgloss.NewStyle().Foreground(s.Colors.Success)
	case "error", "failed", "failure":
		return lipgloss.NewStyle().Foreground(s.Colors.Error)
	case "warning", "warn":
		return lipgloss.NewStyle().Foreground(s.Colors.Warning)
	case "running", "in_progress":
		return lipgloss.NewStyle().Foreground(s.Colors.Running)
	default:
		return lipgloss.NewStyle().Foreground(s.Colors.TextMuted)
	}
}

// StatusIcon returns the appropriate icon for a status
func (s Styles) StatusIcon(status string) string {
	switch status {
	case "success", "ok", "passed":
		return s.Icons.Success
	case "error", "failed", "failure":
		return s.Icons.Error
	case "warning", "warn":
		return s.Icons.Warning
	case "running", "in_progress":
		return s.Icons.Running
	case "idle":
		return s.Icons.Idle
	case "paused":
		return s.Icons.Paused
	default:
		return s.Icons.Dot
	}
}
