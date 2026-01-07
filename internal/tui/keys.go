package tui

import (
	"github.com/charmbracelet/bubbles/key"
)

type keyMap struct {
	Quit    key.Binding
	Tab     key.Binding
	PrevTab key.Binding

	Init    key.Binding
	Refresh key.Binding

	Build key.Binding
	Run   key.Binding
	Test  key.Binding
	Logs  key.Binding

	Cancel key.Binding
	Help   key.Binding
}

func defaultKeyMap() keyMap {
	return keyMap{
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next tab"),
		),
		PrevTab: key.NewBinding(
			key.WithKeys("shift+tab"),
			key.WithHelp("shift+tab", "prev tab"),
		),
		Init: key.NewBinding(
			key.WithKeys("i"),
			key.WithHelp("i", "init config"),
		),
		Refresh: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "refresh context"),
		),
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
		Logs: key.NewBinding(
			key.WithKeys("l"),
			key.WithHelp("l", "logs tab"),
		),
		Cancel: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "cancel"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
	}
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Build, k.Run, k.Test, k.Logs, k.Tab, k.Init, k.Quit}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Build, k.Run, k.Test, k.Logs},
		{k.Init, k.Refresh, k.Cancel},
		{k.Tab, k.PrevTab, k.Help, k.Quit},
	}
}
