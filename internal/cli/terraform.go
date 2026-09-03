package cli

import (
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/samuelsulo/kitsu/internal/config"
	"github.com/samuelsulo/kitsu/internal/terraform"
	"github.com/spf13/cobra"
)

// newTerraformCmd builds the "terraform" command group: a thin,
// convention-aware wrapper around the Terraform CLI. It assumes the
// directory layout shared by kitsu's client projects — one Terraform
// root at <infra-dir>/live, configured per environment from files under
// <infra-dir>/environments/<env>/ — so day-to-day commands don't need to
// repeat -var-file/-backend-config by hand.
func newTerraformCmd() *cobra.Command {
	var envName, infraDir, bin, docsBin string

	cmd := &cobra.Command{
		Use:   "terraform",
		Short: "Run Terraform against the infrastructure/<env> convention",
		Long: `terraform wraps the Terraform CLI with the directory and flag
conventions shared across projects: one Terraform root at
<infra-dir>/live, configured per environment from
<infra-dir>/environments/<env>/{backend.hcl,environment.tfvars}.`,
	}

	cmd.PersistentFlags().StringVar(&envName, "env", "sandbox", "Environment name (a directory under <infra-dir>/environments/)")
	cmd.PersistentFlags().StringVar(&infraDir, "infra-dir", "infrastructure", "Infrastructure root directory")
	cmd.PersistentFlags().StringVar(&bin, "terraform-bin", "terraform", "Terraform binary to invoke")
	cmd.PersistentFlags().StringVar(&docsBin, "terraform-docs-bin", "terraform-docs", "terraform-docs binary to invoke (used by 'docs')")

	// runnerFor builds the Runner for one command invocation, streaming
	// I/O through cmd itself so tests can redirect it via
	// SetOut/SetErr/SetIn and interactive prompts (destroy, destroy-target)
	// read from the real terminal.
	runnerFor := func(cmd *cobra.Command) terraform.Runner {
		return terraform.Runner{
			Env: terraform.Env{
				Bin:      bin,
				InfraDir: infraDir,
				Name:     envName,
			},
			Stdout: cmd.OutOrStdout(),
			Stderr: cmd.ErrOrStderr(),
			Stdin:  cmd.InOrStdin(),
		}
	}

	cmd.AddCommand(
		newTerraformSimpleCmd(runnerFor, "init", "Initialize Terraform and configure the backend",
			func(r terraform.Runner) error { return r.Init() }),
		newTerraformSimpleCmd(runnerFor, "validate", "Initialize (see init) and validate the configuration",
			func(r terraform.Runner) error { return r.Validate() }),
		newTerraformSimpleCmd(runnerFor, "fmt", "Format Terraform (.tf/.tfvars) and generic HCL files (e.g. backend.hcl)",
			func(r terraform.Runner) error { return r.Fmt() }),
		newTerraformSimpleCmd(runnerFor, "fmt-check", "Check formatting without modifying files (useful in CI)",
			func(r terraform.Runner) error { return r.FmtCheck() }),
		&cobra.Command{
			Use:   "fmt-staged <file>...",
			Short: "Format exactly the given files, not the whole tree (for a pre-commit hook)",
			Long: `fmt-staged formats exactly the given files, applying the same rules
as fmt: .tf/.tfvars files in place, generic .hcl files (e.g.
backend.hcl) through the same stdin trick fmt uses, and .tftest.hcl
files left untouched (unlike fmt, whose recursive pass covers them as
a side effect — a targeted file list has no such pass to piggyback
on).

Intended for a pre-commit hook that formats only staged files, never
the whole tree — pass it the staged files under the infrastructure
directory, e.g.:

  kitsu terraform fmt-staged $(git diff --cached --name-only --diff-filter=ACMR)`,
			Args: cobra.MinimumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return runnerFor(cmd).FmtStaged(args)
			},
		},
		newTerraformSimpleCmd(runnerFor, "plan", "Validate (see validate) and generate and save an execution plan",
			func(r terraform.Runner) error { return r.Plan() }),
		newTerraformSimpleCmd(runnerFor, "show-plan", "Show the plan previously saved by plan",
			func(r terraform.Runner) error { return r.ShowPlan() }),
		newTerraformSimpleCmd(runnerFor, "apply", "Apply the plan previously saved by plan",
			func(r terraform.Runner) error { return r.Apply() }),
		newTerraformTargetCmd(runnerFor, "apply-target", "Validate (see validate) and apply changes to a single resource target",
			func(r terraform.Runner, target string) error { return r.ApplyTarget(target) }),
		newTerraformSimpleCmd(runnerFor, "apply-auto", "Validate, then plan and apply in one step, without a saved plan (use with caution)",
			func(r terraform.Runner) error { return r.ApplyAuto() }),
		newTerraformSimpleCmd(runnerFor, "plan-destroy", "Preview what a destroy would remove, without applying anything",
			func(r terraform.Runner) error { return r.PlanDestroy() }),
		newTerraformSimpleCmd(runnerFor, "refresh", "Reconcile the state with the real infrastructure, without changing either",
			func(r terraform.Runner) error { return r.Refresh() }),
		newTerraformDestroyCmd(runnerFor),
		newTerraformDestroyTargetCmd(runnerFor),
		newTerraformImportCmd(runnerFor),
		newTerraformStateCmd(runnerFor),
		newTerraformUnlockCmd(runnerFor),
		newTerraformAddressCmd(runnerFor, "taint <address>", "Mark a resource for recreation on the next apply",
			func(r terraform.Runner, address string) error { return r.Taint(address) }),
		newTerraformAddressCmd(runnerFor, "untaint <address>", "Undo a previous taint",
			func(r terraform.Runner, address string) error { return r.Untaint(address) }),
		newTerraformSimpleCmd(runnerFor, "console", "Open an interactive console to evaluate expressions against the current state",
			func(r terraform.Runner) error { return r.Console() }),
		newTerraformSimpleCmd(runnerFor, "providers", "Show the provider requirements and versions in use",
			func(r terraform.Runner) error { return r.Providers() }),
		newTerraformSimpleCmd(runnerFor, "version", "Show the Terraform and provider versions",
			func(r terraform.Runner) error { return r.TerraformVersion() }),
		newTerraformSimpleCmd(runnerFor, "upgrade", "Re-initialize and upgrade provider/module versions to the latest allowed",
			func(r terraform.Runner) error { return r.Upgrade() }),
		newTerraformSimpleCmd(runnerFor, "output", "Show the outputs of the environment's current state",
			func(r terraform.Runner) error { return r.Output() }),
		newTerraformSimpleCmd(runnerFor, "output-json", "Show the outputs of the environment's current state, in JSON format",
			func(r terraform.Runner) error { return r.OutputJSON() }),
		newTerraformSimpleCmd(runnerFor, "clean", "Remove the local Terraform cache and this environment's saved plan",
			func(r terraform.Runner) error { return r.Clean() }),
		&cobra.Command{
			Use:   "docs",
			Short: "Regenerate each module's README input/output tables with terraform-docs",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				return runnerFor(cmd).Docs(docsBin)
			},
		},
		newTerraformScaffoldCmd(runnerFor),
		newTerraformCatalogCmd(runnerFor),
		newTerraformBootstrapBackendCmd(),
	)

	return cmd
}

