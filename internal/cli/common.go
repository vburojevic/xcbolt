package cli

import (
	"fmt"
	"os"

	"github.com/xcbolt/xcbolt/internal/core"
	"github.com/xcbolt/xcbolt/internal/util"
)

type GlobalFlags struct {
	JSON              bool
	EventVersion      int
	Config            string
	Project           string
	Verbose           bool
	LogFormat         string
	LogFormatArgs     []string
	UseXcodebuildList bool
}

func resolveProjectRoot(projectFlag string) (string, error) {
	start := projectFlag
	if start == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		start = wd
	}
	return util.FindProjectRoot(start)
}

type AppContext struct {
	ProjectRoot string
	ConfigPath  string
	Config      core.Config
	Emitter     core.Emitter
	Flags       GlobalFlags
}

func NewAppContext(flags GlobalFlags) (AppContext, error) {
	root, err := resolveProjectRoot(flags.Project)
	if err != nil {
		return AppContext{}, err
	}
	cfg, err := core.LoadConfig(root, flags.Config)
	if err != nil {
		return AppContext{}, err
	}
	if flags.LogFormat != "" {
		cfg.Xcodebuild.LogFormat = flags.LogFormat
	}
	if len(flags.LogFormatArgs) > 0 {
		cfg.Xcodebuild.LogFormatArgs = flags.LogFormatArgs
	}
	emit := core.Emitter(core.NewTextEmitter(os.Stdout))
	if flags.JSON {
		if flags.EventVersion != core.EventSchemaVersion {
			return AppContext{}, fmt.Errorf("unsupported --event-version %d (supported: %d)", flags.EventVersion, core.EventSchemaVersion)
		}
		emit = core.NewNDJSONEmitter(os.Stdout, flags.EventVersion)
	}
	cfgPath := flags.Config
	if cfgPath == "" {
		cfgPath = core.ConfigPath(root)
	}
	return AppContext{
		ProjectRoot: root,
		ConfigPath:  cfgPath,
		Config:      cfg,
		Emitter:     emit,
		Flags:       flags,
	}, nil
}

func PrintFatal(err error) {
	if ee, ok := err.(ExitError); ok {
		fmt.Fprintln(os.Stderr, ee.Error())
		os.Exit(ee.Code)
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

type ExitError struct {
	Code int
	Err  error
}

func (e ExitError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "command failed"
}

func persistConfigIfChanged(ac AppContext, cfg core.Config) {
	if cfg.Scheme == "" || cfg.Configuration == "" {
		return
	}
	if cfg.Scheme == ac.Config.Scheme &&
		cfg.Configuration == ac.Config.Configuration &&
		cfg.Workspace == ac.Config.Workspace &&
		cfg.Project == ac.Config.Project &&
		cfg.Destination.Kind == ac.Config.Destination.Kind &&
		cfg.Destination.UDID == ac.Config.Destination.UDID &&
		cfg.Destination.ID == ac.Config.Destination.ID &&
		cfg.Destination.PlatformFamily == ac.Config.Destination.PlatformFamily &&
		cfg.Destination.TargetType == ac.Config.Destination.TargetType {
		return
	}
	if err := core.SaveConfig(ac.ProjectRoot, ac.ConfigPath, cfg); err != nil {
		ac.Emitter.Emit(core.Warn("config", "Failed to save config: "+err.Error()))
	}
}
