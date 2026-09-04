// Package cli wires up kitsu's command tree (Cobra root + subcommands).
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// NewRootCmd builds the root "kitsu" command with all subcommands attached.
func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "kitsu",
		Short: "kitsu is a personal automation CLI",
		Long: `kitsu bundles automations used across projects
(scaffolding, checks, repetitive tasks, ...) behind a single binary.`,
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.AddCommand(newVersionCmd())
	root.AddCommand(newHooksCmd())
	root.AddCommand(newTerraformCmd())
	root.AddCommand(newWebsiteCmd())
	root.AddCommand(newSkillsCmd())
	root.AddCommand(newConfigCmd())

	return root
}

// Execute runs the root command and prints any error to stderr, returning a
// process exit code.
func Execute() int {
	if err := NewRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	return 0
}
