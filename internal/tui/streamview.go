package tui

import "strings"

// StreamView renders raw or pretty log streams with scroll support.
type StreamView struct {
	RawLines    []string
	PrettyLines []string
	ShowPretty  bool
	ScrollPos   int
	VisibleRows int
	AutoFollow  bool
	Width       int
}

func NewStreamView() StreamView {
	return StreamView{
		AutoFollow: true,
	}
}

func (v *StreamView) SetSize(width, height int) {
	v.Width = width
	v.VisibleRows = height
}

func (v *StreamView) Clear() {
	v.RawLines = nil
	v.PrettyLines = nil
	v.ShowPretty = false
	v.ScrollPos = 0
	v.AutoFollow = true
}

func (v *StreamView) AddRawLine(line string) {
	if strings.TrimSpace(line) == "" {
		return
	}
	v.RawLines = append(v.RawLines, line)
	if len(v.RawLines) > 4000 {
		v.RawLines = v.RawLines[len(v.RawLines)-4000:]
	}
	v.autoScroll()
}

func (v *StreamView) AddPrettyLine(line string) {
	if strings.TrimSpace(line) == "" {
		return
	}
	v.PrettyLines = append(v.PrettyLines, line)
	if len(v.PrettyLines) > 4000 {
		v.PrettyLines = v.PrettyLines[len(v.PrettyLines)-4000:]
	}
	v.ShowPretty = true
	v.autoScroll()
}

func (v *StreamView) lines() []string {
	if v.ShowPretty && len(v.PrettyLines) > 0 {
		return v.PrettyLines
	}
	return v.RawLines
}

func (v *StreamView) autoScroll() {
	lines := v.lines()
	if !v.AutoFollow {
		return
	}
	maxScroll := len(lines) - v.VisibleRows
	if maxScroll < 0 {
		maxScroll = 0
	}
	if v.ScrollPos >= maxScroll-3 {
		v.ScrollPos = maxScroll
	}
}

func (v *StreamView) ScrollUp(lines int) {
	v.ScrollPos -= lines
	if v.ScrollPos < 0 {
		v.ScrollPos = 0
	}
	v.AutoFollow = false
}

func (v *StreamView) ScrollDown(lines int) {
	maxScroll := len(v.lines()) - v.VisibleRows
	if maxScroll < 0 {
		maxScroll = 0
	}
	v.ScrollPos += lines
	if v.ScrollPos > maxScroll {
		v.ScrollPos = maxScroll
	}
	v.AutoFollow = v.ScrollPos == maxScroll
}

func (v *StreamView) GotoTop() {
	v.ScrollPos = 0
	v.AutoFollow = false
}

func (v *StreamView) GotoBottom() {
	maxScroll := len(v.lines()) - v.VisibleRows
	if maxScroll < 0 {
		maxScroll = 0
	}
	v.ScrollPos = maxScroll
	v.AutoFollow = true
}

func (v StreamView) View() string {
	lines := v.lines()
	if len(lines) == 0 {
		return ""
	}

	startIdx := v.ScrollPos
	endIdx := startIdx + v.VisibleRows
	if startIdx >= len(lines) {
		startIdx = len(lines) - 1
		if startIdx < 0 {
			startIdx = 0
		}
	}
	if endIdx > len(lines) {
		endIdx = len(lines)
	}

	return strings.Join(lines[startIdx:endIdx], "\n")
}

func (v StreamView) HasLines() bool {
	return len(v.lines()) > 0
}
