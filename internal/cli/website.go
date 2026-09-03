package cli

import (
	"github.com/samuelsulo/kitsu/internal/website"
	"github.com/spf13/cobra"
)

// newWebsiteCmd builds the "website" command group.
func newWebsiteCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "website",
		Short: "Build and deploy the project's website",
	}

	cmd.AddCommand(newWebsiteDeployCmd(), newWebsiteCurrentCmd(), newWebsiteHistoryCmd())

	return cmd
}

func newWebsiteDeployCmd() *cobra.Command {
	var env, tag, infraDir, websiteDir, terraformBin string
	var force bool

	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Build website/ and deploy it to the target environment",
		Long: `deploy builds the website and syncs it to the S3 bucket and
CloudFront distribution of the given environment, read from that
environment's Terraform state (never guessed or hardcoded).

In production, it deploys the commit pointed at by an explicit
"website/vX.Y.Z" tag (--tag, required) — not necessarily the currently
checked-out commit — and refuses to redeploy an already-deployed tag
or downgrade to an older one unless --force is passed. Every other
environment deploys whatever commit is currently checked out,
versioned by its short SHA, and accepts neither --tag nor --force.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return website.Deploy(website.DeployOptions{
				Env:          env,
				Tag:          tag,
				Force:        force,
				InfraDir:     infraDir,
				WebsiteDir:   websiteDir,
				TerraformBin: terraformBin,
				Stdout:       cmd.OutOrStdout(),
				Stderr:       cmd.ErrOrStderr(),
			})
		},
	}

	cmd.Flags().StringVar(&env, "env", "", `Environment to deploy to: "sandbox" or "production" (required)`)
	cmd.MarkFlagRequired("env")
	cmd.Flags().StringVar(&tag, "tag", "", "Release tag to deploy, e.g. website/v1.0.1 (required for production, forbidden otherwise)")
	cmd.Flags().BoolVar(&force, "force", false, "Skip the already-deployed/downgrade guards (production only)")
	cmd.Flags().StringVar(&infraDir, "infra-dir", "infrastructure", "Infrastructure root directory, relative to the repo root")
	cmd.Flags().StringVar(&websiteDir, "website-dir", "website", "Website project directory, relative to the repo root")
	cmd.Flags().StringVar(&terraformBin, "terraform-bin", "terraform", "Terraform binary to invoke")

	return cmd
}

// newInspectCmd builds a "website" subcommand that only needs
// InspectOptions (--env, --infra-dir, --terraform-bin), for commands
// that read deploy version tracking rather than deploying anything.
func newInspectCmd(use, short, long string, run func(website.InspectOptions) error) *cobra.Command {
	var env, infraDir, terraformBin string

	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Long:  long,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(website.InspectOptions{
				Env:          env,
				InfraDir:     infraDir,
				TerraformBin: terraformBin,
				Stdout:       cmd.OutOrStdout(),
				Stderr:       cmd.ErrOrStderr(),
			})
		},
	}

	cmd.Flags().StringVar(&env, "env", "", `Environment to inspect, e.g. "production" (required)`)
	cmd.MarkFlagRequired("env")
	cmd.Flags().StringVar(&infraDir, "infra-dir", "infrastructure", "Infrastructure root directory, relative to the repo root")
	cmd.Flags().StringVar(&terraformBin, "terraform-bin", "terraform", "Terraform binary to invoke")

	return cmd
}

func newWebsiteCurrentCmd() *cobra.Command {
	return newInspectCmd("current", "Print the version currently live in an environment",
		`current prints the tag currently live in --env — and nothing else,
so it's easy to use in a script, e.g.:

  if [ "$(kitsu website current --env production)" = "$TAG" ]; then ...

Only environments deploy records version markers for (currently just
"production" — see 'kitsu website deploy') have a current version.`,
		website.Current)
}

func newWebsiteHistoryCmd() *cobra.Command {
	return newInspectCmd("history", "List every version ever deployed to an environment",
		`history lists every version ever deployed to --env, most recently
deployed first, marking whichever one is currently live. Ordered by
deploy time rather than version number, so a rollback (an older
version deployed more recently) is visible as such rather than
looking like a gap.

Only environments deploy records version markers for (currently just
"production" — see 'kitsu website deploy') have any history.`,
		website.History)
}
