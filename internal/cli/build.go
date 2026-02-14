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
			if err := applyOverrides(&ac.Config, scheme, configuration, platform, target, targetType, companionTarget); err != nil {
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
	cmd.Flags().StringVar(&platform, "platform", "", "Destination platform family (ios|ipados|tvos|visionos|watchos|macos|catalyst)")
	cmd.Flags().StringVar(&target, "target", "", "Destination ID or exact name")
	cmd.Flags().StringVar(&targetType, "target-type", "", "Destination target type (simulator|device|local)")
	cmd.Flags().StringVar(&companionTarget, "companion-target", "", "Companion destination ID/name (watchOS physical runs)")

	return cmd
}

func applyOverrides(cfg *core.Config, scheme, configuration, platform, target, targetType, companionTarget string) error {
	if scheme != "" {
		cfg.Scheme = scheme
	}
	if configuration != "" {
		cfg.Configuration = configuration
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
		switch tt {
		case core.TargetSimulator:
			cfg.Destination.Kind = core.DestSimulator
		case core.TargetDevice:
			cfg.Destination.Kind = core.DestDevice
		}
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
	return nil
}
