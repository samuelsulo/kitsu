package skills

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// ShowOptions configures Show.
type ShowOptions struct {
	// Skill is the skill's folder name under skills/.
	Skill string
	// Local and Repo are the same as InstallOptions.
	Local bool
	Repo  string

	Stdout, Stderr io.Writer
}

// Show prints a skill's name and description, read from its SKILL.md
// frontmatter. The frontmatter's name must match Skill (the folder it
// lives in) — Show errors otherwise, since every other skills command
// relies on that invariant to address a skill by its folder name alone.
func Show(opts ShowOptions) error {
	root, cleanup, err := resolveSkillsRoot(opts.Local, opts.Repo)
	if err != nil {
		return err
	}
	defer cleanup()

	skillSrc := filepath.Join(root, "skills", opts.Skill)
	if err := validateSkillDir(skillSrc, opts.Skill); err != nil {
		return err
	}

	skillMDPath := filepath.Join(skillSrc, "SKILL.md")
	content, err := os.ReadFile(skillMDPath)
	if err != nil {
		return err
	}

	fm, err := parseFrontmatter(content)
	if err != nil {
		return fmt.Errorf("%s/SKILL.md: %w", opts.Skill, err)
	}
	if fm.Name != opts.Skill {
		return fmt.Errorf(
			"%s/SKILL.md's frontmatter name is %q, but the skill's folder is named %q — they must match",
			opts.Skill, fm.Name, opts.Skill,
		)
	}

	fmt.Fprintf(opts.Stdout, "name: %s\n", fm.Name)
	fmt.Fprintf(opts.Stdout, "description: %s\n", fm.Description)
	return nil
}
