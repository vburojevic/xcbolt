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
	var platform string
	var target string
	var targetType string
	var companionTarget string
	var console bool

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Build, install, and launch the app",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := NewAppContext(flags)
			if err != nil {
				return err
			}
			if err := applyOverrides(&ac.Config, scheme, configuration, simulator, device, platform, target, targetType, companionTarget, ac.Emitter); err != nil {
				return err
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			_, cfg2, err := core.Run(ctx, ac.ProjectRoot, ac.Config, console, ac.Emitter)
			// Persist auto-selected scheme/config for future runs.
			persistConfigIfChanged(ac, cfg2)
			return err
		},
	}

	cmd.Flags().StringVar(&scheme, "scheme", "", "Override scheme")
	cmd.Flags().StringVar(&configuration, "configuration", "", "Override configuration")
	cmd.Flags().StringVar(&simulator, "simulator", "", "Simulator UDID")
	cmd.Flags().StringVar(&device, "device", "", "Device UDID")
	cmd.Flags().StringVar(&platform, "platform", "", "Destination platform family (ios|ipados|tvos|visionos|watchos|macos|catalyst)")
	cmd.Flags().StringVar(&target, "target", "", "Destination ID or exact name")
	cmd.Flags().StringVar(&targetType, "target-type", "", "Destination target type (simulator|device|local)")
	cmd.Flags().StringVar(&companionTarget, "companion-target", "", "Companion destination ID/name (watchOS physical runs)")
	cmd.Flags().BoolVar(&console, "console", false, "Attempt to stream app output (simctl --console / devicectl --console)")

	return cmd
}