// runnerFactory builds a Runner scoped to one command invocation's flags
// and I/O streams.
type runnerFactory func(cmd *cobra.Command) terraform.Runner

// newTerraformSimpleCmd builds a terraform subcommand that takes no
// arguments of its own beyond the "terraform" group's persistent flags.
func newTerraformSimpleCmd(runnerFor runnerFactory, use, short string, run func(terraform.Runner) error) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(runnerFor(cmd))
		},
	}
}

// newTerraformTargetCmd builds a terraform subcommand that operates on a
// single resource address, given via the required --target flag.
func newTerraformTargetCmd(runnerFor runnerFactory, use, short string, run func(terraform.Runner, string) error) *cobra.Command {
	var target string

	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(runnerFor(cmd), target)
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Resource address to target (required)")
	cmd.MarkFlagRequired("target")

	return cmd
}

// newTerraformAddressCmd builds a terraform subcommand that operates on a
// single resource address, given as a positional argument.
func newTerraformAddressCmd(runnerFor runnerFactory, use, short string, run func(terraform.Runner, string) error) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return run(runnerFor(cmd), args[0])
		},
	}
}

func newTerraformImportCmd(runnerFor runnerFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "import <address> <id>",
		Short: "Import an existing resource, identified by its provider-specific id, into the Terraform state",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runnerFor(cmd).Import(args[0], args[1])
		},
	}
}

