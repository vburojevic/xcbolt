package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

type ContextInfo struct {
	ProjectRoot string      `json:"projectRoot"`
	Workspaces  []string    `json:"workspaces"`
	Projects    []string    `json:"projects"`
	Schemes     []string    `json:"schemes"`
	Simulators  []Simulator `json:"simulators"`
	Devices     []Device    `json:"devices"`
}

func DiscoverContext(ctx context.Context, projectRoot string, cfg Config, emit Emitter) (ContextInfo, Config, error) {
	entries, err := os.ReadDir(projectRoot)
	if err != nil {
		return ContextInfo{}, cfg, err
	}
	workspaces := []string{}
	projects := []string{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() && strings.HasSuffix(name, ".xcworkspace") {
			workspaces = append(workspaces, name)
		}
		if e.IsDir() && strings.HasSuffix(name, ".xcodeproj") {
			projects = append(projects, name)
		}
	}

	// Auto-pick workspace/project if unset.
	if cfg.Workspace == "" && len(workspaces) == 1 {
		cfg.Workspace = workspaces[0]
	}
	if cfg.Project == "" && cfg.Workspace == "" && len(projects) == 1 {
		cfg.Project = projects[0]
	}

	// Schemes via xcodebuild -list -json
	schemes := []string{}
	if cfg.Workspace != "" || cfg.Project != "" {
		if sc, err := XcodebuildList(ctx, projectRoot, cfg, emit); err == nil {
			schemes = sc
			if cfg.Scheme == "" && len(sc) == 1 {
				cfg.Scheme = sc[0]
			}
		} else {
			emitMaybe(emit, Warn("context", "Could not list schemes: "+err.Error()))
		}
	}

	simulators := []Simulator{}
	if list, err := SimctlList(ctx, emit); err == nil {
		simulators = FlattenSimulators(list)
	} else {
		emitMaybe(emit, Warn("context", "Could not list simulators: "+err.Error()))
	}

	devices := []Device{}
	if DevicectlAvailable(ctx) {
		if devs, err := DevicectlList(ctx, emit); err == nil {
			devices = devs
		} else {
			emitMaybe(emit, Warn("context", "Could not list devices: "+err.Error()))
		}
	} else {
		emitMaybe(emit, Warn("context", "devicectl not available (install Xcode Command Line Tools / select Xcode)"))
	}

	info := ContextInfo{
		ProjectRoot: projectRoot,
		Workspaces:  workspaces,
		Projects:    projects,
		Schemes:     schemes,
		Simulators:  simulators,
		Devices:     devices,
	}
	return info, cfg, nil
}

func absJoin(root, maybeRel string) string {
	if maybeRel == "" {
		return ""
	}
	if filepath.IsAbs(maybeRel) {
		return maybeRel
	}
	return filepath.Join(root, maybeRel)
}
