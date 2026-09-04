// Package skills installs and packages Claude Code skills from a
// skills/<name>/ folder — either in the current git repository
// (--local, for testing a skill before it's pushed) or freshly cloned
// from a configured GitHub repository.
package skills

import (
	"archive/zip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/samuelsulo/kitsu/internal/git"
)

// InstallOptions configures Install.
type InstallOptions struct {
	// Skill is the skill's folder name under skills/.
	Skill string
	// Local reads skills/<Skill>/ from the current git repository
	// instead of cloning Repo.
	Local bool
	// Repo is the git URL (or local path) to shallow-clone from when
	// Local is false.
	Repo string

	Stdout, Stderr io.Writer
}

// Install copies skills/<Skill>/ into ~/.claude/skills/<Skill>/
// (overwritten if it already exists), so Claude Code picks it up.
func Install(opts InstallOptions) error {
	root, cleanup, err := resolveSkillsRoot(opts.Local, opts.Repo)
	if err != nil {
		return err
	}
	defer cleanup()

	skillSrc := filepath.Join(root, "skills", opts.Skill)
	if err := validateSkillDir(skillSrc, opts.Skill); err != nil {
		return err
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("locating the home directory: %w", err)
	}
	destDir := filepath.Join(home, ".claude", "skills")
	destPath := filepath.Join(destDir, opts.Skill)

	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	if err := os.RemoveAll(destPath); err != nil {
		return err
	}
	if err := os.CopyFS(destPath, os.DirFS(skillSrc)); err != nil {
		return fmt.Errorf("copying %s: %w", skillSrc, err)
	}

	fmt.Fprintf(opts.Stdout, "✓ %s installed -> %s\n", opts.Skill, destPath)
	return nil
}

// PackageOptions configures Package.
type PackageOptions struct {
	// Skill is the skill's folder name under skills/.
	Skill string
	// OutputDir is where <Skill>.zip is written. Empty means the
	// current directory.
	OutputDir string
	// Local and Repo are the same as InstallOptions.
	Local bool
	Repo  string

	Stdout, Stderr io.Writer
}

// Package zips skills/<Skill>/ into <OutputDir>/<Skill>.zip (overwritten
// if it already exists), ready to be uploaded to a new machine/Claude
// instance.
func Package(opts PackageOptions) (zipPath string, err error) {
	root, cleanup, err := resolveSkillsRoot(opts.Local, opts.Repo)
	if err != nil {
		return "", err
	}
	defer cleanup()

	skillsDir := filepath.Join(root, "skills")
	skillSrc := filepath.Join(skillsDir, opts.Skill)
	if err := validateSkillDir(skillSrc, opts.Skill); err != nil {
		return "", err
	}

	outDir := opts.OutputDir
	if outDir == "" {
		outDir = "."
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return "", err
	}
	absOutDir, err := filepath.Abs(outDir)
	if err != nil {
		return "", err
	}
	zipPath = filepath.Join(absOutDir, opts.Skill+".zip")
	_ = os.Remove(zipPath)

	fmt.Fprintf(opts.Stderr, "Packaging %q -> %s\n", opts.Skill, zipPath)
	if err := zipSkill(skillsDir, opts.Skill, zipPath); err != nil {
		return "", err
	}

	fmt.Fprintf(opts.Stdout, "✓ %s\n", zipPath)
	return zipPath, nil
}

// resolveSkillsRoot returns the directory to read skills/<name>/ from:
// the current git repository's root when local is true, or a fresh
// shallow clone of repo into a temp directory otherwise — along with a
// cleanup func the caller must defer, removing that temp directory (a
// no-op in the local case).
func resolveSkillsRoot(local bool, repo string) (root string, cleanup func(), err error) {
	noop := func() {}

	if local {
		root, err = git.Root("")
		if err != nil {
			return "", noop, err
		}
		return root, noop, nil
	}

	tmpDir, err := os.MkdirTemp("", "kitsu-skills-")
	if err != nil {
		return "", noop, err
	}
	cleanup = func() { _ = os.RemoveAll(tmpDir) }

	cmd := exec.Command("git", "clone", "--quiet", "--depth", "1", repo, tmpDir)
	if out, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return "", noop, fmt.Errorf("cloning %s: %w (%s)", repo, err, strings.TrimSpace(string(out)))
	}

	return tmpDir, cleanup, nil
}

// validateSkillDir errors out unless dir is a folder containing a
// SKILL.md, naming skill in the message.
func validateSkillDir(dir, skill string) error {
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("%q is not a folder in skills/", skill)
	}
	if _, err := os.Stat(filepath.Join(dir, "SKILL.md")); err != nil {
		return fmt.Errorf("%s/SKILL.md not found — is %q really a skill folder?", skill, skill)
	}
	return nil
}

// zipSkill zips skillsDir/skill into zipPath, with archive paths
// relative to skillsDir (so the archive's root is "<skill>/...", same
// as running `zip -r` from inside skillsDir).
func zipSkill(skillsDir, skill, zipPath string) error {
	zf, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer zf.Close()

	w := zip.NewWriter(zf)
	defer w.Close()

	srcDir := filepath.Join(skillsDir, skill)
	return filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(skillsDir, path)
		if err != nil {
			return err
		}

		entry, err := w.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = entry.Write(content)
		return err
	})
}
