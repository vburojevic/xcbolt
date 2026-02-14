package tui

import (
	"strings"
	"testing"
)

func TestStatusBarMinimalView(t *testing.T) {
	styles := DefaultStyles()
	sb := NewStatusBar()
	sb.Scheme = "MyScheme"
	sb.Configuration = "Debug"
	sb.Destination = "iPhone 16 Pro"
	sb.Running = true
	out := sb.ViewMinimal(120, styles)
	if !strings.Contains(out, "MyScheme:Debug") {
		t.Fatalf("missing scheme/config in minimal view: %q", out)
	}
	if !strings.Contains(out, "iPhone") {
		t.Fatalf("missing destination in minimal view: %q", out)
	}
}

func TestStatusBarCenterSectionFallbacks(t *testing.T) {
	styles := DefaultStyles()
	sb := NewStatusBar()
	out := sb.ViewWithMinimal(120, styles, false)
	if strings.TrimSpace(out) == "" {
		t.Fatalf("expected non-empty statusbar output")
	}
	if !strings.Contains(out, "No scheme") {
		t.Fatalf("expected fallback text for scheme: %q", out)
	}
}
