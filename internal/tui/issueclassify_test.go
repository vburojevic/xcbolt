package tui

import "testing"

func TestIssueSeverity(t *testing.T) {
	tests := []struct {
		line string
		want TabLineType
	}{
		{"warning: something", TabLineTypeWarning},
		{"note: something", TabLineTypeNote},
		{"fatal error: boom", TabLineTypeError},
		{"clang: error: link failed", TabLineTypeError},
		{"Sources/App.swift:12:5: error: no such module", TabLineTypeError},
		{"error: something weird", TabLineTypeWarning},
		{"normal output", TabLineTypeNormal},
	}
	for _, tc := range tests {
		got := issueSeverity(tc.line)
		if got != tc.want {
			t.Fatalf("issueSeverity(%q) = %v, want %v", tc.line, got, tc.want)
		}
	}
}
