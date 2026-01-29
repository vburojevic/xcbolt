package tui

import "testing"

func TestRepeatLastOpWithoutPrevious(t *testing.T) {
	m := Model{}

	cmd := m.repeatLastOp()

	if cmd != nil {
		t.Fatalf("expected nil cmd, got %v", cmd)
	}
	if m.statusMsg != "No previous action to repeat" {
		t.Fatalf("unexpected status: %q", m.statusMsg)
	}
}

func TestRepeatLastOpWhileRunning(t *testing.T) {
	m := Model{
		lastOp:  "build",
		running: true,
	}

	cmd := m.repeatLastOp()

	if cmd != nil {
		t.Fatalf("expected nil cmd, got %v", cmd)
	}
	if m.pendingOp != "build" {
		t.Fatalf("expected pendingOp to be %q, got %q", "build", m.pendingOp)
	}
	if m.statusMsg != "Canceling…" {
		t.Fatalf("unexpected status: %q", m.statusMsg)
	}
}
