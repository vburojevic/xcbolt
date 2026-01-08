package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/xcbolt/xcbolt/internal/core"
)

type wizardDoneMsg struct {
	cfg     core.Config
	aborted bool
	err     error
}

type wizardModel struct {
	info core.ContextInfo
	cfg  core.Config

	projectChoice string
	scheme        string
	configuration string
	destKind      string
	targetUDID    string

	form *huh.Form
}

func newWizard(info core.ContextInfo, cfg core.Config, width int) wizardModel {
	w := wizardModel{
		info: info,
		cfg:  cfg,
	}

	// Defaults
	if cfg.Workspace != "" {
		w.projectChoice = "workspace:" + cfg.Workspace
	} else if cfg.Project != "" {
		w.projectChoice = "project:" + cfg.Project
	}
	w.scheme = cfg.Scheme
	w.configuration = cfg.Configuration
	w.destKind = string(cfg.Destination.Kind)
	if w.destKind == "" || w.destKind == string(core.DestAuto) {
		w.destKind = string(core.DestSimulator)
	}
	w.targetUDID = cfg.Destination.UDID

	projOpts := []huh.Option[string]{}
	for _, ws := range info.Workspaces {
		projOpts = append(projOpts, huh.NewOption("Workspace: "+ws, "workspace:"+ws))
	}
	for _, pr := range info.Projects {
		projOpts = append(projOpts, huh.NewOption("Project: "+pr, "project:"+pr))
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

	confList := normalizeConfigurations(info.Configurations, w.configuration)
	confOpts := make([]huh.Option[string], 0, len(confList))
	for _, c := range confList {
		confOpts = append(confOpts, huh.NewOption(c, c))
	}

	kindOpts := []huh.Option[string]{
		huh.NewOption("Simulator", string(core.DestSimulator)),
		huh.NewOption("Device", string(core.DestDevice)),
		huh.NewOption("macOS", string(core.DestMacOS)),
		huh.NewOption("Mac Catalyst", string(core.DestCatalyst)),
	}

	// Dynamic target select based on destination kind.
	targetTitle := func() string {
		switch core.DestinationKind(w.destKind) {
		case core.DestDevice:
			return "Device"
		case core.DestSimulator:
			return "Simulator"
		case core.DestMacOS, core.DestCatalyst:
			return "Target"
		default:
			return "Target"
		}
	}
	targetOptions := func() []huh.Option[string] {
		switch core.DestinationKind(w.destKind) {
		case core.DestDevice:
			opts := []huh.Option[string]{}
			for _, d := range info.Devices {
				label := d.Name
				if d.OSVersion != "" {
					label = fmt.Sprintf("%s (iOS %s)", label, d.OSVersion)
				}
				opts = append(opts, huh.NewOption(label, d.Identifier))
			}
			if len(opts) == 0 {
				opts = append(opts, huh.NewOption("(No devices found)", ""))
			}
			return opts
		case core.DestSimulator:
			opts := []huh.Option[string]{}
			for _, s := range info.Simulators {
				if !s.Available {
					continue
				}
				label := fmt.Sprintf("%s (%s) [%s]", s.Name, s.RuntimeName, strings.ToLower(s.State))
				opts = append(opts, huh.NewOption(label, s.UDID))
			}
			if len(opts) == 0 {
				opts = append(opts, huh.NewOption("(No simulators found)", ""))
			}
			return opts
		default:
			return []huh.Option[string]{huh.NewOption("(No target required)", "")}
		}
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Workspace / Project").
				Options(projOpts...).
				Value(&w.projectChoice),
			huh.NewSelect[string]().
				Title("Scheme").
				Options(schemeOpts...).
				Value(&w.scheme),
			huh.NewSelect[string]().
				Title("Configuration").
				Options(confOpts...).
				Value(&w.configuration),
		),
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Destination").
				Options(kindOpts...).
				Value(&w.destKind),
			huh.NewSelect[string]().
				TitleFunc(targetTitle, &w.destKind).
				OptionsFunc(targetOptions, &w.destKind).
				Value(&w.targetUDID),
		),
	).WithShowHelp(true)

	if width > 0 {
		form.WithWidth(width - 6)
	}
	w.form = form
	return w
}

func (w wizardModel) Init() tea.Cmd {
	return w.form.Init()
}

func (w wizardModel) Update(msg tea.Msg) (wizardModel, tea.Cmd) {
	m, cmd := w.form.Update(msg)
	if fm, ok := m.(*huh.Form); ok {
		w.form = fm
	}

	switch w.form.State {
	case huh.StateCompleted:
		cfg := w.cfg
		cfg.Scheme = strings.TrimSpace(w.scheme)
		cfg.Configuration = strings.TrimSpace(w.configuration)
		cfg.Destination.Kind = core.DestinationKind(w.destKind)
		cfg.Destination.UDID = strings.TrimSpace(w.targetUDID)

		if strings.HasPrefix(w.projectChoice, "workspace:") {
			cfg.Workspace = strings.TrimPrefix(w.projectChoice, "workspace:")
			cfg.Project = ""
		} else if strings.HasPrefix(w.projectChoice, "project:") {
			cfg.Project = strings.TrimPrefix(w.projectChoice, "project:")
			cfg.Workspace = ""
		}

		// Resolve target display name.
		switch cfg.Destination.Kind {
		case core.DestSimulator:
			for _, s := range w.info.Simulators {
				if s.UDID == cfg.Destination.UDID {
					cfg.Destination.Name = s.Name
					cfg.Destination.Platform = "iOS Simulator"
					cfg.Destination.OS = s.OSVersion
					break
				}
			}
		case core.DestDevice:
			for _, d := range w.info.Devices {
				if d.Identifier == cfg.Destination.UDID {
					cfg.Destination.Name = d.Name
					cfg.Destination.Platform = "iOS"
					cfg.Destination.OS = d.OSVersion
					break
				}
			}
		}

		return w, tea.Batch(cmd, func() tea.Msg { return wizardDoneMsg{cfg: cfg} })
	case huh.StateAborted:
		return w, tea.Batch(cmd, func() tea.Msg { return wizardDoneMsg{aborted: true} })
	default:
		return w, cmd
	}
}

func (w wizardModel) View() string {
	return w.form.View()
}
