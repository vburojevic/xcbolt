package tui

import (
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// =============================================================================
// Color Palette - Pastel Theme
// =============================================================================

// Colors defines the complete color palette with light/dark mode support
type Colors struct {
	// Primary accent color
	Accent lipgloss.AdaptiveColor

	// Semantic colors
	Success lipgloss.AdaptiveColor
	Warning lipgloss.AdaptiveColor
	Error   lipgloss.AdaptiveColor
	Running lipgloss.AdaptiveColor

	// Background colors
	Background lipgloss.AdaptiveColor
	Surface    lipgloss.AdaptiveColor

	// Text colors
	Text       lipgloss.AdaptiveColor
	TextMuted  lipgloss.AdaptiveColor
	TextSubtle lipgloss.AdaptiveColor

	// Border colors
	Border      lipgloss.AdaptiveColor
	BorderMuted lipgloss.AdaptiveColor
}

// DefaultColors returns the pastel color palette
func DefaultColors() Colors {
	return Colors{
		// Soft sky blue accent
		Accent: lipgloss.AdaptiveColor{Light: "#5BA4E0", Dark: "#A8D8FF"},

		// Semantic pastel colors
		Success: lipgloss.AdaptiveColor{Light: "#4CAF7D", Dark: "#A8E6CF"}, // Mint green
		Warning: lipgloss.AdaptiveColor{Light: "#E6A23C", Dark: "#FFE5B4"}, // Peach
		Error:   lipgloss.AdaptiveColor{Light: "#E57373", Dark: "#FFB5B5"}, // Soft coral
		Running: lipgloss.AdaptiveColor{Light: "#9575CD", Dark: "#D4B8FF"}, // Soft lavender

		// Backgrounds
		Background: lipgloss.AdaptiveColor{Light: "#FAFAFA", Dark: "#1A1A2E"}, // Deep navy / off-white
		Surface:    lipgloss.AdaptiveColor{Light: "#FFFFFF", Dark: "#252542"}, // Muted purple-gray

		// Text
		Text:       lipgloss.AdaptiveColor{Light: "#2D2D3A", Dark: "#E8E8F0"},
		TextMuted:  lipgloss.AdaptiveColor{Light: "#6B6B8D", Dark: "#9E9EB0"},
		TextSubtle: lipgloss.AdaptiveColor{Light: "#9E9EB0", Dark: "#6B6B8D"},

		// Borders
		Border:      lipgloss.AdaptiveColor{Light: "#E0E0E0", Dark: "#3D3D5C"},
		BorderMuted: lipgloss.AdaptiveColor{Light: "#EEEEEE", Dark: "#2D2D4A"},
	}
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
		Build:   "🔨",
		Run:     "▶",
		Test:    "⚗",
		Clean:   "🗑",
		Stop:    "■",
		Archive: "📦",

		// Navigation
		ChevronDown:  "▾",
		ChevronRight: "▸",
		ArrowRight:   "→",

		// UI
		Search:   "🔍",
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

	// Default to Unicode fallback for safety
	return false
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
	Header     HeaderStyles
	ActionBar  ActionBarStyles
	Logs       LogsStyles
	ResultsBar ResultsBarStyles
	Popup      PopupStyles
	Selector   SelectorStyles
	Help       HelpStyles
}

// HeaderStyles for the top header bar
type HeaderStyles struct {
	Container   lipgloss.Style
	Brand       lipgloss.Style
	Selector    lipgloss.Style
	SelectorKey lipgloss.Style
	Status      lipgloss.Style
	StatusIcon  lipgloss.Style
}

// ActionBarStyles for the action buttons row
type ActionBarStyles struct {
	Container  lipgloss.Style
	Action     lipgloss.Style
	ActionKey  lipgloss.Style
	ActionText lipgloss.Style
	Separator  lipgloss.Style
}

// LogsStyles for the scrollable log viewport
type LogsStyles struct {
	Container  lipgloss.Style
	Line       lipgloss.Style
	LineNumber lipgloss.Style
	Timestamp  lipgloss.Style
	StatusIcon lipgloss.Style
	EmptyState lipgloss.Style
}

// ResultsBarStyles for the bottom results/status bar
type ResultsBarStyles struct {
	Container lipgloss.Style
	Result    lipgloss.Style
	Icon      lipgloss.Style
	Duration  lipgloss.Style
	Hints     lipgloss.Style
	HintKey   lipgloss.Style
}

// PopupStyles for centered modal popups
type PopupStyles struct {
	Container lipgloss.Style
	Title     lipgloss.Style
	Border    lipgloss.Style
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
	Group     lipgloss.Style
	Key       lipgloss.Style
	Desc      lipgloss.Style
	Separator lipgloss.Style
}

// DefaultStyles returns the complete style configuration
func DefaultStyles() Styles {
	colors := DefaultColors()
	icons := GetIcons()

	return Styles{
		Colors: colors,
		Icons:  icons,

		Header:     defaultHeaderStyles(colors),
		ActionBar:  defaultActionBarStyles(colors),
		Logs:       defaultLogsStyles(colors),
		ResultsBar: defaultResultsBarStyles(colors),
		Popup:      defaultPopupStyles(colors),
		Selector:   defaultSelectorStyles(colors),
		Help:       defaultHelpStyles(colors),
	}
}

func defaultHeaderStyles(c Colors) HeaderStyles {
	return HeaderStyles{
		Container: lipgloss.NewStyle().
			Padding(0, 1).
			BorderStyle(lipgloss.Border{Bottom: "─"}).
			BorderForeground(c.Border).
			BorderBottom(true),

		Brand: lipgloss.NewStyle().
			Bold(true).
			Foreground(c.Text),

		Selector: lipgloss.NewStyle().
			Foreground(c.Text).
			Padding(0, 1),

		SelectorKey: lipgloss.NewStyle().
			Foreground(c.Accent),

		Status: lipgloss.NewStyle().
			Foreground(c.TextMuted),

		StatusIcon: lipgloss.NewStyle(),
	}
}

func defaultActionBarStyles(c Colors) ActionBarStyles {
	return ActionBarStyles{
		Container: lipgloss.NewStyle().
			Padding(0, 1).
			BorderStyle(lipgloss.Border{Bottom: "─"}).
			BorderForeground(c.BorderMuted).
			BorderBottom(true),

		Action: lipgloss.NewStyle().
			Padding(0, 1),

		ActionKey: lipgloss.NewStyle().
			Foreground(c.Accent).
			Bold(true),

		ActionText: lipgloss.NewStyle().
			Foreground(c.TextMuted),

		Separator: lipgloss.NewStyle().
			Foreground(c.BorderMuted),
	}
}

func defaultLogsStyles(c Colors) LogsStyles {
	return LogsStyles{
		Container: lipgloss.NewStyle().
			Padding(0, 1),

		Line: lipgloss.NewStyle().
			Foreground(c.Text),

		LineNumber: lipgloss.NewStyle().
			Foreground(c.TextSubtle).
			Width(4).
			Align(lipgloss.Right),

		Timestamp: lipgloss.NewStyle().
			Foreground(c.TextSubtle),

		StatusIcon: lipgloss.NewStyle(),

		EmptyState: lipgloss.NewStyle().
			Foreground(c.TextMuted).
			Italic(true).
			Align(lipgloss.Center),
	}
}

func defaultResultsBarStyles(c Colors) ResultsBarStyles {
	return ResultsBarStyles{
		Container: lipgloss.NewStyle().
			Padding(0, 1).
			BorderStyle(lipgloss.Border{Top: "─"}).
			BorderForeground(c.Border).
			BorderTop(true),

		Result: lipgloss.NewStyle().
			Foreground(c.Text),

		Icon: lipgloss.NewStyle(),

		Duration: lipgloss.NewStyle().
			Foreground(c.TextMuted),

		Hints: lipgloss.NewStyle().
			Foreground(c.TextSubtle),

		HintKey: lipgloss.NewStyle().
			Foreground(c.Accent),
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

		Border: lipgloss.NewStyle().
			BorderForeground(c.Border),

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

		Group: lipgloss.NewStyle().
			MarginBottom(1),

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

// =============================================================================
// Legacy compatibility - to be removed after full migration
// =============================================================================

// styleSet is the legacy style struct - kept for gradual migration
type styleSet struct {
	Header      lipgloss.Style
	Brand       lipgloss.Style
	Meta        lipgloss.Style
	TabActive   lipgloss.Style
	TabInactive lipgloss.Style
	Panel       lipgloss.Style
	PanelTitle  lipgloss.Style
	Muted       lipgloss.Style
	OK          lipgloss.Style
	Warn        lipgloss.Style
	Err         lipgloss.Style
	Toast       lipgloss.Style
}

// defaultStyles returns the legacy style set for backwards compatibility
func defaultStyles() styleSet {
	c := DefaultColors()
	return styleSet{
		Header: lipgloss.NewStyle().
			Padding(0, 1).
			Border(lipgloss.Border{Bottom: "─"}).
			BorderForeground(c.Border),
		Brand: lipgloss.NewStyle().
			Bold(true).
			Foreground(c.Text),
		Meta: lipgloss.NewStyle().
			Foreground(c.TextMuted),
		TabActive: lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1).
			Border(lipgloss.Border{Bottom: "━"}).
			BorderForeground(c.Accent),
		TabInactive: lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(c.TextMuted),
		Panel: lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(c.Border),
		PanelTitle: lipgloss.NewStyle().
			Bold(true),
		Muted: lipgloss.NewStyle().
			Foreground(c.TextMuted),
		OK: lipgloss.NewStyle().
			Foreground(c.Success),
		Warn: lipgloss.NewStyle().
			Foreground(c.Warning),
		Err: lipgloss.NewStyle().
			Foreground(c.Error),
		Toast: lipgloss.NewStyle().
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(c.Border),
	}
}
