package util

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func Exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func FindProjectRoot(start string) (string, error) {
	start, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	cur := start
	for {
		if looksLikeRoot(cur) {
			return cur, nil
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			return start, nil
		}
		cur = parent
	}
}

func looksLikeRoot(dir string) bool {
	if Exists(filepath.Join(dir, ".xcbolt", "config.json")) {
		return true
	}
	if Exists(filepath.Join(dir, ".git")) {
		return true
	}
	// Any workspace/project file in this folder.
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasSuffix(name, ".xcworkspace") || strings.HasSuffix(name, ".xcodeproj") {
			return true
		}
	}
	return false
}

func ListFilesWithSuffix(dir, suffix string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	out := []string{}
	for _, e := range entries {
		if e.IsDir() && strings.HasSuffix(e.Name(), suffix) {
			out = append(out, filepath.Join(dir, e.Name()))
		}
	}
	return out, nil
}

func RemoveAllIfExists(path string) error {
	if !Exists(path) {
		return nil
	}
	return os.RemoveAll(path)
}

func WalkFind(dir string, match func(path string, d fs.DirEntry) bool) (string, error) {
	var found string
	errStop := errors.New("stop")
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if match(path, d) {
			found = path
			return errStop
		}
		return nil
	})
	if err != nil && !errors.Is(err, errStop) {
		return "", err
	}
	return found, nil
}
