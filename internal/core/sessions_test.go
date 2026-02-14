package core

import (
	"encoding/json"
	"os"
	"testing"
)

func TestLoadSessionsResetsLegacyVersion(t *testing.T) {
	root := t.TempDir()
	if err := EnsureProjectDirs(root); err != nil {
		t.Fatalf("EnsureProjectDirs: %v", err)
	}
	legacy := map[string]any{
		"version": 1,
		"items": []map[string]any{{
			"id":       "com.example.app",
			"bundleId": "com.example.app",
		}},
	}
	b, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(sessionsPath(root), append(b, '\n'), 0o644); err != nil {
		t.Fatalf("write sessions: %v", err)
	}

	s, err := LoadSessions(root)
	if err != nil {
		t.Fatalf("LoadSessions: %v", err)
	}
	if s.Version != SessionsVersion {
		t.Fatalf("version = %d, want %d", s.Version, SessionsVersion)
	}
	if len(s.Items) != 0 {
		t.Fatalf("items len = %d, want 0", len(s.Items))
	}
}

func TestAddSessionWithDestinationStoresMetadata(t *testing.T) {
	root := t.TempDir()
	dst := Destination{
		Kind:              DestDevice,
		PlatformFamily:    PlatformWatchOS,
		TargetType:        TargetDevice,
		ID:                "WATCH-UDID",
		CompanionTargetID: "PHONE-UDID",
		CompanionBundleID: "com.example.phone",
	}
	sess, err := AddSessionWithDestination(root, "com.example.watch", 1234, dst)
	if err != nil {
		t.Fatalf("AddSessionWithDestination: %v", err)
	}
	if sess.TargetID != "WATCH-UDID" {
		t.Fatalf("target id = %q", sess.TargetID)
	}
	if sess.CompanionTargetID != "PHONE-UDID" {
		t.Fatalf("companion target id = %q", sess.CompanionTargetID)
	}
	if sess.CompanionBundleID != "com.example.phone" {
		t.Fatalf("companion bundle id = %q", sess.CompanionBundleID)
	}

	loaded, err := LoadSessions(root)
	if err != nil {
		t.Fatalf("LoadSessions: %v", err)
	}
	if len(loaded.Items) != 1 {
		t.Fatalf("items len = %d, want 1", len(loaded.Items))
	}
}

func TestRemoveSessionByBundleID(t *testing.T) {
	root := t.TempDir()
	_, err := AddSession(root, "com.example.app", 999, "device", "DEVICE-1")
	if err != nil {
		t.Fatalf("AddSession: %v", err)
	}
	if err := RemoveSession(root, "com.example.app"); err != nil {
		t.Fatalf("RemoveSession: %v", err)
	}
	loaded, err := LoadSessions(root)
	if err != nil {
		t.Fatalf("LoadSessions: %v", err)
	}
	if len(loaded.Items) != 0 {
		t.Fatalf("items len = %d, want 0", len(loaded.Items))
	}
}
