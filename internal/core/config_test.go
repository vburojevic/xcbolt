package core

import (
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