func newTerraformUnlockCmd(runnerFor runnerFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "unlock <lock-id>",
		Short: "Force-release a stuck state lock",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			r := runnerFor(cmd)
			ok, err := confirm(cmd, fmt.Sprintf(
				"Force-unlocking state for environment %q. Only do this if you are sure no other apply is running.",
				r.Env.Name,
			))
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("aborted")
			}
			return r.Unlock(args[0])
		},
	}
}

// newTerraformStateCmd builds the "state" subcommand group, mirroring
// Terraform's own `terraform state <subcommand>` naming.
func newTerraformStateCmd(runnerFor runnerFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "state",
		Short: "Inspect or modify the Terraform state",
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List every resource in the Terraform state",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				return runnerFor(cmd).StateList()
			},
		},
		&cobra.Command{
			Use:   "show <address>",
			Short: "Show the state attributes of a single resource",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				return runnerFor(cmd).StateShow(args[0])
			},
		},
		&cobra.Command{
			Use:   "rm <address>",
			Short: "Remove a resource from the state without destroying it",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				r := runnerFor(cmd)
				ok, err := confirm(cmd, fmt.Sprintf(
					"Removing %q from the Terraform state for environment %q.", args[0], r.Env.Name,
				))
				if err != nil {
					return err
				}
				if !ok {
					return fmt.Errorf("aborted")
				}
				return r.StateRemove(args[0])
			},
		},
	)

	return cmd
}

func newTerraformDestroyCmd(runnerFor runnerFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "destroy",
		Short: "Destroy every resource in the target environment",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r := runnerFor(cmd)
			ok, err := confirm(cmd, fmt.Sprintf("Destroying all resources for environment %q.", r.Env.Name))
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("aborted")
			}
			return r.Destroy()
		},
	}
}

func newTerraformDestroyTargetCmd(runnerFor runnerFactory) *cobra.Command {
	var target string

	cmd := &cobra.Command{
		Use:   "destroy-target",
		Short: "Destroy a single resource target",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r := runnerFor(cmd)
			ok, err := confirm(cmd, fmt.Sprintf("Destroying target %q for environment %q.", target, r.Env.Name))
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("aborted")
			}
			return r.DestroyTarget(target)
		},
	}

	cmd.Flags().StringVar(&target, "target", "", "Resource address to target (required)")
	cmd.MarkFlagRequired("target")

	return cmd
}

// confirm prints prompt to cmd's output and asks the user to type "yes" on
// cmd's input, returning whether they did. Used before every destructive
// operation that isn't otherwise guarded by a saved plan.
//
// destroy/destroy-target run the real `terraform destroy`, which asks for
// its own "yes" on the same input afterwards — so this must read exactly
// one line and no further, leaving whatever comes after it untouched for
// that second prompt to read.
func confirm(cmd *cobra.Command, prompt string) (bool, error) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s Type 'yes' to confirm: ", prompt)

	line, err := readLine(cmd.InOrStdin())
	if err != nil && line == "" {
		return false, nil
	}

	return strings.TrimSpace(line) == "yes", nil
}

// readLine reads one newline-terminated line from r a byte at a time, so
// it never buffers ahead past the line itself: unlike bufio.Reader, it
// won't silently consume input meant for whatever reads r next.
func readLine(r io.Reader) (string, error) {
	var line bytes.Buffer
	b := make([]byte, 1)

	for {
		n, err := r.Read(b)
		if n > 0 {
			if b[0] == '\n' {
				return line.String(), nil
			}
			line.WriteByte(b[0])
		}
		if err != nil {
			return line.String(), err
		}
	}
}

// accountIDPattern matches a bare 12-digit AWS account id.
var accountIDPattern = regexp.MustCompile(`^[0-9]{12}$`)

// newTerraformScaffoldCmd builds the "scaffold" subcommand group.
func newTerraformScaffoldCmd(runnerFor runnerFactory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "scaffold",
		Short: "Scaffold a new environment or module",
	}

	cmd.AddCommand(newTerraformScaffoldEnvironmentCmd(runnerFor), newTerraformScaffoldModuleCmd(runnerFor))

	return cmd
}

