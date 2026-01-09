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

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Show or edit the xcbolt config",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := NewAppContext(flags)
			if err != nil {
				return err
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
	return cmd
}
