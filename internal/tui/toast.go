package tui

import (
	"strings"
	"time"

	"github.com/charmbracelet/harmonica"
)

type toastModel struct {
	msg     string
	visible bool
	until   time.Time

	x float64
	v float64

	spring harmonica.Spring
}

func newToast() toastModel {
	return toastModel{
		spring: harmonica.NewSpring(harmonica.FPS(60), 8.0, 0.65),
	}
}

func (t *toastModel) Show(msg string, d time.Duration) {
	t.msg = msg
	t.visible = true
	t.until = time.Now().Add(d)
	// Reset animation state for snappy feedback.
	t.x = 0
	t.v = 0
}

func (t *toastModel) Update() {
	if t.msg == "" {
		return
	}
	if t.visible && time.Now().After(t.until) {
		t.visible = false
	}
	target := 0.0
	if t.visible {
		target = 1.0
	}
	t.x, t.v = t.spring.Update(t.x, t.v, target)
	// When fully hidden, clear.
	if !t.visible && t.x < 0.02 {
		t.msg = ""
	}
}

func (t toastModel) View(styles Styles) string {
	if t.msg == "" {
		return ""
	}
	// Slide in from left.
	offset := int((1.0 - t.x) * 18.0)
	if offset < 0 {
		offset = 0
	}
	pad := strings.Repeat(" ", offset)
	return pad + styles.Toast.Container.Render(t.msg)
}
