package cli

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/xcbolt/xcbolt/internal/core"
)

func newStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop <bundle-id-or-session-id>",
		Short: "Stop a running app previously launched by xcbolt",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := NewAppContext(flags)
			if err != nil {
				return err
			}
			id := args[0]
			s, err := core.LoadSessions(ac.ProjectRoot)
			if err != nil {
				return err
			}
			var sess *core.Session
			for i := range s.Items {
				if s.Items[i].ID == id || s.Items[i].BundleID == id {
					sess = &s.Items[i]
					break
				}
			}
			if sess == nil {
				return errors.New("no tracked session found for " + id)
			}

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			switch sess.Target {
			case "simulator":
				if sess.UDID == "" {
					return errors.New("session missing simulator udid")
				}
				_, err := core.RunStreaming(ctx, core.CmdSpec{
					Path: "xcrun",
					Args: []string{"simctl", "terminate", sess.UDID, sess.BundleID},
				})
				if err != nil {
					return err
				}
			case "device":
				if sess.UDID == "" {
					return errors.New("session missing device udid")
				}
				if err := core.DevicectlStop(ctx, sess.UDID, sess.PID, sess.BundleID, ac.Emitter); err != nil {
					return err
				}
			default:
				return fmt.Errorf("stop not implemented for target %q", sess.Target)
			}

			_ = core.RemoveSession(ac.ProjectRoot, sess.ID)
			fmt.Fprintln(cmd.OutOrStdout(), "Stopped", sess.BundleID)
			return nil
		},
	}
	return cmd
}
