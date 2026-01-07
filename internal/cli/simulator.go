package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"
	"github.com/xcbolt/xcbolt/internal/core"
)

func newSimulatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "simulator",
		Short: "Manage iOS simulators (simctl)",
	}

	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List simulators",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := NewAppContext(flags)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			list, err := core.SimctlList(ctx, ac.Emitter)
			if err != nil {
				return err
			}
			sims := core.FlattenSimulators(list)
			if ac.Flags.JSON {
				ac.Emitter.Emit(core.Event{Cmd: "simulator", Type: "simulator_list", Data: sims})
				return nil
			}
			b, _ := json.MarshalIndent(sims, "", "  ")
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "boot <udid>",
		Short: "Boot a simulator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			udid := args[0]
			_ = core.SimctlBoot(ctx, udid)
			return core.SimctlBootStatus(ctx, udid)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "shutdown <udid>",
		Short: "Shutdown a simulator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			return core.SimctlShutdown(ctx, args[0])
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "erase <udid>",
		Short: "Erase a simulator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
			defer cancel()
			return core.SimctlErase(ctx, args[0])
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "open",
		Short: "Open Simulator.app",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			return core.SimctlOpenSimulatorApp(ctx)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "openurl <udid> <url>",
		Short: "Open a URL on a simulator",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			return core.SimctlOpenURL(ctx, args[0], args[1])
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "screenshot <udid> [out.png]",
		Short: "Take a screenshot from a simulator",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			udid := args[0]
			out := ""
			if len(args) == 2 {
				out = args[1]
			} else {
				out = filepath.Join(".xcbolt", "screenshots", udid+".png")
			}
			ac, err := NewAppContext(flags)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			out = filepath.Join(ac.ProjectRoot, out)
			return core.SimctlScreenshot(ctx, udid, out)
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "create <name> <deviceTypeId> <runtimeId>",
		Short: "Create a simulator",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			udid, err := core.SimctlCreate(ctx, args[0], args[1], args[2])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), udid)
			return nil
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "delete <udid>",
		Short: "Delete a simulator",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			return core.SimctlDelete(ctx, args[0])
		},
	})

	cmd.AddCommand(&cobra.Command{
		Use:   "prune",
		Short: "Delete unavailable simulators",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			return core.SimctlPrune(ctx)
		},
	})

	return cmd
}
