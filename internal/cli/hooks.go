package cli

import (
	"fmt"

	"github.com/samuelsulo/kitsu/internal/git"
	"github.com/samuelsulo/kitsu/internal/githooks"
	"github.com/spf13/cobra"
)

func newHooksCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "hooks",
		Short: "Manage this project's git hooks",
	}

	cmd.AddCommand(newHooksInstallCmd())

	return cmd
}

func newHooksInstallCmd() *cobra.Command {
	var dir string

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Point git at the hooks tracked in the repository",
		Long: `Install sets core.hooksPath to the given directory (default:
.githooks) and makes every hook file directly inside it executable.

Git never tracks .git/hooks/ itself, so hooks meant to be shared with a
repository must live elsewhere in the tree (by convention, .githooks/)
and be wired up explicitly after cloning — this is a one-time step to run
right after cloning a project that ships hooks this way.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := git.Root("")
			if err != nil {
				return err
			}

			if err := githooks.Install(root, dir); err != nil {
				return err
			}

			fmt.Fprintf(cmd.OutOrStdout(), "✓ git hooks installed (core.hooksPath -> %s)\n", dir)
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", ".githooks", "Directory (relative to the repo root) containing the tracked hooks")

	return cmd
}
