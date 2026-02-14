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
	var platform string
	var target string
	var targetType string
	var companionTarget string
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
			if err := applyOverrides(&ac.Config, scheme, configuration, platform, target, targetType, companionTarget); err != nil {
				return err
			}

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

			_, cfg2, err := core.Test(ctx, ac.ProjectRoot, ac.Config, only, skip, ac.Emitter)
			persistConfigIfChanged(ac, cfg2)
			return err
		},
	}

	cmd.Flags().BoolVar(&list, "list", false, "List tests (xcodebuild -enumerate-tests)")
	cmd.Flags().StringArrayVar(&only, "only", []string{}, "Run only these tests (repeatable); value format: <Target>/<Class>/<testMethod>")
	cmd.Flags().StringArrayVar(&skip, "skip", []string{}, "Skip these tests (repeatable)")
	cmd.Flags().StringVar(&scheme, "scheme", "", "Override scheme")
	cmd.Flags().StringVar(&configuration, "configuration", "", "Override configuration")
	cmd.Flags().StringVar(&platform, "platform", "", "Destination platform family (ios|ipados|tvos|visionos|watchos|macos|catalyst)")
	cmd.Flags().StringVar(&target, "target", "", "Destination ID or exact name")
	cmd.Flags().StringVar(&targetType, "target-type", "", "Destination target type (simulator|device|local)")
	cmd.Flags().StringVar(&companionTarget, "companion-target", "", "Companion destination ID/name (watchOS physical runs)")

	return cmd
}
