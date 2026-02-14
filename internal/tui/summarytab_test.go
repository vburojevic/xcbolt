package tui

import (
	"testing"
	"time"
)

func TestSummaryTabRunningAndResult(t *testing.T) {
	st := NewSummaryTab()
	st.SetProjectInfo("Proj", "Scheme", "iPhone", "Debug")
	st.SetSystemInfo("Xcode 16", "Booted", true)
	st.SetRunning("run")
	if st.Status != BuildStatusRunning {
		t.Fatalf("status = %v", st.Status)
	}
	if st.ActionType != "run" {
		t.Fatalf("actionType = %q", st.ActionType)
	}
	st.UpdateProgress("/tmp/Sources/App.swift", 3, 10, "Compile")
	if st.CurrentFile != "App.swift" {
		t.Fatalf("current file = %q", st.CurrentFile)
	}
	if st.FileProgress != 3 || st.FilesTotal != 10 {
		t.Fatalf("progress = %d/%d", st.FileProgress, st.FilesTotal)
	}
	st.IncrementErrors()
	st.IncrementWarnings()
	st.SetResult(BuildStatusSuccess, "12s", []PhaseResult{{Name: "Compile", Count: 10}}, st.ErrorCount, st.WarningCount)
	if st.Status != BuildStatusSuccess {
		t.Fatalf("status = %v", st.Status)
	}
	if !st.LastBuildSuccess {
		t.Fatalf("expected last build success")
	}
	if st.FileCount != 10 {
		t.Fatalf("file count = %d", st.FileCount)
	}
}

func TestSummaryTabElapsedTimeAndSpinner(t *testing.T) {
	st := NewSummaryTab()
	if got := st.ElapsedTime(); got != "0:00" {
		t.Fatalf("elapsed = %q", got)
	}
	st.StartTime = time.Now().Add(-90 * time.Second)
	if got := st.ElapsedTime(); got == "0:00" {
		t.Fatalf("expected non-zero elapsed")
	}
	start := st.SpinnerFrame
	st.AdvanceSpinner()
	if st.SpinnerFrame == start {
		t.Fatalf("spinner did not advance")
	}
}
