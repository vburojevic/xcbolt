package util

import (
	"os"
	"path/filepath"
	"testing"
)

type sample struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

func TestWriteAndReadJSONFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sample.json")
	in := sample{Name: "vedran", Age: 42}
	if err := WriteJSONFile(path, in, 0o644); err != nil {
		t.Fatalf("WriteJSONFile: %v", err)
	}
	var out sample
	if err := ReadJSONFile(path, &out); err != nil {
		t.Fatalf("ReadJSONFile: %v", err)
	}
	if out != in {
		t.Fatalf("out = %#v, want %#v", out, in)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file: %v", err)
	}
	if len(b) == 0 || b[len(b)-1] != '\n' {
		t.Fatalf("expected trailing newline in json file")
	}
}

func TestReadJSONFileMissing(t *testing.T) {
	var out sample
	err := ReadJSONFile("/no/such/path.json", &out)
	if err == nil {
		t.Fatalf("expected error for missing path")
	}
}
