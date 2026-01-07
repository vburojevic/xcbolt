package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const ConfigVersion = 1

type DestinationKind string

const (
	DestAuto      DestinationKind = "auto"
	DestSimulator DestinationKind = "simulator"
	DestDevice    DestinationKind = "device"
	DestMacOS     DestinationKind = "macos"
	DestCatalyst  DestinationKind = "catalyst"
)

type Destination struct {
	Kind     DestinationKind `json:"kind"`
	UDID     string          `json:"udid,omitempty"`
	Name     string          `json:"name,omitempty"`
	Platform string          `json:"platform,omitempty"` // e.g. iOS, iOS Simulator, macOS
	OS       string          `json:"os,omitempty"`       // e.g. 17.2
}

type XcodebuildConfig struct {
	Options       []string          `json:"options,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	LogFormat     string            `json:"logFormat,omitempty"`
	LogFormatArgs []string          `json:"logFormatArgs,omitempty"`
}

type LaunchConfig struct {
	Options []string          `json:"options,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

type Config struct {
	Version int `json:"version"`

	Workspace string `json:"workspace,omitempty"`
	Project   string `json:"project,omitempty"`
	Scheme    string `json:"scheme,omitempty"`

	Configuration string `json:"configuration,omitempty"`

	Destination Destination `json:"destination"`

	DerivedDataPath    string `json:"derivedDataPath,omitempty"`
	ResultBundlesPath  string `json:"resultBundlesPath,omitempty"`
	LastResultBundle   string `json:"-"`
	LastBuiltAppBundle string `json:"-"`

	Xcodebuild XcodebuildConfig `json:"xcodebuild,omitempty"`
	Launch     LaunchConfig     `json:"launch,omitempty"`
}

func DefaultConfig(projectRoot string) Config {
	return Config{
		Version:           ConfigVersion,
		Configuration:     "Debug",
		Destination:       Destination{Kind: DestAuto},
		DerivedDataPath:   filepath.Join(projectRoot, ".xcbolt", "DerivedData"),
		ResultBundlesPath: filepath.Join(projectRoot, ".xcbolt", "Results"),
		Xcodebuild:        XcodebuildConfig{Env: map[string]string{}, Options: []string{}, LogFormat: "auto", LogFormatArgs: []string{}},
		Launch:            LaunchConfig{Env: map[string]string{}, Options: []string{}},
	}
}

func ConfigPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".xcbolt", "config.json")
}

func EnsureProjectDirs(projectRoot string) error {
	return os.MkdirAll(filepath.Join(projectRoot, ".xcbolt"), 0o755)
}

func LoadConfig(projectRoot string, overridePath string) (Config, error) {
	cfg := DefaultConfig(projectRoot)

	path := overridePath
	if path == "" {
		path = ConfigPath(projectRoot)
	}

	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, err
	}
	if err := json.Unmarshal(b, &cfg); err != nil {
		return cfg, fmt.Errorf("failed to parse config %s: %w", path, err)
	}
	if cfg.Version == 0 {
		cfg.Version = ConfigVersion
	}
	// Ensure defaults for computed paths if missing.
	if cfg.DerivedDataPath == "" {
		cfg.DerivedDataPath = filepath.Join(projectRoot, ".xcbolt", "DerivedData")
	}
	if cfg.ResultBundlesPath == "" {
		cfg.ResultBundlesPath = filepath.Join(projectRoot, ".xcbolt", "Results")
	}
	if cfg.Xcodebuild.Env == nil {
		cfg.Xcodebuild.Env = map[string]string{}
	}
	if cfg.Xcodebuild.LogFormat == "" {
		cfg.Xcodebuild.LogFormat = "auto"
	}
	if cfg.Launch.Env == nil {
		cfg.Launch.Env = map[string]string{}
	}
	return cfg, nil
}

func SaveConfig(projectRoot string, overridePath string, cfg Config) error {
	if err := EnsureProjectDirs(projectRoot); err != nil {
		return err
	}
	path := overridePath
	if path == "" {
		path = ConfigPath(projectRoot)
	}
	cfg.Version = ConfigVersion
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}
