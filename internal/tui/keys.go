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
	ModeSearch // New search mode
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

	// Search & Navigation
	Search         key.Binding
	NextError      key.Binding
	PrevError      key.Binding
	OpenXcode      key.Binding
	OpenEditor     key.Binding
	ToggleRawView  key.Binding
	ToggleCollapse key.Binding
	ExpandAll      key.Binding
	CollapseAll    key.Binding

	// Viewport/Scroll (arrow keys only, no vim keys)
	ScrollUp     key.Binding
	ScrollDown   key.Binding
	ScrollTop    key.Binding
	ScrollBottom key.Binding
	PageUp       key.Binding
	PageDown     key.Binding
	HalfPageUp   key.Binding
	HalfPageDown key.Binding

	// Keep for backwards compatibility but repurposed
	ToggleAutoFollow key.Binding

	// Selector navigation (used when in selector/palette mode)
	SelectUp    key.Binding
	SelectDown  key.Binding
	SelectEnter key.Binding

	// Run mode split view
	SwitchPane key.Binding
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

		// Search & Navigation
		Search: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search logs"),
		),
		NextError: key.NewBinding(
			key.WithKeys("n"),
			key.WithHelp("n", "next error"),
		),
		PrevError: key.NewBinding(
			key.WithKeys("N"),
			key.WithHelp("N", "prev error"),
		),
		OpenXcode: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "open in Xcode"),
		),
		OpenEditor: key.NewBinding(
			key.WithKeys("O"),
			key.WithHelp("O", "open in $EDITOR"),
		),
		ToggleRawView: key.NewBinding(
			key.WithKeys("v"),
			key.WithHelp("v", "toggle raw/grouped view"),
		),
		ToggleCollapse: key.NewBinding(
			key.WithKeys("enter", " "),
			key.WithHelp("enter", "toggle phase collapse"),
		),
		ExpandAll: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "expand all phases"),
		),
		CollapseAll: key.NewBinding(
			key.WithKeys("E"),
			key.WithHelp("E", "collapse all phases"),
		),

		// Viewport/Scroll - arrow keys only (no vim j/k)
		ScrollUp: key.NewBinding(
			key.WithKeys("up"),
			key.WithHelp("↑", "scroll up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("down"),
			key.WithHelp("↓", "scroll down"),
		),
		ScrollTop: key.NewBinding(
			key.WithKeys("home"),
			key.WithHelp("home", "top"),
		),
		ScrollBottom: key.NewBinding(
			key.WithKeys("end"),
			key.WithHelp("end", "bottom"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("pgup"),
			key.WithHelp("PgUp", "page up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("pgdown"),
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
		// Repurposed: now toggles raw/grouped view
		ToggleAutoFollow: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "toggle view"),
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
			key.WithHelp("enter", "select"),
		),

		// Run mode split view
		SwitchPane: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "switch pane"),
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
		// Actions
		{k.Build, k.Run, k.Test, k.Clean, k.Stop},
		// Configuration
		{k.Scheme, k.Destination, k.Palette, k.Init, k.Refresh},
		// View controls
		{k.ToggleRawView, k.ToggleCollapse, k.ExpandAll, k.CollapseAll},
		// Scrolling
		{k.ScrollUp, k.ScrollDown, k.PageUp, k.PageDown, k.ScrollTop, k.ScrollBottom},
		// Navigation & Other
		{k.Search, k.NextError, k.PrevError, k.OpenXcode, k.OpenEditor, k.Help, k.Quit},
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
	return "b:build  r:run  t:test  s:scheme  d:dest  ^K:palette  ?:help"
}
