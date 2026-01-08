package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
)

type ContextInfo struct {
	ProjectRoot    string      `json:"projectRoot"`
	Workspaces     []string    `json:"workspaces"`
	Projects       []string    `json:"projects"`
	Schemes        []string    `json:"schemes"`
	Configurations []string    `json:"configurations"`
	Simulators     []Simulator `json:"simulators"`
	Devices        []Device    `json:"devices"`
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

	// Schemes/configurations via xcodebuild -list -json
	schemes := []string{}
	configurations := []string{}
	if cfg.Workspace != "" || cfg.Project != "" {
		if list, err := XcodebuildList(ctx, projectRoot, cfg, emit); err == nil {
			schemes = list.Schemes
			configurations = list.Configurations
			if cfg.Scheme == "" && len(list.Schemes) == 1 {
				cfg.Scheme = list.Schemes[0]
			}
			if cfg.Configuration == "" && len(list.Configurations) == 1 {
				cfg.Configuration = list.Configurations[0]
			}
		} else {
			emitMaybe(emit, Warn("context", "Could not list schemes/configurations: "+err.Error()))
		}
	}

	// If workspace was chosen but configs weren't found, try a single project as fallback.
	if len(configurations) == 0 && cfg.Workspace != "" && cfg.Project == "" && len(projects) == 1 {
		tmpCfg := cfg
		tmpCfg.Workspace = ""
		tmpCfg.Project = projects[0]
		if list, err := XcodebuildList(ctx, projectRoot, tmpCfg, emit); err == nil {
			configurations = list.Configurations
			if cfg.Configuration == "" && len(list.Configurations) == 1 {
				cfg.Configuration = list.Configurations[0]
			}
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
		ProjectRoot:    projectRoot,
		Workspaces:     workspaces,
		Projects:       projects,
		Schemes:        schemes,
		Configurations: configurations,
		Simulators:     simulators,
		Devices:        devices,
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
