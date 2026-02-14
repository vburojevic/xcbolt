package core

import (
	"os"
	"path/filepath"
	"testing"

	"howett.net/plist"
)

func TestReadAppBundleInfoWatchFields(t *testing.T) {
	root := t.TempDir()
	appPath := filepath.Join(root, "Watch.app")
	if err := os.MkdirAll(appPath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	m := map[string]any{
		"CFBundleIdentifier":             "com.example.watch",
		"CFBundleDisplayName":            "Watch App",
		"CFBundleName":                   "WatchApp",
		"CFBundleExecutable":             "WatchExec",
		"WKWatchKitApp":                  true,
		"WKCompanionAppBundleIdentifier": "com.example.phone",
		"CFBundleShortVersionString":     "1.0",
		"CFBundleVersion":                "100",
	}
	b, err := plist.Marshal(m, plist.XMLFormat)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(filepath.Join(appPath, "Info.plist"), b, 0o644); err != nil {
		t.Fatalf("write plist: %v", err)
	}

	info, err := ReadAppBundleInfo(appPath)
	if err != nil {
		t.Fatalf("ReadAppBundleInfo: %v", err)
	}
	if !info.IsWatchApp {
		t.Fatalf("expected IsWatchApp true")
	}
	if info.CompanionBundleID != "com.example.phone" {
		t.Fatalf("companion bundle = %q", info.CompanionBundleID)
	}
}
