package terraform

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ScaffoldEnvironment creates <infra-dir>/environments/<env>/{environment.tfvars,backend.hcl}
// for a new AWS account, reading 'project' and 'aws_region' from
// live/project.auto.tfvars. roleARNTemplate is a fmt template with a
// single %s for the AWS account id (e.g.
// "arn:aws:iam::%s:role/MyAdminRole").
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
	environmentTFVars := fmt.Sprintf(
		"environment    = %q\naws_account_id = %q\naws_role_arn   = %q\n",
		r.Env.Name, accountID, roleARN,
	)
	environmentTFVarsPath := filepath.Join(r.Env.Dir(), "environment.tfvars")
	if err := os.WriteFile(environmentTFVarsPath, []byte(environmentTFVars), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(r.Stdout, "✓ %s written (environment=%s, aws_account_id=%s)\n", environmentTFVarsPath, r.Env.Name, accountID)

	backendHCL := fmt.Sprintf(
		"bucket = %q\nkey = %q\nregion = %q\nuse_lockfile = true\nencrypt = true\n",
		StateBucketName(accountID, region),
		project+"/terraform.tfstate",
		region,
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
