package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

type XcodeListInfo struct {
	Schemes        []string `json:"schemes"`
	Configurations []string `json:"configurations"`
	Name           string   `json:"name"`
}

type xcodebuildListJSON struct {
	Project   *XcodeListInfo `json:"project,omitempty"`
	Workspace *XcodeListInfo `json:"workspace,omitempty"`
}

// XcodebuildList returns schemes/configurations for a workspace or project using `xcodebuild -list -json`.
func XcodebuildList(ctx context.Context, projectRoot string, cfg Config, emit Emitter) (XcodeListInfo, error) {
	args := []string{"xcodebuild", "-list", "-json"}
	if cfg.Workspace != "" {
		args = append(args, "-workspace", filepath.Join(projectRoot, cfg.Workspace))
	} else if cfg.Project != "" {
		args = append(args, "-project", filepath.Join(projectRoot, cfg.Project))
	} else {
		return XcodeListInfo{}, errors.New("no workspace/project configured")
	}

	var out strings.Builder
	res, err := RunStreaming(ctx, CmdSpec{
		Path: "xcrun",
		Args: args,
		Dir:  projectRoot,
		StdoutLine: func(s string) {
			out.WriteString(s)
			out.WriteString("\n")
		},
		StderrLine: func(s string) {
			if emit != nil {
				emit.Emit(Log("context", s))
			}
		},
	})
	_ = res
	if err != nil {
		return XcodeListInfo{}, err
	}
	b := []byte(out.String())
	var parsed xcodebuildListJSON
	if err := json.Unmarshal(b, &parsed); err != nil {
		// Some Xcode versions may emit extra prefixes; attempt to extract JSON object.
		trim := extractJSONObject(out.String())
		if trim == "" {
			return XcodeListInfo{}, fmt.Errorf("failed to parse xcodebuild -list -json output: %w", err)
		}
		if err := json.Unmarshal([]byte(trim), &parsed); err != nil {
			return XcodeListInfo{}, fmt.Errorf("failed to parse xcodebuild -list -json output: %w", err)
		}
	}

	var info XcodeListInfo
	if parsed.Workspace != nil {
		info = *parsed.Workspace
	}
	if parsed.Project != nil {
		if info.Name == "" {
			info.Name = parsed.Project.Name
		}
		if len(info.Schemes) == 0 {
			info.Schemes = parsed.Project.Schemes
		}
		if len(info.Configurations) == 0 {
			info.Configurations = parsed.Project.Configurations
		}
	}
	return info, nil
}

var jsonObjectRE = regexp.MustCompile(`(?s)\{.*\}`)

func extractJSONObject(s string) string {
	m := jsonObjectRE.FindString(s)
	return strings.TrimSpace(m)
}

// BuildDestinationString builds an xcodebuild -destination string from config.
func BuildDestinationString(cfg Config) string {
	switch cfg.Destination.Kind {
	case DestSimulator:
		udid := cfg.Destination.UDID
		if udid == "" {
			return ""
		}
		// Prefer explicit platform string; fall back to iOS Simulator.
		platform := cfg.Destination.Platform
		if platform == "" {
			platform = "iOS Simulator"
		}
		return fmt.Sprintf("platform=%s,id=%s", platform, udid)
	case DestDevice:
		udid := cfg.Destination.UDID
		if udid == "" {
			return ""
		}
		platform := cfg.Destination.Platform
		if platform == "" {
			platform = "iOS"
		}
		return fmt.Sprintf("platform=%s,id=%s", platform, udid)
	case DestMacOS:
		return "platform=macOS"
	case DestCatalyst:
		return "platform=macOS,variant=Mac Catalyst"
	default:
		return ""
	}
}

type BuildSettings map[string]string

// ShowBuildSettings runs `xcodebuild -showBuildSettings` and returns a map.
func ShowBuildSettings(ctx context.Context, projectRoot string, cfg Config) (BuildSettings, error) {
	args := []string{"xcodebuild", "-showBuildSettings"}
	if cfg.Workspace != "" {
		args = append(args, "-workspace", filepath.Join(projectRoot, cfg.Workspace))
	} else if cfg.Project != "" {
		args = append(args, "-project", filepath.Join(projectRoot, cfg.Project))
	}
	if cfg.Scheme != "" {
		args = append(args, "-scheme", cfg.Scheme)
	}
	if cfg.Configuration != "" {
		args = append(args, "-configuration", cfg.Configuration)
	}
	if dest := BuildDestinationString(cfg); dest != "" {
		args = append(args, "-destination", dest)
	}
	if cfg.DerivedDataPath != "" {
		args = append(args, "-derivedDataPath", cfg.DerivedDataPath)
	}

	var lines []string
	_, err := RunStreaming(ctx, CmdSpec{
		Path:       "xcrun",
		Args:       args,
		Dir:        projectRoot,
		StdoutLine: func(s string) { lines = append(lines, s) },
	})
	if err != nil {
		return nil, err
	}

	settings := BuildSettings{}
	for _, ln := range lines {
		// Expected: "    KEY = VALUE"
		if !strings.Contains(ln, "=") {
			continue
		}
		parts := strings.SplitN(strings.TrimSpace(ln), "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		settings[k] = v
	}
	return settings, nil
}
