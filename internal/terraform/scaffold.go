package terraform

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// scaffoldEnvironmentFiles are the empty files scaffolded by
// ScaffoldEnvironment for a new environment.
var scaffoldEnvironmentFiles = []string{"environment.tfvars", "backend.hcl"}

// ScaffoldEnvironment creates <infra-dir>/environments/<env>/ with empty
// environment.tfvars and backend.hcl files, skipping (and reporting) any
// that already exist rather than overwriting them. kitsu doesn't assume
// any particular cloud provider or backend convention — filling these
// files in is left to the project.
func (r Runner) ScaffoldEnvironment() error {
	dir := r.Env.Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	for _, f := range scaffoldEnvironmentFiles {
		path := filepath.Join(dir, f)

		if _, err := os.Stat(path); err == nil {
			fmt.Fprintf(r.Stdout, "  ⤳ %s already exists, skipping\n", path)
			continue
		} else if !os.IsNotExist(err) {
			return err
		}

		if err := os.WriteFile(path, nil, 0o644); err != nil {
			return err
		}
		fmt.Fprintf(r.Stdout, "  ✓ %s created\n", path)
	}

	return nil
}

// scaffoldModuleFiles are the empty files scaffolded by ScaffoldModule
// for a new Terraform module.
var scaffoldModuleFiles = []string{"main.tf", "variables.tf", "outputs.tf", "versions.tf", "README.md"}

// ScaffoldModule creates <infra-dir>/modules/local/<name>/ with the
// standard set of empty Terraform module files, skipping (and reporting)
// any that already exist rather than overwriting them.
func (r Runner) ScaffoldModule(name string) error {
	dir := filepath.Join(r.Env.infraDir(), "modules", "local", name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	for _, f := range scaffoldModuleFiles {
		path := filepath.Join(dir, f)

		if _, err := os.Stat(path); err == nil {
			fmt.Fprintf(r.Stdout, "  ⤳ %s already exists, skipping\n", path)
			continue
		} else if !os.IsNotExist(err) {
			return err
		}

		if err := os.WriteFile(path, nil, 0o644); err != nil {
			return err
		}
		fmt.Fprintf(r.Stdout, "  ✓ %s created\n", path)
	}

	return nil
}

// ScaffoldInfra replaces <infra-dir> with a fresh copy of repo (at ref, a
// branch or tag, or the repository's default branch if ref is empty),
// for bootstrapping a whole project's Terraform tree from an existing
// one. It refuses to touch <infra-dir> if it already exists and isn't
// empty, unless force is true — a plain overwrite would otherwise
// silently discard whatever a previous 'scaffold environment' or
// 'catalog vendor' run had already put there.
func (r Runner) ScaffoldInfra(repo, ref string, force bool) error {
	dir := r.Env.infraDir()

	entries, err := os.ReadDir(dir)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if len(entries) > 0 && !force {
		return fmt.Errorf("%s already exists and is not empty; pass --force to replace it", dir)
	}

	tmpDir, err := os.MkdirTemp("", "kitsu-infra-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	cloneArgs := []string{"clone", "--quiet", "--depth", "1"}
	if ref != "" {
		cloneArgs = append(cloneArgs, "--branch", ref)
	}
	cloneArgs = append(cloneArgs, repo, tmpDir)
	if out, err := exec.Command("git", cloneArgs...).CombinedOutput(); err != nil {
		return fmt.Errorf("cloning %s: %w (%s)", repo, err, strings.TrimSpace(string(out)))
	}

	commitCmd := exec.Command("git", "-C", tmpDir, "rev-parse", "HEAD")
	var commitOut bytes.Buffer
	commitCmd.Stdout = &commitOut
	if err := commitCmd.Run(); err != nil {
		return fmt.Errorf("resolving commit for %s: %w", repo, err)
	}
	commit := strings.TrimSpace(commitOut.String())

	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(tmpDir, ".git")); err != nil {
		return err
	}
	if err := os.CopyFS(dir, os.DirFS(tmpDir)); err != nil {
		return fmt.Errorf("copying %s: %w", tmpDir, err)
	}

	fmt.Fprintf(r.Stdout, "✓ %s replaced from %s (commit %s)\n", dir, repo, commit)
	return nil
}
