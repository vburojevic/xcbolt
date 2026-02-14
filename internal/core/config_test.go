package core

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnsureProjectDirsCreatesGitignore(t *testing.T) {
	root := t.TempDir()

	if err := EnsureProjectDirs(root); err != nil {
		t.Fatalf("EnsureProjectDirs: %v", err)
	}

	path := filepath.Join(root, ".xcbolt", ".gitignore")
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}

	content := string(b)
	if !strings.Contains(content, "DerivedData/") {
		t.Fatalf("missing DerivedData entry: %q", content)
	}
	if !strings.Contains(content, "Results/") {
		t.Fatalf("missing Results entry: %q", content)
	}
}

func TestEnsureProjectDirsAppendsMissingGitignoreEntries(t *testing.T) {
	root := t.TempDir()
	xcboltDir := filepath.Join(root, ".xcbolt")
	if err := os.MkdirAll(xcboltDir, 0o755); err != nil {
		t.Fatalf("mkdir .xcbolt: %v", err)
	}

	path := filepath.Join(xcboltDir, ".gitignore")
	if err := os.WriteFile(path, []byte("DerivedData/\n# keep\n"), 0o644); err != nil {
		t.Fatalf("write .gitignore: %v", err)
	}

	if err := EnsureProjectDirs(root); err != nil {
		t.Fatalf("EnsureProjectDirs: %v", err)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read .gitignore: %v", err)
	}

	content := string(b)
	if strings.Count(content, "DerivedData/") != 1 {
		t.Fatalf("unexpected DerivedData entries: %q", content)
	}
	if strings.Count(content, "Results/") != 1 {
		t.Fatalf("unexpected Results entries: %q", content)
	}
}

func TestLoadConfigRejectsLegacyVersion(t *testing.T) {
	root := t.TempDir()
	if err := EnsureProjectDirs(root); err != nil {
		t.Fatalf("EnsureProjectDirs: %v", err)
	}
	path := ConfigPath(root)
	legacy := map[string]any{
		"version": 1,
		"scheme":  "App",
	}
	b, err := json.Marshal(legacy)
	if err != nil {
		t.Fatalf("marshal legacy config: %v", err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	_, err = LoadConfig(root, "")
	if err == nil {
		t.Fatalf("expected version mismatch error")
	}
	var verr ConfigVersionError
	if !strings.Contains(err.Error(), "version mismatch") {
		t.Fatalf("unexpected error: %v", err)
	}
	if !errors.As(err, &verr) {
		t.Fatalf("expected ConfigVersionError, got %T", err)
	}
}

func TestSaveAndLoadConfigV3RoundTrip(t *testing.T) {
	root := t.TempDir()
	cfg := DefaultConfig(root)
	cfg.Scheme = "App"
	cfg.Destination = Destination{
		Kind:              DestDevice,
		PlatformFamily:    PlatformWatchOS,
		TargetType:        TargetDevice,
		ID:                "WATCH-UDID",
		UDID:              "WATCH-UDID",
		Name:              "My Watch",
		Platform:          "watchOS",
		CompanionTargetID: "PHONE-UDID",
		CompanionBundleID: "com.example.phone",
	}

	if err := SaveConfig(root, "", cfg); err != nil {
		t.Fatalf("SaveConfig: %v", err)
	}
	loaded, err := LoadConfig(root, "")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if loaded.Version != ConfigVersion {
		t.Fatalf("version = %d, want %d", loaded.Version, ConfigVersion)
	}
	if loaded.Destination.PlatformFamily != PlatformWatchOS {
		t.Fatalf("platformFamily = %q", loaded.Destination.PlatformFamily)
	}
	if loaded.Destination.CompanionTargetID != "PHONE-UDID" {
		t.Fatalf("companion target = %q", loaded.Destination.CompanionTargetID)
	}
}

func TestMigrateConfigFromV2CreatesBackup(t *testing.T) {
	root := t.TempDir()
	if err := EnsureProjectDirs(root); err != nil {
		t.Fatalf("EnsureProjectDirs: %v", err)
	}
	path := ConfigPath(root)
	v2 := map[string]any{
		"version":       2,
		"scheme":        "App",
		"configuration": "Debug",
		"destination": map[string]any{
			"kind":           "simulator",
			"platformFamily": "ios",
			"targetType":     "simulator",
			"udid":           "SIM-1",
		},
	}
	b, err := json.Marshal(v2)
	if err != nil {
		t.Fatalf("marshal config: %v", err)
	}
	if err := os.WriteFile(path, append(b, '\n'), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	res, err := MigrateConfig(root, "")
	if err != nil {
		t.Fatalf("MigrateConfig: %v", err)
	}
	if res.FromVersion != 2 || res.ToVersion != ConfigVersion {
		t.Fatalf("unexpected versions: %+v", res)
	}
	if res.BackupPath == "" {
		t.Fatalf("expected backup path")
	}
	if _, err := os.Stat(res.BackupPath); err != nil {
		t.Fatalf("backup missing: %v", err)
	}

	loaded, err := LoadConfig(root, "")
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if loaded.Version != ConfigVersion {
		t.Fatalf("version = %d, want %d", loaded.Version, ConfigVersion)
	}
	if loaded.Destination.ID != "SIM-1" {
		t.Fatalf("destination id = %q", loaded.Destination.ID)
	}
}
