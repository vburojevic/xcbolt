package tui

import (
	"github.com/charmbracelet/bubbles/key"
)

// Mode represents the current UI mode
type Mode int

const (
	ModeNormal Mode = iota
	ModeSelector
	ModePalette
	ModeHelp
	ModeWizard
)

// SelectorType represents what the selector is selecting
type SelectorType int

const (
	SelectorScheme SelectorType = iota
	SelectorDestination
)

// keyMap defines all keybindings for the TUI
type keyMap struct {
	// Navigation
	Quit   key.Binding
	Help   key.Binding
	Cancel key.Binding

	// Actions
	Build key.Binding
	Run   key.Binding
	Test  key.Binding
	Clean key.Binding
	Stop  key.Binding

	// Selectors
	Scheme      key.Binding
	Destination key.Binding
	Palette     key.Binding

	// Configuration
	Init    key.Binding
	Refresh key.Binding

	// Layout
	ToggleSidebar key.Binding
	SwitchFocus   key.Binding
	Search        key.Binding

	// Viewport/Scroll
	ScrollUp         key.Binding
	ScrollDown       key.Binding
	ScrollTop        key.Binding
	ScrollBottom     key.Binding
	PageUp           key.Binding
	PageDown         key.Binding
	HalfPageUp       key.Binding
	HalfPageDown     key.Binding
	ToggleAutoFollow key.Binding

	// Selector navigation (used when in selector/palette mode)
	SelectUp    key.Binding
	SelectDown  key.Binding
	SelectEnter key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		// Navigation
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel/close"),
		),

		// Actions
		Build: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "build"),
		),
		Run: key.NewBinding(
			key.WithKeys("r"),
			key.WithHelp("r", "run"),
		),
		Test: key.NewBinding(
			key.WithKeys("t"),
			key.WithHelp("t", "test"),
		),
		Clean: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "clean"),
		),
		Stop: key.NewBinding(
			key.WithKeys("x"),
			key.WithHelp("x", "stop app"),
		),

		// Selectors
		Scheme: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "scheme"),
		),
		Destination: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "destination"),
		),
		Palette: key.NewBinding(
			key.WithKeys("ctrl+k"),
			key.WithHelp("^K", "commands"),
		),

		// Configuration
		Init: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "init config"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("ctrl+r"),
			key.WithHelp("^R", "refresh"),
		),

		// Layout
		ToggleSidebar: key.NewBinding(
			key.WithKeys("ctrl+b"),
			key.WithHelp("^B", "toggle sidebar"),
		),
		SwitchFocus: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("Tab", "switch focus"),
		),
		Search: key.NewBinding(
			key.WithKeys("ctrl+f", "/"),
			key.WithHelp("^F", "search logs"),
		),

		// Viewport/Scroll
		ScrollUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "scroll up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "scroll down"),
		),
		ScrollTop: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g", "top"),
		),
		ScrollBottom: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G", "bottom"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup", "ctrl+b"),
			key.WithHelp("PgUp", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown", "ctrl+f"),
			key.WithHelp("PgDn", "page down"),
		),
		HalfPageUp: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("^U", "half page up"),
		),
		HalfPageDown: key.NewBinding(
			key.WithKeys("ctrl+d"),
			key.WithHelp("^D", "half page down"),
		),
		ToggleAutoFollow: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "follow logs"),
		),

		// Selector navigation
		SelectUp: key.NewBinding(
			key.WithKeys("up", "ctrl+p"),
			key.WithHelp("↑", "previous"),
		),
		SelectDown: key.NewBinding(
			key.WithKeys("down", "ctrl+n"),
			key.WithHelp("↓", "next"),
		),
		SelectEnter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("⏎", "select"),
		),
	}
}

// ShortHelp returns bindings shown in compact help
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Run, k.Build, k.Test, k.Scheme, k.Destination, k.Help, k.Quit}
}

// FullHelp returns all bindings grouped for full help view
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		// Actions row
		{k.Build, k.Run, k.Test, k.Clean, k.Stop},
		// Configuration row
		{k.Scheme, k.Destination, k.Palette, k.Init},
		// Layout row
		{k.SwitchFocus, k.ToggleSidebar, k.Search},
		// Scrolling row
		{k.ScrollUp, k.ScrollDown, k.ScrollTop, k.ScrollBottom, k.ToggleAutoFollow},
		// Other row
		{k.Refresh, k.Cancel, k.Help, k.Quit},
	}
}

// ActionHints returns the hints for the action bar
func (k keyMap) ActionHints() []struct {
	Key  string
	Name string
} {
	return []struct {
		Key  string
		Name string
	}{
		{"r", "Run"},
		{"b", "Build"},
		{"t", "Test"},
		{"c", "Clean"},
		{"^K", "Commands"},
	}
}

// FooterHints returns the hints for the footer bar
func (k keyMap) FooterHints() string {
	return "s:scheme  d:dest  ?:help  q:quit"
}
