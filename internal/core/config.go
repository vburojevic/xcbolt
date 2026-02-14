package core

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const ConfigVersion = 2

type ConfigVersionError struct {
	Path string
	Got  int
	Want int
}

func (e ConfigVersionError) Error() string {
	return fmt.Sprintf("config version mismatch for %s: got v%d, expected v%d (run `xcbolt init` to regenerate config)", e.Path, e.Got, e.Want)
}

type DestinationKind string

const (
	DestAuto      DestinationKind = "auto"
	DestSimulator DestinationKind = "simulator"
	DestDevice    DestinationKind = "device"
	DestMacOS     DestinationKind = "macos"
	DestCatalyst  DestinationKind = "catalyst"
)

type Destination struct {
	Kind              DestinationKind `json:"kind"`
	UDID              string          `json:"udid,omitempty"` // legacy alias for ID
	Name              string          `json:"name,omitempty"`
	Platform          string          `json:"platform,omitempty"` // e.g. iOS, iOS Simulator, macOS
	OS                string          `json:"os,omitempty"`       // e.g. 17.2
	PlatformFamily    PlatformFamily  `json:"platformFamily,omitempty"`
	TargetType        TargetType      `json:"targetType,omitempty"`
	ID                string          `json:"id,omitempty"`
	RuntimeID         string          `json:"runtimeId,omitempty"`
	CompanionTargetID string          `json:"companionTargetId,omitempty"`
	CompanionBundleID string          `json:"companionBundleId,omitempty"`
}

type XcodebuildConfig struct {
	Options       []string          `json:"options,omitempty"`
	Env           map[string]string `json:"env,omitempty"`
	LogFormat     string            `json:"logFormat,omitempty"`
	LogFormatArgs []string          `json:"logFormatArgs,omitempty"`
	DryRun        bool              `json:"dryRun,omitempty"`
}

type LaunchConfig struct {
	Options           []string          `json:"options,omitempty"`
	Env               map[string]string `json:"env,omitempty"`
	StreamUnifiedLogs *bool             `json:"streamUnifiedLogs,omitempty"`
	StreamSystemLogs  *bool             `json:"streamSystemLogs,omitempty"`
	ConsoleLogLevels  map[string]bool   `json:"consoleLogLevels,omitempty"`
}

type TUIConfig struct {
	ShowAllLogs bool `json:"showAllLogs,omitempty"`
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
	TUI        TUIConfig        `json:"tui,omitempty"`
}

func DefaultConfig(projectRoot string) Config {
	streamUnified := true
	streamSystem := false
	consoleLevels := map[string]bool{
		"D": true,
		"I": true,
		"W": true,
		"E": true,
		"F": true,
	}
	return Config{
		Version:           ConfigVersion,
		Configuration:     "Debug",
		Destination:       Destination{Kind: DestAuto, TargetType: TargetAuto},
		DerivedDataPath:   filepath.Join(projectRoot, ".xcbolt", "DerivedData"),
		ResultBundlesPath: filepath.Join(projectRoot, ".xcbolt", "Results"),
		Xcodebuild:        XcodebuildConfig{Env: map[string]string{}, Options: []string{}, LogFormat: "auto", LogFormatArgs: []string{}},
		Launch:            LaunchConfig{Env: map[string]string{}, Options: []string{}, StreamUnifiedLogs: &streamUnified, StreamSystemLogs: &streamSystem, ConsoleLogLevels: consoleLevels},
		TUI:               TUIConfig{ShowAllLogs: true},
	}
}

func ConfigPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".xcbolt", "config.json")
}

func EnsureProjectDirs(projectRoot string) error {
	dir := filepath.Join(projectRoot, ".xcbolt")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	return ensureXcboltGitignore(dir)
}

func ensureXcboltGitignore(xcboltDir string) error {
	path := filepath.Join(xcboltDir, ".gitignore")
	entries := []string{
		"DerivedData/",
		"Results/",
	}

	b, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			content := strings.Join(entries, "\n") + "\n"
			return os.WriteFile(path, []byte(content), 0o644)
		}
		return err
	}

	existing := string(b)
	missing := []string{}
	for _, entry := range entries {
		if !hasGitignoreLine(existing, entry) {
			missing = append(missing, entry)
		}
	}
	if len(missing) == 0 {
		return nil
	}
	if existing != "" && !strings.HasSuffix(existing, "\n") {
		existing += "\n"
	}
	existing += strings.Join(missing, "\n") + "\n"
	return os.WriteFile(path, []byte(existing), 0o644)
}

func hasGitignoreLine(content string, line string) bool {
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == line {
			return true
		}
	}
	return false
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
	if cfg.Version != ConfigVersion {
		return cfg, ConfigVersionError{Path: path, Got: cfg.Version, Want: ConfigVersion}
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
	syncDestinationLegacy(&cfg.Destination)
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
	syncDestinationLegacy(&cfg.Destination)
	cfg.Version = ConfigVersion
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	b = append(b, '\n')
	return os.WriteFile(path, b, 0o644)
}
