package core

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

const StateVersion = 2

// RecentProject tracks a recently used project
type RecentProject struct {
	Root      string `json:"root"`
	Workspace string `json:"workspace,omitempty"`
	Project   string `json:"project,omitempty"`
	Scheme    string `json:"scheme,omitempty"`
	UpdatedAt string `json:"updatedAt"`
}

// RecentCombo tracks a recently used scheme+destination combination
type RecentCombo struct {
	Scheme      string `json:"scheme"`
	Destination string `json:"destination"` // Display name (e.g., "iPhone 15 Pro")
	DestUDID    string `json:"destUdid"`    // UDID for selection
	DestKind    string `json:"destKind"`    // "simulator", "device", etc.
	UsedAt      string `json:"usedAt"`
}

// State persists user preferences across sessions
type State struct {
	Version int             `json:"version"`
	Recent  []RecentProject `json:"recent,omitempty"`

	// Per-project recent combos, keyed by project root
	Combos map[string][]RecentCombo `json:"combos,omitempty"`
}

const MaxRecentCombos = 5

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

// AddRecentCombo adds a scheme+destination combo to recents
func (st *State) AddRecentCombo(projectRoot string, combo RecentCombo) {
	if st.Combos == nil {
		st.Combos = make(map[string][]RecentCombo)
	}

	combos := st.Combos[projectRoot]

	// Remove existing entry for same scheme+destination
	filtered := make([]RecentCombo, 0, len(combos))
	for _, c := range combos {
		if c.Scheme != combo.Scheme || c.DestUDID != combo.DestUDID {
			filtered = append(filtered, c)
		}
	}

	// Add new entry at front
	filtered = append([]RecentCombo{combo}, filtered...)

	// Limit to max
	if len(filtered) > MaxRecentCombos {
		filtered = filtered[:MaxRecentCombos]
	}

	st.Combos[projectRoot] = filtered
}

// GetRecentCombos returns recent combos for a project
func (st *State) GetRecentCombos(projectRoot string) []RecentCombo {
	if st.Combos == nil {
		return nil
	}
	return st.Combos[projectRoot]
}
