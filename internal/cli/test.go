package cli

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/xcbolt/xcbolt/internal/core"
)

func newTestCmd() *cobra.Command {
	var scheme string
	var configuration string
	var simulator string
	var device string
	var list bool
	var only []string
	var skip []string

	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run tests for the configured scheme",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := NewAppContext(flags)
			if err != nil {
				return err
			}
			applyOverrides(&ac.Config, scheme, configuration, simulator, device)

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if list {
				et, err := core.XcodebuildEnumerateTests(ctx, ac.ProjectRoot, ac.Config)
				if err != nil {
					return err
				}
				if ac.Flags.JSON {
					ac.Emitter.Emit(core.Event{Cmd: "test", Type: "test_list", Data: et.Raw})
					return nil
				}
				b, _ := json.MarshalIndent(et.Raw, "", "  ")
				fmt.Fprintln(cmd.OutOrStdout(), string(b))
				return nil
			}

			_, _, err = core.Test(ctx, ac.ProjectRoot, ac.Config, only, skip, ac.Emitter)
			return err
		},
	}

	cmd.Flags().BoolVar(&list, "list", false, "List tests (xcodebuild -enumerate-tests)")
	cmd.Flags().StringArrayVar(&only, "only", []string{}, "Run only these tests (repeatable); value format: <Target>/<Class>/<testMethod>")
	cmd.Flags().StringArrayVar(&skip, "skip", []string{}, "Skip these tests (repeatable)")
	cmd.Flags().StringVar(&scheme, "scheme", "", "Override scheme")
	cmd.Flags().StringVar(&configuration, "configuration", "", "Override configuration")
	cmd.Flags().StringVar(&simulator, "simulator", "", "Simulator UDID")
	cmd.Flags().StringVar(&device, "device", "", "Device UDID")

	return cmd
}
