package cli

import (
	"fmt"
	"os"

	"github.com/xcbolt/xcbolt/internal/core"
	"github.com/xcbolt/xcbolt/internal/util"
)

type GlobalFlags struct {
	JSON    bool
	Config  string
	Project string
	Verbose bool
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
	emit := core.Emitter(core.NewTextEmitter(os.Stdout))
	if flags.JSON {
		emit = core.NewNDJSONEmitter(os.Stdout)
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
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
