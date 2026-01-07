package core

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const StateVersion = 1

type RecentProject struct {
	Root      string `json:"root"`
	Workspace string `json:"workspace,omitempty"`
	Project   string `json:"project,omitempty"`
	Scheme    string `json:"scheme,omitempty"`
	UpdatedAt string `json:"updatedAt"`
}

type State struct {
	Version int             `json:"version"`
	Recent  []RecentProject `json:"recent,omitempty"`
}

func defaultState() State { return State{Version: StateVersion, Recent: []RecentProject{}} }

func UserStatePath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "xcbolt", "state.json"), nil
}

func LoadState() (State, error) {
	st := defaultState()
	path, err := UserStatePath()
	if err != nil {
		return st, err
	}
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return st, nil
		}
		return st, err
	}
	if err := json.Unmarshal(b, &st); err != nil {
		return st, err
	}
	if st.Version == 0 {
		st.Version = StateVersion
	}
	return st, nil
}

func SaveState(st State) error {
	path, err := UserStatePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	st.Version = StateVersion
	b, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}
