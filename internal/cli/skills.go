package cli

import (
	"fmt"

	"github.com/samuelsulo/kitsu/internal/config"
	"github.com/samuelsulo/kitsu/internal/skills"
	"github.com/spf13/cobra"
)

// newSkillsCmd builds the "skills" command group.
func newSkillsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "skills",
		Short: "Install and package Claude Code skills",
	}

	cmd.AddCommand(newSkillsInstallCmd(), newSkillsPackageCmd(), newSkillsListCmd(), newSkillsShowCmd())

	return cmd
}

// skillsRepoFlags adds the --repo/--local flags shared by install and
// package, and returns a resolver that turns them into the git clone
// source Install/Package expect: "" (unused) when local is true,
// otherwise a full GitHub clone URL built from --repo (or its
// fallbacks — see config.ResolveSkillsRepo).
func skillsRepoFlags(cmd *cobra.Command) (resolve func() (repo string, local bool, err error)) {
	var repo string
	var local bool

	cmd.Flags().StringVar(&repo, "repo", "",
		`GitHub "owner/repo" to install/package from (defaults to skills.repo in the kitsu config file, or "samuelsulo/claude-skills")`)
	cmd.Flags().BoolVar(&local, "local", false,
		"Read skills/<skill>/ from the current git repository instead of cloning --repo (e.g. to test a change before it's pushed)")

	return func() (string, bool, error) {
		if local {
			if repo != "" {
				return "", false, fmt.Errorf("--repo is not used with --local")
			}
			return "", true, nil
		}

		ownerRepo, err := config.ResolveSkillsRepo(repo)
		if err != nil {
			return "", false, err
		}
		return "https://github.com/" + ownerRepo + ".git", false, nil
	}
}

func newSkillsInstallCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install <skill>",
		Short: "Install a skill into ~/.claude/skills/",
		Long: `install copies skills/<skill>/ into ~/.claude/skills/<skill>/
(overwritten if it already exists), so Claude Code picks it up.

By default, it clones fresh into a temp directory from --repo (a
GitHub "owner/repo", defaulting to skills.repo in the kitsu config
file, or "samuelsulo/claude-skills" if neither is set). Pass --local
to instead read skills/<skill>/ from the current git repository, e.g.
while working inside a checkout of the skills repo itself, to test a
change before it's pushed.`,
		Args: cobra.ExactArgs(1),
	}

	resolve := skillsRepoFlags(cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		repo, local, err := resolve()
		if err != nil {
			return err
		}
		return skills.Install(skills.InstallOptions{
			Skill:  args[0],
			Local:  local,
			Repo:   repo,
			Stdout: cmd.OutOrStdout(),
			Stderr: cmd.ErrOrStderr(),
		})
	}

	return cmd
}

func newSkillsListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List every skill's name",
		Long: `list prints the name of every skill folder under skills/ — one
per line, sorted — without reading each one's SKILL.md (see 'show'
for that).

By default, it clones fresh into a temp directory from --repo (a
GitHub "owner/repo", defaulting to skills.repo in the kitsu config
file, or "samuelsulo/claude-skills" if neither is set). Pass --local
to instead read skills/ from the current git repository.`,
		Args: cobra.NoArgs,
	}

	resolve := skillsRepoFlags(cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		repo, local, err := resolve()
		if err != nil {
			return err
		}
		return skills.List(skills.ListOptions{
			Local:  local,
			Repo:   repo,
			Stdout: cmd.OutOrStdout(),
			Stderr: cmd.ErrOrStderr(),
		})
	}

	return cmd
}

func newSkillsShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <skill>",
		Short: "Show a skill's name and description",
		Long: `show prints a skill's name and description, read from its
SKILL.md frontmatter. The frontmatter's name must match <skill> (the
folder it lives in) — show errors otherwise, since every other skills
command relies on that invariant to address a skill by its folder
name alone.

By default, it clones fresh into a temp directory from --repo (a
GitHub "owner/repo", defaulting to skills.repo in the kitsu config
file, or "samuelsulo/claude-skills" if neither is set). Pass --local
to instead read skills/<skill>/ from the current git repository.`,
		Args: cobra.ExactArgs(1),
	}

	resolve := skillsRepoFlags(cmd)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		repo, local, err := resolve()
		if err != nil {
			return err
		}
		return skills.Show(skills.ShowOptions{
			Skill:  args[0],
			Local:  local,
			Repo:   repo,
			Stdout: cmd.OutOrStdout(),
			Stderr: cmd.ErrOrStderr(),
		})
	}

	return cmd
}

func newSkillsPackageCmd() *cobra.Command {
	var outputDir string

	cmd := &cobra.Command{
		Use:   "package <skill>",
		Short: "Zip a skill, ready to upload to a new machine/Claude instance",
		Long: `package zips skills/<skill>/ into <output-dir>/<skill>.zip
(overwritten if it already exists).

By default, it clones fresh into a temp directory from --repo (a
GitHub "owner/repo", defaulting to skills.repo in the kitsu config
file, or "samuelsulo/claude-skills" if neither is set). Pass --local
to instead read skills/<skill>/ from the current git repository.`,
		Args: cobra.ExactArgs(1),
	}

	resolve := skillsRepoFlags(cmd)
	cmd.Flags().StringVar(&outputDir, "output-dir", ".", "Directory to write <skill>.zip into")

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		repo, local, err := resolve()
		if err != nil {
			return err
		}
		_, err = skills.Package(skills.PackageOptions{
			Skill:     args[0],
			OutputDir: outputDir,
			Local:     local,
			Repo:      repo,
			Stdout:    cmd.OutOrStdout(),
			Stderr:    cmd.ErrOrStderr(),
		})
		return err
	}

	return cmd
}
