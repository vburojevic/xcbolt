package cli

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/xcbolt/xcbolt/internal/core"
)

func newBuildCmd() *cobra.Command {
	var scheme string
	var configuration string
	var simulator string
	var device string
	var platform string
	var target string
	var targetType string
	var companionTarget string

	cmd := &cobra.Command{
		Use:   "build",
		Short: "Build the configured scheme",
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

			_, cfg2, err := core.Build(ctx, ac.ProjectRoot, ac.Config, ac.Emitter)
			persistConfigIfChanged(ac, cfg2)
			return err
		},
	}

	cmd.Flags().StringVar(&scheme, "scheme", "", "Override scheme")
	cmd.Flags().StringVar(&configuration, "configuration", "", "Override configuration (Debug/Release/...)")
	cmd.Flags().StringVar(&simulator, "simulator", "", "Simulator UDID (sets destination kind to simulator)")
	cmd.Flags().StringVar(&device, "device", "", "Device UDID (sets destination kind to device)")
	cmd.Flags().StringVar(&platform, "platform", "", "Destination platform family (ios|ipados|tvos|visionos|watchos|macos|catalyst)")
	cmd.Flags().StringVar(&target, "target", "", "Destination ID or exact name")
	cmd.Flags().StringVar(&targetType, "target-type", "", "Destination target type (simulator|device|local)")
	cmd.Flags().StringVar(&companionTarget, "companion-target", "", "Companion destination ID/name (watchOS physical runs)")

	return cmd
}

func applyOverrides(cfg *core.Config, scheme, configuration, simulator, device, platform, target, targetType, companionTarget string, emit core.Emitter) error {
	if scheme != "" {
		cfg.Scheme = scheme
	}
	if configuration != "" {
		cfg.Configuration = configuration
	}

	if (simulator != "" || device != "") && (platform != "" || target != "" || targetType != "") {
		return fmt.Errorf("cannot combine --simulator/--device with --platform/--target/--target-type")
	}

	if simulator != "" {
		cfg.Destination.Kind = core.DestSimulator
		cfg.Destination.TargetType = core.TargetSimulator
		cfg.Destination.PlatformFamily = core.PlatformIOS
		cfg.Destination.UDID = simulator
		cfg.Destination.ID = simulator
		cfg.Destination.Platform = "iOS Simulator"
	}
	if device != "" {
		cfg.Destination.Kind = core.DestDevice
		cfg.Destination.TargetType = core.TargetDevice
		cfg.Destination.PlatformFamily = core.PlatformIOS
		cfg.Destination.UDID = device
		cfg.Destination.ID = device
		cfg.Destination.Platform = "iOS"
	}
	if platform != "" {
		pf := core.NormalizePlatformFamily(platform)
		if pf == core.PlatformUnknown {
			return fmt.Errorf("unknown --platform value %q", platform)
		}
		cfg.Destination.PlatformFamily = pf
	}
	if targetType != "" {
		tt := core.NormalizeTargetType(targetType)
		if tt == core.TargetAuto {
			return fmt.Errorf("unknown --target-type value %q", targetType)
		}
		cfg.Destination.TargetType = tt
	}
	if target != "" {
		target = strings.TrimSpace(target)
		cfg.Destination.ID = target
		cfg.Destination.UDID = target
		cfg.Destination.Name = target
	}
	if companionTarget != "" {
		cfg.Destination.CompanionTargetID = strings.TrimSpace(companionTarget)
	}

	if cfg.Destination.TargetType == core.TargetLocal {
		if cfg.Destination.PlatformFamily == core.PlatformCatalyst {
			cfg.Destination.Kind = core.DestCatalyst
			cfg.Destination.Name = "My Mac (Catalyst)"
		} else {
			if cfg.Destination.PlatformFamily == core.PlatformUnknown {
				cfg.Destination.PlatformFamily = core.PlatformMacOS
			}
			cfg.Destination.Kind = core.DestMacOS
			cfg.Destination.Name = "My Mac"
		}
		cfg.Destination.ID = ""
		cfg.Destination.UDID = ""
		cfg.Destination.Platform = "macOS"
		cfg.Destination.OS = "macOS"
	}

	if (simulator != "" || device != "") && emit != nil {
		emit.Emit(core.Warn("config", "--simulator/--device are deprecated; use --platform + --target + --target-type"))
	}
	return nil
}
