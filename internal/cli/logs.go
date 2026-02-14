package cli

import (
	"context"
	"errors"

	"github.com/spf13/cobra"
	"github.com/xcbolt/xcbolt/internal/core"
)

func newLogsCmd() *cobra.Command {
	var predicate string
	var platform string
	var target string
	var targetType string

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Stream logs (simulator via log stream; device logs best-effort)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := NewAppContext(flags)
			if err != nil {
				return err
			}
			if err := applyOverrides(&ac.Config, "", "", platform, target, targetType, ""); err != nil {
				return err
			}

			ctx := context.Background()

			switch ac.Config.Destination.Kind {
			case core.DestSimulator:
				udid := ac.Config.Destination.ID
				if udid == "" {
					udid = ac.Config.Destination.UDID
				}
				if udid == "" {
					return errors.New("missing simulator target id (set in config or pass --target)")
				}
				args := []string{"simctl", "spawn", udid, "log", "stream", "--style", "compact"}
				if predicate != "" {
					args = append(args, "--predicate", predicate)
				}
				_, err := core.RunStreaming(ctx, core.CmdSpec{
					Path: "xcrun",
					Args: args,
					StdoutLine: func(s string) {
						if ac.Flags.JSON {
							ac.Emitter.Emit(core.Log("logs", s))
						} else {
							cmd.Println(s)
						}
					},
					StderrLine: func(s string) {
						if ac.Flags.JSON {
							ac.Emitter.Emit(core.Log("logs", s))
						} else {
							cmd.Println(s)
						}
					},
				})
				return err
			case core.DestDevice:
				// Best effort: advise.
				if ac.Flags.JSON {
					ac.Emitter.Emit(core.Warn("logs", "Device log streaming is best-effort; prefer `xcbolt run --console` or use Console.app."))
					return nil
				}
				cmd.Println("Device logs: best-effort. Prefer `xcbolt run --console` or open Console.app and filter by device/bundle id.")
				return nil
			default:
				return errors.New("logs not supported for this destination kind")
			}
		},
	}

	cmd.Flags().StringVar(&predicate, "predicate", "", "log stream predicate (simulator only)")
	cmd.Flags().StringVar(&platform, "platform", "", "Destination platform family (ios|ipados|tvos|visionos|watchos|macos|catalyst)")
	cmd.Flags().StringVar(&target, "target", "", "Destination ID or exact name")
	cmd.Flags().StringVar(&targetType, "target-type", "", "Destination target type (simulator|device|local)")

	return cmd
}
