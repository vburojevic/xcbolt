package tui

import (
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const scrollbarWidth = 1

func renderScrollbarLines(height, content, scrollPos int, styles Styles) []string {
	if height <= 0 {
		return nil
	}

	lines := make([]string, height)
	blank := strings.Repeat(" ", scrollbarWidth)
	if content <= height || content <= 0 {
		for i := 0; i < height; i++ {
			lines[i] = blank
		}
		return lines
	}

	trackStyle := lipgloss.NewStyle().Foreground(styles.Colors.BorderMuted)
	thumbStyle := lipgloss.NewStyle().Foreground(styles.Colors.TextMuted)
	track := trackStyle.Render("│")
	thumb := thumbStyle.Render("█")

	thumbHeight := int(math.Round(float64(height) * float64(height) / float64(content)))
	if thumbHeight < 1 {
		thumbHeight = 1
	}
	if thumbHeight > height {
		thumbHeight = height
	}

	maxScroll := content - height
	if maxScroll < 1 {
		maxScroll = 1
	}
	thumbTop := int(math.Round(float64(scrollPos) / float64(maxScroll) * float64(height-thumbHeight)))
	if thumbTop < 0 {
		thumbTop = 0
	}
	if thumbTop > height-thumbHeight {
		thumbTop = height - thumbHeight
	}

	for i := 0; i < height; i++ {
		lines[i] = track
	}
	for i := thumbTop; i < thumbTop+thumbHeight; i++ {
		if i >= 0 && i < height {
			lines[i] = thumb
		}
	}
	return lines
}

func withScrollbar(content string, height, width, totalLines, scrollPos int, styles Styles) string {
	if height <= 0 || scrollbarWidth <= 0 {
		return content
	}

	lines := strings.Split(content, "\n")
	if len(lines) == 1 && lines[0] == "" {
		lines = []string{}
	}
	lines = padLines(lines, height)

	if width > 0 {
		w := lipgloss.NewStyle().Width(width)
		for i := range lines {
			lines[i] = w.Render(lines[i])
		}
	}

	bar := renderScrollbarLines(height, totalLines, scrollPos, styles)
	if len(bar) != height {
		return strings.Join(lines, "\n")
	}
	for i := 0; i < height; i++ {
		lines[i] += bar[i]
	}
	return strings.Join(lines, "\n")
}

func padLines(lines []string, height int) []string {
	for len(lines) < height {
		lines = append(lines, "")
	}
	if len(lines) > height {
		lines = lines[:height]
	}
	return lines
}
