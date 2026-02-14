package cli

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"github.com/xcbolt/xcbolt/internal/core"
)

func newInitCmd() *cobra.Command {
	var nonInteractive bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize project configuration (.xcbolt/config.json)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := NewAppContext(flags)
			if err != nil {
				// Allow init to recover from schema cutovers.
				root, rerr := resolveProjectRoot(flags.Project)
				if rerr != nil {
					return err
				}
				cfg := core.DefaultConfig(root)
				if flags.LogFormat != "" {
					cfg.Xcodebuild.LogFormat = flags.LogFormat
				}
				if len(flags.LogFormatArgs) > 0 {
					cfg.Xcodebuild.LogFormatArgs = flags.LogFormatArgs
				}
				emit := core.Emitter(core.NewTextEmitter(cmd.OutOrStdout()))
				if flags.JSON {
					emit = core.NewNDJSONEmitter(cmd.OutOrStdout())
				}
				var verr core.ConfigVersionError
				if errors.As(err, &verr) {
					emit.Emit(core.Warn("init", err.Error()))
					cfg = core.DefaultConfig(root)
					cfgPath := flags.Config
					if cfgPath == "" {
						cfgPath = core.ConfigPath(root)
					}
					ac = AppContext{
						ProjectRoot: root,
						ConfigPath:  cfgPath,
						Config:      cfg,
						Emitter:     emit,
						Flags:       flags,
					}
				} else {
					return err
				}
			}
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			ac.Emitter.Emit(core.Status("init", "Loading project context…", nil))
			info, cfg, err := core.DiscoverContext(ctx, ac.ProjectRoot, ac.Config, ac.Emitter, core.ContextOptions{
				UseXcodebuildList:     ac.Flags.UseXcodebuildList,
				AllowXcodebuildList:   true,
				XcodebuildListTimeout: 5 * time.Second,
			})
			if err != nil {
				return err
			}

			if nonInteractive || ac.Flags.JSON {
				// Non-interactive best-effort defaults.
				cfg = pickInitDefaults(info, cfg)
				if nonInteractive {
					if cfg.Workspace == "" && cfg.Project == "" {
						return ExitError{Code: 2, Err: fmt.Errorf("no workspace/project detected")}
					}
					if cfg.Scheme == "" {
						return ExitError{Code: 3, Err: fmt.Errorf("no scheme detected")}
					}
					if cfg.Configuration == "" {
						return ExitError{Code: 4, Err: fmt.Errorf("no build configuration detected")}
					}
				}
				if err := core.SaveConfig(ac.ProjectRoot, ac.ConfigPath, cfg); err != nil {
					return err
				}
				ac.Emitter.Emit(core.Result("init", true, map[string]any{"config": ac.ConfigPath}))
				return nil
			}

			cfg, err = runInitWizard(info, cfg)
			if err != nil {
				return err
			}

			if err := core.SaveConfig(ac.ProjectRoot, ac.ConfigPath, cfg); err != nil {
				return err
			}

			ac.Emitter.Emit(core.Status("init", "Wrote config", map[string]any{"path": ac.ConfigPath}))
			return nil
		},
	}
	cmd.Flags().BoolVar(&nonInteractive, "non-interactive", false, "Use defaults without prompts (CI-friendly)")
	return cmd
}

func pickInitDefaults(info core.ContextInfo, cfg core.Config) core.Config {
	// workspace > project
	if cfg.Workspace == "" && len(info.Workspaces) > 0 {
		cfg.Workspace = filepath.Base(info.Workspaces[0])
	}
	if cfg.Project == "" && cfg.Workspace == "" && len(info.Projects) > 0 {
		cfg.Project = filepath.Base(info.Projects[0])
	}
	if cfg.Scheme == "" && len(info.Schemes) > 0 {
		cfg.Scheme = info.Schemes[0]
	}
	if cfg.Configuration == "" && len(info.Configurations) > 0 {
		cfg.Configuration = info.Configurations[0]
	}
	if cfg.Configuration == "" {
		cfg.Configuration = "Debug"
	}
	if cfg.Destination.Kind == core.DestAuto {
		// leave auto; core will resolve to best simulator.
	}
	return cfg
}

