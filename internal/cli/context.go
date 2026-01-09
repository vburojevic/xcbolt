package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/xcbolt/xcbolt/internal/core"
)

func newContextCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "context",
		Short: "Discover project, schemes, simulators, and devices",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := NewAppContext(flags)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			info, cfg, err := core.DiscoverContext(ctx, ac.ProjectRoot, ac.Config, ac.Emitter, core.ContextOptions{
				UseXcodebuildList:     ac.Flags.UseXcodebuildList,
				AllowXcodebuildList:   true,
				XcodebuildListTimeout: 5 * time.Second,
			})
			if err != nil {
				return err
			}
			ac.Config = cfg

			if ac.Flags.JSON {
				ac.Emitter.Emit(core.Event{V: 1, TS: core.NowTS(), Cmd: "context", Type: "context", Data: info})
				return nil
			}

			b, _ := json.MarshalIndent(info, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		},
	}
	return cmd
}
