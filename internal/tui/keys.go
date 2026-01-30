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
	SelectorConfiguration
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
	Scheme        key.Binding
	Configuration key.Binding
	Destination   key.Binding
	Palette       key.Binding

	// Configuration
	Init    key.Binding
	Refresh key.Binding

	// Tab navigation (new tab-based log view)
	Tab1    key.Binding // Logs tab
	Tab2    key.Binding // Issues tab
	Tab3    key.Binding // Summary tab
	TabNext key.Binding // Cycle to next tab

	// Copy
	CopyLine    key.Binding // Copy current line
	CopyVisible key.Binding // Copy all visible content

	// Display toggles
	ToggleLineNumbers key.Binding
	ToggleTimestamps  key.Binding

	// Search & Navigation
	Search           key.Binding
	NextError        key.Binding
	PrevError        key.Binding
	OpenXcode        key.Binding
	OpenEditor       key.Binding
	ToggleRawView    key.Binding
	ToggleCollapse   key.Binding
	ExpandAll        key.Binding
	CollapseAll      key.Binding
	ToggleErrorsOnly key.Binding

	// Viewport/Scroll (arrow keys + vim keys)
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
			key.WithHelp("x", "stop"),
		),

		// Selectors
		Scheme: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "scheme"),
		),
		Configuration: key.NewBinding(
			key.WithKeys("~"),
			key.WithHelp("~", "build config"),
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

		// Tab navigation
		Tab1: key.NewBinding(
			key.WithKeys("1"),
			key.WithHelp("1", "logs tab"),
		),
		Tab2: key.NewBinding(
			key.WithKeys("2"),
			key.WithHelp("2", "issues tab"),
		),
		Tab3: key.NewBinding(
			key.WithKeys("3"),
			key.WithHelp("3", "summary tab"),
		),
		TabNext: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next tab"),
		),

		// Copy
		CopyLine: key.NewBinding(
			key.WithKeys("y"),
			key.WithHelp("y", "copy line"),
		),
		CopyVisible: key.NewBinding(
			key.WithKeys("Y"),
			key.WithHelp("Y", "copy visible"),
		),

		// Display toggles
		ToggleLineNumbers: key.NewBinding(
			key.WithKeys("L"),
			key.WithHelp("L", "toggle line numbers"),
		),
		ToggleTimestamps: key.NewBinding(
			key.WithKeys("T"),
			key.WithHelp("T", "toggle timestamps"),
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
			key.WithHelp("v", "toggle logs view"),
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
		ToggleErrorsOnly: key.NewBinding(
			key.WithKeys("F"),
			key.WithHelp("F", "toggle errors-only"),
		),

		// Viewport/Scroll - vim keys + arrow keys
		ScrollUp: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "scroll up"),
		),
		ScrollDown: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "scroll down"),
		),
		ScrollTop: key.NewBinding(
			key.WithKeys("home", "g"),
			key.WithHelp("home/g", "top"),
		),
		ScrollBottom: key.NewBinding(
			key.WithKeys("end", "G"),
			key.WithHelp("end/G", "bottom"),
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
		// Repurposed: now toggles stream view
		ToggleAutoFollow: key.NewBinding(
			key.WithKeys("f"),
			key.WithHelp("f", "toggle logs view"),
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
	return []key.Binding{k.Run, k.Build, k.Test, k.Scheme, k.Configuration, k.Destination, k.Help, k.Quit}
}

// FullHelp returns all bindings grouped for full help view
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		// Actions
		{k.Build, k.Run, k.Test, k.Clean, k.Stop},
		// Configuration
		{k.Scheme, k.Configuration, k.Destination, k.Palette, k.Init, k.Refresh},
		// Tabs
		{k.Tab1, k.Tab2, k.Tab3, k.TabNext},
		// View controls
		{k.ToggleRawView, k.ToggleLineNumbers, k.ToggleTimestamps, k.ToggleErrorsOnly, k.ExpandAll, k.CollapseAll},
		// Scrolling
		{k.ScrollUp, k.ScrollDown, k.PageUp, k.PageDown, k.ScrollTop, k.ScrollBottom},
		// Navigation & Other
		{k.Search, k.NextError, k.PrevError, k.CopyLine, k.CopyVisible, k.OpenXcode, k.OpenEditor, k.Cancel, k.Help, k.Quit},
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
	return "b:build  r:run  t:test  1-3:tabs  /:search  ?:help"
}