func runInitWizard(info core.ContextInfo, cfg core.Config) (core.Config, error) {
	// Choices
	projChoice := ""
	scheme := cfg.Scheme
	conf := cfg.Configuration
	destKind := string(cfg.Destination.Kind)
	if destKind == "" {
		destKind = string(core.DestSimulator)
	}

	projOpts := []huh.Option[string]{}
	for _, w := range info.Workspaces {
		projOpts = append(projOpts, huh.NewOption("Workspace: "+w, "workspace:"+w))
	}
	for _, p := range info.Projects {
		projOpts = append(projOpts, huh.NewOption("Project: "+p, "project:"+p))
	}
	if len(projOpts) == 0 {
		projOpts = append(projOpts, huh.NewOption("(No workspace/project detected)", ""))
	}

	schemeOpts := []huh.Option[string]{}
	for _, s := range info.Schemes {
		schemeOpts = append(schemeOpts, huh.NewOption(s, s))
	}
	if len(schemeOpts) == 0 {
		schemeOpts = append(schemeOpts, huh.NewOption("(No schemes detected)", ""))
	}

	confOpts := initConfigOptions(info.Configurations, conf)

	kindOpts := []huh.Option[string]{
		huh.NewOption("Simulator", string(core.DestSimulator)),
		huh.NewOption("Device", string(core.DestDevice)),
		huh.NewOption("macOS", string(core.DestMacOS)),
		huh.NewOption("Mac Catalyst", string(core.DestCatalyst)),
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Workspace / Project").
				Options(projOpts...).
				Value(&projChoice),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Scheme").
				Options(schemeOpts...).
				Value(&scheme),
			huh.NewSelect[string]().
				Title("Configuration").
				Options(confOpts...).
				Value(&conf),
			huh.NewSelect[string]().
				Title("Destination").
				Options(kindOpts...).
				Value(&destKind),
		),
	)
	if err := form.Run(); err != nil {
		return cfg, err
	}

	if conf == "__other__" {
		var custom string
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewInput().Title("Configuration name").Value(&custom),
			),
		).Run(); err != nil {
			return cfg, err
		}
		if strings.TrimSpace(custom) != "" {
			conf = strings.TrimSpace(custom)
		} else {
			conf = "Debug"
		}
	}

	// Destination details
	udid := ""
	name := ""
	platformFamily := core.PlatformUnknown
	targetType := core.TargetAuto
	runtimeID := ""
	switch core.DestinationKind(destKind) {
	case core.DestSimulator:
		targetType = core.TargetSimulator
		simOpts := []huh.Option[string]{}
		for _, s := range info.Simulators {
			if !s.Available {
				continue
			}
			label := fmt.Sprintf("%s (%s) [%s]", s.Name, s.RuntimeName, strings.ToLower(s.State))
			simOpts = append(simOpts, huh.NewOption(label, s.UDID))
		}
		if len(simOpts) == 0 {
			simOpts = append(simOpts, huh.NewOption("(No simulators available)", ""))
		}
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().Title("Simulator").Options(simOpts...).Value(&udid),
			),
		).Run(); err != nil {
			return cfg, err
		}
		for _, s := range info.Simulators {
			if s.UDID == udid {
				name = s.Name
				platformFamily = s.PlatformFamily
				cfg.Destination.Platform = core.PlatformStringForDestination(platformFamily, core.TargetSimulator)
				if cfg.Destination.Platform == "" {
					cfg.Destination.Platform = "iOS Simulator"
				}
				cfg.Destination.OS = s.OSVersion
				runtimeID = s.RuntimeID
				break
			}
		}
	case core.DestDevice:
		targetType = core.TargetDevice
		devOpts := []huh.Option[string]{}
		for _, d := range info.Devices {
			label := d.Name
			if d.OSVersion != "" {
				label = fmt.Sprintf("%s (iOS %s)", label, d.OSVersion)
			}
			devOpts = append(devOpts, huh.NewOption(label, d.Identifier))
		}
		if len(devOpts) == 0 {
			devOpts = append(devOpts, huh.NewOption("(No devices available)", ""))
		}
		if err := huh.NewForm(
			huh.NewGroup(
				huh.NewSelect[string]().Title("Device").Options(devOpts...).Value(&udid),
			),
		).Run(); err != nil {
			return cfg, err
		}
		for _, d := range info.Devices {
			if d.Identifier == udid {
				name = d.Name
				platformFamily = d.PlatformFamily
				cfg.Destination.Platform = core.PlatformStringForDestination(platformFamily, core.TargetDevice)
				if cfg.Destination.Platform == "" {
					cfg.Destination.Platform = "iOS"
				}
				cfg.Destination.OS = d.OSVersion
				break
			}
		}
	case core.DestMacOS:
		targetType = core.TargetLocal
		platformFamily = core.PlatformMacOS
		name = "My Mac"
		cfg.Destination.Platform = "macOS"
		cfg.Destination.OS = "macOS"
	case core.DestCatalyst:
		targetType = core.TargetLocal
		platformFamily = core.PlatformCatalyst
		name = "My Mac (Catalyst)"
		cfg.Destination.Platform = "macOS"
		cfg.Destination.OS = "macOS"
	}

	// Apply to config
	cfg.Scheme = scheme
	cfg.Configuration = conf
	cfg.Destination.Kind = core.DestinationKind(destKind)
	cfg.Destination.UDID = strings.TrimSpace(udid)
	cfg.Destination.ID = strings.TrimSpace(udid)
	cfg.Destination.Name = name
	cfg.Destination.TargetType = targetType
	cfg.Destination.PlatformFamily = platformFamily
	cfg.Destination.RuntimeID = runtimeID

	if strings.HasPrefix(projChoice, "workspace:") {
		cfg.Workspace = strings.TrimPrefix(projChoice, "workspace:")
		cfg.Project = ""
	} else if strings.HasPrefix(projChoice, "project:") {
		cfg.Project = strings.TrimPrefix(projChoice, "project:")
		cfg.Workspace = ""
	}

	return cfg, nil
}

func initConfigOptions(configs []string, current string) []huh.Option[string] {
	seen := map[string]struct{}{}
	list := []string{}
	add := func(v string) {
		v = strings.TrimSpace(v)
		if v == "" {
			return
		}
		if _, ok := seen[v]; ok {
			return
		}
		seen[v] = struct{}{}
		list = append(list, v)
	}

	if len(configs) == 0 {
		add("Debug")
		add("Release")
	} else {
		for _, c := range configs {
			add(c)
		}
	}
	add(current)

	opts := make([]huh.Option[string], 0, len(list)+1)
	for _, c := range list {
		opts = append(opts, huh.NewOption(c, c))
	}
	opts = append(opts, huh.NewOption("Other…", "__other__"))
	return opts
}
