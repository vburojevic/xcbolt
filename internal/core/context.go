package core

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"
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

type ContextOptions struct {
	UseXcodebuildList     bool
	AllowXcodebuildList   bool
	XcodebuildListTimeout time.Duration
}

func DiscoverContext(ctx context.Context, projectRoot string, cfg Config, emit Emitter, opts ContextOptions) (ContextInfo, Config, error) {
	emitMaybe(emit, Status("context", "Scanning project root", map[string]any{"path": projectRoot}))
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

	// Schemes/configurations via filesystem (fast, no xcodebuild).
	emitMaybe(emit, Status("context", "Reading schemes/configurations from filesystem", nil))
	schemes := listSchemesFromFS(projectRoot, cfg, projects)
	configurations := listConfigurationsFromPBXProj(projectRoot, cfg, projects)

	// Optionally use xcodebuild -list -json (slow) when requested or as fallback.
	useXcodebuild := opts.UseXcodebuildList || (opts.AllowXcodebuildList && (len(schemes) == 0 || len(configurations) == 0))
	if useXcodebuild && (cfg.Workspace != "" || cfg.Project != "") {
		listCtx := ctx
		cancel := func() {}
		timeout := opts.XcodebuildListTimeout
		if timeout <= 0 {
			timeout = 5 * time.Second
		}
		if timeout > 0 {
			listCtx, cancel = context.WithTimeout(ctx, timeout)
		}
		emitMaybe(emit, Status("context", "Running xcodebuild -list for schemes/configurations", map[string]any{"timeout": timeout.String()}))
		list, err := XcodebuildList(listCtx, projectRoot, cfg, emit)
		cancel()
		if err == nil {
			if len(schemes) == 0 {
				schemes = list.Schemes
			}
			if len(configurations) == 0 {
				configurations = list.Configurations
			}
			if cfg.Scheme == "" && len(list.Schemes) == 1 {
				cfg.Scheme = list.Schemes[0]
			}
			if cfg.Configuration == "" && len(list.Configurations) == 1 {
				cfg.Configuration = list.Configurations[0]
			}
		} else {
			emitMaybe(emit, Warn("context", "Could not list schemes/configurations via xcodebuild: "+err.Error()))
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
