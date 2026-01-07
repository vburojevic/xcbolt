package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xcbolt/xcbolt/internal/core"
)

func newAppsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apps",
		Short: "List apps launched by xcbolt (tracked sessions)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := NewAppContext(flags)
			if err != nil {
				return err
			}
			s, err := core.LoadSessions(ac.ProjectRoot)
			if err != nil {
				return err
			}
			if ac.Flags.JSON {
				ac.Emitter.Emit(core.Event{Cmd: "apps", Type: "apps", Data: s})
				return nil
			}
			b, _ := json.MarshalIndent(s, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		},
	}
	return cmd
}
