package tui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/xcbolt/xcbolt/internal/core"
)

func TestChanEmitter_DropsAfterStop(t *testing.T) {
	events := make(chan core.Event, 1)
	stop := make(chan struct{})
	e := &chanEmitter{ch: events, stop: stop}

	// Fill buffer.
	e.Emit(core.Event{Type: "status", Msg: "one"})

	// Buffer is full: Emit must not block.
	done := make(chan struct{})
	go func() {
		e.Emit(core.Event{Type: "status", Msg: "two"})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("Emit blocked with full buffer")
	}

	// After stop closes: Emit must return immediately.
	close(stop)

	done2 := make(chan struct{})
	go func() {
		e.Emit(core.Event{Type: "status", Msg: "three"})
		close(done2)
	}()
	select {
	case <-done2:
	case <-time.After(250 * time.Millisecond):
		t.Fatal("Emit blocked after stop")
	}
}

func TestWaitForEvent_Stops(t *testing.T) {
	events := make(chan core.Event)
	stop := make(chan struct{})

	cmd := waitForEvent(events, stop)
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()

	close(stop)

	select {
	case msg := <-done:
		if msg != nil {
			t.Fatalf("expected nil msg after stop, got %T", msg)
		}
	case <-time.After(250 * time.Millisecond):
		t.Fatal("waitForEvent did not stop")
	}
}
