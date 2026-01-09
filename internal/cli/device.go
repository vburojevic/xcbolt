package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/xcbolt/xcbolt/internal/core"
)

func newDeviceCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "device",
		Short: "Manage physical devices (devicectl)",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List connected devices",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := NewAppContext(flags)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			devs, err := core.DevicectlList(ctx, ac.Emitter)
			if err != nil {
				return err
			}
			if ac.Flags.JSON {
				ac.Emitter.Emit(core.Event{Cmd: "device", Type: "device_list", Data: devs})
				return nil
			}
			b, _ := json.MarshalIndent(devs, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "install <device-udid> <app-path>",
		Short: "Install an .app bundle on a device",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := NewAppContext(flags)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()
			return core.DevicectlInstallApp(ctx, args[0], args[1], ac.Emitter)
		},
	})

	{
		var console bool
		launchCmd := &cobra.Command{
			Use:   "launch <device-udid> <bundle-id>",
			Short: "Launch an app on a device by bundle id",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				ac, err := NewAppContext(flags)
				if err != nil {
					return err
				}
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				defer cancel()
				info := core.AppBundleInfo{BundleID: args[1]}
				_, err = core.DevicectlLaunchApp(ctx, args[0], args[1], console, nil, info, false, ac.Emitter)
				return err
			},
		}
		launchCmd.Flags().BoolVar(&console, "console", false, "Attempt to stream console output (Xcode 16+ best-effort)")
		cmd.AddCommand(launchCmd)
	}

	return cmd
}
