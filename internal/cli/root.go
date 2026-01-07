package cli

import (
	"os"

	"github.com/spf13/cobra"
	"github.com/xcbolt/xcbolt/internal/tui"
)

var (
	flags   GlobalFlags
	rootCmd = &cobra.Command{
		Use:           "xcbolt",
		Short:         "xcbolt — a reliable Xcode CLI + TUI",
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default: TUI (like lazygit). Use `xcbolt --help` for help.
			ac, err := NewAppContext(flags)
			if err != nil {
				return err
			}
			return tui.Run(ac.ProjectRoot, ac.ConfigPath)
		},
	}
)

func Execute() {
	rootCmd.SetOut(os.Stdout)
	rootCmd.SetErr(os.Stderr)

	rootCmd.PersistentFlags().BoolVar(&flags.JSON, "json", false, "Emit NDJSON event stream to stdout")
	rootCmd.PersistentFlags().StringVar(&flags.Config, "config", "", "Path to config file (default: .xcbolt/config.json)")
	rootCmd.PersistentFlags().StringVar(&flags.Project, "project", "", "Project directory (default: auto-detected)")
	rootCmd.PersistentFlags().BoolVar(&flags.Verbose, "verbose", false, "Verbose output")

	rootCmd.AddCommand(newTUICmd())
	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newContextCmd())
	rootCmd.AddCommand(newDoctorCmd())
	rootCmd.AddCommand(newBuildCmd())
	rootCmd.AddCommand(newRunCmd())
	rootCmd.AddCommand(newTestCmd())
	rootCmd.AddCommand(newCleanCmd())
	rootCmd.AddCommand(newAppsCmd())
	rootCmd.AddCommand(newStopCmd())
	rootCmd.AddCommand(newLogsCmd())
	rootCmd.AddCommand(newSimulatorCmd())
	rootCmd.AddCommand(newDeviceCmd())

	if err := rootCmd.Execute(); err != nil {
		PrintFatal(err)
	}
}

func newTUICmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tui",
		Short: "Launch the interactive TUI",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := NewAppContext(flags)
			if err != nil {
				return err
			}
			return tui.Run(ac.ProjectRoot, ac.ConfigPath)
		},
	}
	return cmd
}
