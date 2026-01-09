package cli

import (
	"context"

	"github.com/spf13/cobra"
	"github.com/xcbolt/xcbolt/internal/core"
)

func newBuildCmd() *cobra.Command {
	var scheme string
	var configuration string
	var simulator string
	var device string

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build the configured scheme",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := NewAppContext(flags)
			if err != nil {
				return err
			}
			applyOverrides(&ac.Config, scheme, configuration, simulator, device)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			_, cfg2, err := core.Build(ctx, ac.ProjectRoot, ac.Config, ac.Emitter)
			persistConfigIfChanged(ac, cfg2)
			return err
		},
	}

	cmd.Flags().StringVar(&scheme, "scheme", "", "Override scheme")
	cmd.Flags().StringVar(&configuration, "configuration", "", "Override configuration (Debug/Release/...)")
	cmd.Flags().StringVar(&simulator, "simulator", "", "Simulator UDID (sets destination kind to simulator)")
	cmd.Flags().StringVar(&device, "device", "", "Device UDID (sets destination kind to device)")

	return cmd
}

func applyOverrides(cfg *core.Config, scheme, configuration, simulator, device string) {
	if scheme != "" {
		cfg.Scheme = scheme
	}
	if configuration != "" {
		cfg.Configuration = configuration
	}
	if simulator != "" {
		cfg.Destination.Kind = core.DestSimulator
		cfg.Destination.UDID = simulator
		cfg.Destination.Platform = "iOS Simulator"
	}
	if device != "" {
		cfg.Destination.Kind = core.DestDevice
		cfg.Destination.UDID = device
		cfg.Destination.Platform = "iOS"
	}
}
