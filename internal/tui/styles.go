package tui

import "github.com/charmbracelet/lipgloss"

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

func defaultStyles() styleSet {
	return styleSet{
		Header: lipgloss.NewStyle().
			Padding(0, 1).
			Border(lipgloss.Border{Bottom: "─"}).
			BorderForeground(lipgloss.AdaptiveColor{Light: "#999999", Dark: "#333333"}),
		Brand: lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"}),
		Meta: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#555555", Dark: "#AAAAAA"}),
		TabActive: lipgloss.NewStyle().
			Bold(true).
			Padding(0, 1).
			Border(lipgloss.Border{Bottom: "━"}).
			BorderForeground(lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"}),
		TabInactive: lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.AdaptiveColor{Light: "#777777", Dark: "#888888"}),
		Panel: lipgloss.NewStyle().
			Padding(1, 2).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.AdaptiveColor{Light: "#CCCCCC", Dark: "#303030"}),
		PanelTitle: lipgloss.NewStyle().
			Bold(true),
		Muted: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#777777"}),
		OK: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#0B7A0B", Dark: "#3AE374"}),
		Warn: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#9A6B00", Dark: "#F5C542"}),
		Err: lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#B00020", Dark: "#FF5F87"}),
		Toast: lipgloss.NewStyle().
			Padding(0, 1).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.AdaptiveColor{Light: "#DDDDDD", Dark: "#444444"}),
	}
}
