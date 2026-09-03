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

	cmd.AddCommand(newWebsiteDeployCmd())

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
