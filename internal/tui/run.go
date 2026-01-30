package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

func Run(projectRoot string, configPath string, overrides ConfigOverrides) error {
	m := NewModel(projectRoot, configPath, overrides)
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithReportFocus(), // required for huh focus support in larger programs
	)
	_, err := p.Run()
	return err
}