func newTerraformScaffoldEnvironmentCmd(runnerFor runnerFactory) *cobra.Command {
	var accountID, roleARNTemplate string

	cmd := &cobra.Command{
		Use:   "environment",
		Short: "Scaffold a new environment from an AWS account id and live/project.auto.tfvars",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r := runnerFor(cmd)

			if r.Env.Name != "sandbox" && r.Env.Name != "production" {
				return fmt.Errorf("--env must be either \"sandbox\" or \"production\", got %q", r.Env.Name)
			}
			if !accountIDPattern.MatchString(accountID) {
				return fmt.Errorf("--account-id must be a 12-digit AWS account id, got %q", accountID)
			}

			template, err := config.ResolveRoleARNTemplate(roleARNTemplate)
			if err != nil {
				return err
			}

			return r.ScaffoldEnvironment(accountID, template)
		},
	}

	cmd.Flags().StringVar(&accountID, "account-id", "", "12-digit AWS account id (required)")
	cmd.MarkFlagRequired("account-id")
	cmd.Flags().StringVar(&roleARNTemplate, "role-arn-template", "",
		"IAM role ARN template with %s for the account id (defaults to terraform.role_arn_template in the kitsu config file)")

	return cmd
}

func newTerraformScaffoldModuleCmd(runnerFor runnerFactory) *cobra.Command {
	return &cobra.Command{
		Use:   "module <name>",
		Short: "Scaffold a new project-specific Terraform module under modules/local/",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runnerFor(cmd).ScaffoldModule(args[0])
		},
	}
}

// newTerraformCatalogCmd builds the "catalog" subcommand group: every
// operation against the shared Terraform module catalog repository.
func newTerraformCatalogCmd(runnerFor runnerFactory) *cobra.Command {
	var catalogRepo string

	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "List and vendor modules from the Terraform module catalog",
	}

	cmd.PersistentFlags().StringVar(&catalogRepo, "catalog-repo", "",
		"Git URL of the module catalog (defaults to terraform.catalog_repo in the kitsu config file)")

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "List modules available in the catalog",
			Args:  cobra.NoArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				repo, err := config.ResolveCatalogRepo(catalogRepo)
				if err != nil {
					return err
				}
				return runnerFor(cmd).CatalogList(repo)
			},
		},
		&cobra.Command{
			Use:   "versions <module>",
			Short: "List available versions of a catalog module, newest first",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				repo, err := config.ResolveCatalogRepo(catalogRepo)
				if err != nil {
					return err
				}
				return runnerFor(cmd).CatalogVersions(repo, args[0])
			},
		},
		&cobra.Command{
			Use:   "vendor <module> <version>",
			Short: "Copy a module from the catalog into modules/vendor/, pinned to a tag",
			Args:  cobra.ExactArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				repo, err := config.ResolveCatalogRepo(catalogRepo)
				if err != nil {
					return err
				}
				return runnerFor(cmd).CatalogVendor(repo, args[0], args[1])
			},
		},
	)

	return cmd
}

// newTerraformBootstrapBackendCmd builds the "bootstrap-backend" command.
// Unlike the rest of the group, it doesn't need a Runner: it operates on
// one AWS account as a whole (via the AWS CLI), not on any particular
// Terraform environment.
func newTerraformBootstrapBackendCmd() *cobra.Command {
	var region string
	var lifecycleDays int

	cmd := &cobra.Command{
		Use:   "bootstrap-backend",
		Short: "Create (once) the S3 bucket for the Terraform state of the current AWS account",
		Long: `bootstrap-backend creates the S3 bucket that every Terraform
project's backend.hcl in this AWS account points at (one bucket per
account, one key per project). It always operates on the account of the
currently active AWS credentials (aws sts get-caller-identity) — never
an account id passed by hand — so run it once per account, with that
account's credentials active, before the first 'kitsu terraform init'
there.

Idempotent: if the bucket already exists, it only verifies/realigns the
configuration (versioning, encryption, public access block, ownership,
TLS-only policy, lifecycle, tags).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return terraform.BootstrapBackend(terraform.BootstrapBackendOptions{
				Region:                          region,
				NoncurrentVersionExpirationDays: lifecycleDays,
				Stdout:                          cmd.OutOrStdout(),
				Stderr:                          cmd.ErrOrStderr(),
			})
		},
	}

	cmd.Flags().StringVar(&region, "region", "eu-south-1", "AWS region the state bucket lives in")
	cmd.Flags().IntVar(&lifecycleDays, "noncurrent-version-expiration-days", 90, "Days to keep noncurrent state versions before they're deleted")

	return cmd
}
