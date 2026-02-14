package core

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const SessionsVersion = 2

type Session struct {
	ID                string `json:"id"`
	BundleID          string `json:"bundleId"`
	PID               int    `json:"pid,omitempty"`
	Target            string `json:"target"` // simulator|device|macos
	UDID              string `json:"udid,omitempty"`
	PlatformFamily    string `json:"platformFamily,omitempty"`
	TargetType        string `json:"targetType,omitempty"`
	TargetID          string `json:"targetId,omitempty"`
	CompanionTargetID string `json:"companionTargetId,omitempty"`
	CompanionBundleID string `json:"companionBundleId,omitempty"`
	StartedAt         string `json:"startedAt"`
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
	if s.Version != SessionsVersion {
		// Hard-cutover for session schema: reset to empty state.
		return Sessions{Version: SessionsVersion, Items: []Session{}}, nil
	}
	return s, nil
}

func SaveSessions(projectRoot string, s Sessions) error {
	if err := EnsureProjectDirs(projectRoot); err != nil {
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
	dst := Destination{}
	switch target {
	case "simulator":
		dst.Kind = DestSimulator
		dst.TargetType = TargetSimulator
	case "device":
		dst.Kind = DestDevice
		dst.TargetType = TargetDevice
	case "catalyst":
		dst.Kind = DestCatalyst
		dst.TargetType = TargetLocal
		dst.PlatformFamily = PlatformCatalyst
	case "macos":
		dst.Kind = DestMacOS
		dst.TargetType = TargetLocal
		dst.PlatformFamily = PlatformMacOS
	}
	dst.UDID = udid
	dst.ID = udid
	return AddSessionWithDestination(projectRoot, bundleID, pid, dst)
}

func AddSessionWithDestination(projectRoot string, bundleID string, pid int, dst Destination) (Session, error) {
	s, err := LoadSessions(projectRoot)
	if err != nil {
		return Session{}, err
	}
	dst = normalizeDestination(dst)
	target := string(dst.Kind)
	if target == "" || dst.Kind == DestAuto {
		target = string(dst.TargetType)
	}
	udid := strings.TrimSpace(dst.ID)
	if udid == "" {
		udid = strings.TrimSpace(dst.UDID)
	}
	id := bundleID
	if udid != "" {
		id = id + "@" + udid
	}
	sess := Session{
		ID:                id,
		BundleID:          bundleID,
		PID:               pid,
		Target:            target,
		UDID:              udid,
		TargetID:          udid,
		PlatformFamily:    string(dst.PlatformFamily),
		TargetType:        string(dst.TargetType),
		CompanionTargetID: dst.CompanionTargetID,
		CompanionBundleID: dst.CompanionBundleID,
		StartedAt:         time.Now().UTC().Format(time.RFC3339Nano),
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
