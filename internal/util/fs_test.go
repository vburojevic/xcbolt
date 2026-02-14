package util

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExists(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file.txt")
	if Exists(f) {
		t.Fatalf("expected file to not exist")
	}
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if !Exists(f) {
		t.Fatalf("expected file to exist")
	}
}

func TestFindProjectRootPrefersMarkers(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	got, err := FindProjectRoot(nested)
	if err != nil {
		t.Fatalf("FindProjectRoot: %v", err)
	}
	if got != root {
		t.Fatalf("root = %q, want %q", got, root)
	}
}

func TestFindProjectRootFallsBackToStart(t *testing.T) {
	root := t.TempDir()
	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	got, err := FindProjectRoot(nested)
	if err != nil {
		t.Fatalf("FindProjectRoot: %v", err)
	}
	if got != nested {
		t.Fatalf("root = %q, want %q", got, nested)
	}
}

func TestListFilesWithSuffix(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "A.xcodeproj"), 0o755); err != nil {
		t.Fatalf("mkdir A: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "B.xcworkspace"), 0o755); err != nil {
		t.Fatalf("mkdir B: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "note.xcodeproj"), []byte("nope"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	got, err := ListFilesWithSuffix(dir, ".xcodeproj")
	if err != nil {
		t.Fatalf("ListFilesWithSuffix: %v", err)
	}
	if len(got) != 1 || filepath.Base(got[0]) != "A.xcodeproj" {
		t.Fatalf("unexpected list: %v", got)
	}
}

func TestRemoveAllIfExists(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "to-remove")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatalf("mkdir target: %v", err)
	}
	if err := RemoveAllIfExists(target); err != nil {
		t.Fatalf("RemoveAllIfExists: %v", err)
	}
	if Exists(target) {
		t.Fatalf("target should be removed")
	}
	if err := RemoveAllIfExists(target); err != nil {
		t.Fatalf("RemoveAllIfExists missing target: %v", err)
	}
}

func TestWalkFind(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a")
	b := filepath.Join(a, "b")
	if err := os.MkdirAll(b, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	target := filepath.Join(b, "needle.txt")
	if err := os.WriteFile(target, []byte("x"), 0o644); err != nil {
		t.Fatalf("write needle: %v", err)
	}
	found, err := WalkFind(dir, func(path string, d os.DirEntry) bool {
		return filepath.Base(path) == "needle.txt"
	})
	if err != nil {
		t.Fatalf("WalkFind: %v", err)
	}
	if found != target {
		t.Fatalf("found = %q, want %q", found, target)
	}
}
