package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/xcbolt/xcbolt/internal/util"
)

func newCleanCmd() *cobra.Command {
	var all bool
	var derived bool
	var results bool
	var sessions bool
	var spmCache bool

	cmd := &cobra.Command{
		Use:   "clean",
		Short: "Clean xcbolt artifacts (DerivedData, Results, sessions)",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac, err := NewAppContext(flags)
			if err != nil {
				return err
			}

			dd := derived || all || (!derived && !results && !sessions && !spmCache && !all)
			rb := results || all || (!derived && !results && !sessions && !spmCache && !all)
			sess := sessions || all || (!derived && !results && !sessions && !spmCache && !all)
			spm := spmCache || all

			if dd {
				path := filepath.Join(ac.ProjectRoot, ".xcbolt", "DerivedData")
				_ = util.RemoveAllIfExists(path)
				fmt.Fprintln(cmd.OutOrStdout(), "Removed", path)
			}
			if rb {
				path := filepath.Join(ac.ProjectRoot, ".xcbolt", "Results")
				_ = util.RemoveAllIfExists(path)
				fmt.Fprintln(cmd.OutOrStdout(), "Removed", path)
			}
			if sess {
				path := filepath.Join(ac.ProjectRoot, ".xcbolt", "sessions.json")
				_ = util.RemoveAllIfExists(path)
				fmt.Fprintln(cmd.OutOrStdout(), "Removed", path)
			}
			if spm {
				if home, err := os.UserHomeDir(); err == nil {
					path := filepath.Join(home, "Library", "Caches", "org.swift.swiftpm")
					_ = util.RemoveAllIfExists(path)
					fmt.Fprintln(cmd.OutOrStdout(), "Removed", path)
					path = filepath.Join(home, "Library", "Developer", "Xcode", "SourcePackages")
					_ = util.RemoveAllIfExists(path)
					fmt.Fprintln(cmd.OutOrStdout(), "Removed", path)
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Remove everything under .xcbolt")
	cmd.Flags().BoolVar(&derived, "derived-data", false, "Remove .xcbolt/DerivedData")
	cmd.Flags().BoolVar(&results, "results", false, "Remove .xcbolt/Results")
	cmd.Flags().BoolVar(&sessions, "sessions", false, "Remove .xcbolt/sessions.json")
	cmd.Flags().BoolVar(&spmCache, "spm-cache", false, "Remove SwiftPM caches (~/Library/Caches/org.swift.swiftpm, ~/Library/Developer/Xcode/SourcePackages)")

	return cmd
}
