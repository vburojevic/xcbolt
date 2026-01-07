package core

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

const SessionsVersion = 1

type Session struct {
	ID        string `json:"id"`
	BundleID  string `json:"bundleId"`
	PID       int    `json:"pid,omitempty"`
	Target    string `json:"target"` // simulator|device|macos
	UDID      string `json:"udid,omitempty"`
	StartedAt string `json:"startedAt"`
}

type Sessions struct {
	Version int       `json:"version"`
	Items   []Session `json:"items"`
}

func sessionsPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".xcbolt", "sessions.json")
}

func LoadSessions(projectRoot string) (Sessions, error) {
	path := sessionsPath(projectRoot)
	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Sessions{Version: SessionsVersion, Items: []Session{}}, nil
		}
		return Sessions{}, err
	}
	var s Sessions
	if err := json.Unmarshal(b, &s); err != nil {
		return Sessions{}, err
	}
	if s.Version == 0 {
		s.Version = SessionsVersion
	}
	return s, nil
}

func SaveSessions(projectRoot string, s Sessions) error {
	if err := os.MkdirAll(filepath.Join(projectRoot, ".xcbolt"), 0o755); err != nil {
		return err
	}
	path := sessionsPath(projectRoot)
	s.Version = SessionsVersion
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}

func AddSession(projectRoot string, bundleID string, pid int, target string, udid string) (Session, error) {
	s, err := LoadSessions(projectRoot)
	if err != nil {
		return Session{}, err
	}
	id := bundleID
	if udid != "" {
		id = id + "@" + udid
	}
	sess := Session{
		ID:        id,
		BundleID:  bundleID,
		PID:       pid,
		Target:    target,
		UDID:      udid,
		StartedAt: time.Now().UTC().Format(time.RFC3339Nano),
	}
	// Remove any existing with same ID
	filtered := make([]Session, 0, len(s.Items))
	for _, it := range s.Items {
		if it.ID != id {
			filtered = append(filtered, it)
		}
	}
	s.Items = append(filtered, sess)
	return sess, SaveSessions(projectRoot, s)
}

func RemoveSession(projectRoot string, id string) error {
	s, err := LoadSessions(projectRoot)
	if err != nil {
		return err
	}
	filtered := make([]Session, 0, len(s.Items))
	for _, it := range s.Items {
		if it.ID != id && it.BundleID != id {
			filtered = append(filtered, it)
		}
	}
	s.Items = filtered
	return SaveSessions(projectRoot, s)
}
