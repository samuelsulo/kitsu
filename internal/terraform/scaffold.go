package terraform

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// ScaffoldEnvironment creates <infra-dir>/environments/<env>/{environment.tfvars,backend.hcl}
// for a new AWS account, reading 'project' and 'aws_region' from
// live/project.auto.tfvars. roleARNTemplate is a fmt template with a
// single %s for the AWS account id (e.g.
// "arn:aws:iam::%s:role/MyAdminRole"), written into backend.hcl's
// assume_role.role_arn; environment.tfvars only gets the
// aws_assume_role_enabled flag, not the ARN itself.
func (r Runner) ScaffoldEnvironment(accountID, roleARNTemplate string) error {
	projectTFVars := filepath.Join(r.Env.LiveDir(), "project.auto.tfvars")

	project, err := readTFVarsString(projectTFVars, "project")
	if err != nil {
		return err
	}
	region, err := readTFVarsString(projectTFVars, "aws_region")
	if err != nil {
		return err
	}
	if project == "" || region == "" {
		return fmt.Errorf("could not read 'project' and 'aws_region' from %s", projectTFVars)
	}

	if err := os.MkdirAll(r.Env.Dir(), 0o755); err != nil {
		return err
	}

	roleARN := fmt.Sprintf(roleARNTemplate, accountID)
	// aws_role_arn isn't passed to Terraform as a variable: the account
	// to assume into is configured once, on the backend itself (see
	// backendHCL below), and the Terraform code only needs to know
	// whether to assume a role at all.
	environmentTFVars := fmt.Sprintf(
		"environment             = %q\naws_account_id          = %q\naws_assume_role_enabled = true\n",
		r.Env.Name, accountID,
	)
	environmentTFVarsPath := filepath.Join(r.Env.Dir(), "environment.tfvars")
	if err := os.WriteFile(environmentTFVarsPath, []byte(environmentTFVars), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(r.Stdout, "✓ %s written (environment=%s, aws_account_id=%s)\n", environmentTFVarsPath, r.Env.Name, accountID)

	backendHCL := fmt.Sprintf(
		"bucket       = %q\nkey          = %q\nregion       = %q\nuse_lockfile = true\nencrypt      = true\n\nassume_role = {\n  role_arn = %q\n}\n",
		StateBucketName(accountID, region),
		project+"/"+r.Env.Name+"/terraform.tfstate",
		region,
		roleARN,
	)
	backendHCLPath := filepath.Join(r.Env.Dir(), "backend.hcl")
	if err := os.WriteFile(backendHCLPath, []byte(backendHCL), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(r.Stdout, "✓ %s written (project=%s, aws_region=%s)\n", backendHCLPath, project, region)

	return nil
}

// tfVarsLinePattern matches a simple string assignment line in a
// .tfvars file, e.g. `project = "example"`.
var tfVarsLinePattern = regexp.MustCompile(`^\s*(\w+)\s*=\s*"([^"]*)"`)

// readTFVarsString reads the value of a simple string assignment
// (key = "value") from a .tfvars file, returning "" if key isn't found.
func readTFVarsString(path, key string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}

	for _, line := range strings.Split(string(data), "\n") {
		if m := tfVarsLinePattern.FindStringSubmatch(line); m != nil && m[1] == key {
			return m[2], nil
		}
	}
	return "", nil
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
