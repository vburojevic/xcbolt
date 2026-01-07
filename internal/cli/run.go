package cli

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/xcbolt/xcbolt/internal/core"
)

func newRunCmd() *cobra.Command {
	var scheme string
	var configuration string
	var simulator string
	var device string
	var console bool

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Build, install, and launch the app",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := NewAppContext(flags)
			if err != nil {
				return err
			}
			applyOverrides(&ac.Config, scheme, configuration, simulator, device)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			_, _, err = core.Run(ctx, ac.ProjectRoot, ac.Config, console, ac.Emitter)
			return err
		},
	}

	cmd.Flags().StringVar(&scheme, "scheme", "", "Override scheme")
	cmd.Flags().StringVar(&configuration, "configuration", "", "Override configuration")
	cmd.Flags().StringVar(&simulator, "simulator", "", "Simulator UDID")
	cmd.Flags().StringVar(&device, "device", "", "Device UDID")
	cmd.Flags().BoolVar(&console, "console", false, "Attempt to stream app output (simctl --console / devicectl --console)")

	return cmd
}
