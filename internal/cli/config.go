package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/xcbolt/xcbolt/internal/core"
)

func newConfigCmd() *cobra.Command {
	var edit bool
	var migrate bool

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show or edit the xcbolt config",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := NewAppContext(flags)
			if err != nil {
				if !migrate {
					return err
				}
				root, rerr := resolveProjectRoot(flags.Project)
				if rerr != nil {
					return err
				}
				cfgPath := flags.Config
				if cfgPath == "" {
					cfgPath = core.ConfigPath(root)
				}
				emit := core.Emitter(core.NewTextEmitter(cmd.OutOrStdout()))
				if flags.JSON {
					if flags.EventVersion != core.EventSchemaVersion {
						return fmt.Errorf("unsupported --event-version %d (supported: %d)", flags.EventVersion, core.EventSchemaVersion)
					}
					emit = core.NewNDJSONEmitter(cmd.OutOrStdout(), flags.EventVersion)
				}
				ac = AppContext{
					ProjectRoot: root,
					ConfigPath:  cfgPath,
					Emitter:     emit,
					Flags:       flags,
				}
			}

			if migrate {
				res, err := core.MigrateConfig(ac.ProjectRoot, ac.ConfigPath)
				if err != nil {
					return err
				}
				if ac.Flags.JSON {
					ac.Emitter.Emit(core.Event{Cmd: "config", Type: "config_migrated", Data: res})
					return nil
				}
				if res.BackupPath != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "Migrated config from v%d to v%d: %s (backup: %s)\n", res.FromVersion, res.ToVersion, res.Path, res.BackupPath)
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "Config already at v%d: %s\n", res.ToVersion, res.Path)
				}
				return nil
			}

			if edit {
				if err := core.SaveConfig(ac.ProjectRoot, ac.ConfigPath, ac.Config); err != nil {
					return err
				}
				editor := os.Getenv("EDITOR")
				if editor == "" {
					return errors.New("EDITOR is not set; export EDITOR or run without --edit to print config")
				}
				if err := exec.Command(editor, ac.ConfigPath).Start(); err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Opened", ac.ConfigPath, "in", editor)
				return nil
			}

			b, _ := json.MarshalIndent(ac.Config, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		},
	}

	cmd.Flags().BoolVar(&edit, "edit", false, "Open config in $EDITOR")
	cmd.Flags().BoolVar(&migrate, "migrate", false, "Migrate config to the latest schema version")
	return cmd
}
