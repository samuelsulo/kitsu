// Package terraform runs the Terraform CLI against the directory layout
// conventions shared by kitsu's client projects: one Terraform root at
// <infra-dir>/live, configured per environment from files under
// <infra-dir>/environments/<env>/.
package terraform

import "path/filepath"

// Env locates the Terraform root and per-environment configuration for one
// invocation.
type Env struct {
	// Bin is the terraform binary to invoke. Defaults to "terraform".
	Bin string
	// InfraDir is the infrastructure root directory, relative to the
	// current working directory or absolute. Defaults to "infrastructure".
	InfraDir string
	// Name is the environment name (e.g. "sandbox", "production"),
	// matching a directory under <InfraDir>/environments/.
	Name string
}

// binary returns the terraform binary to invoke, defaulting to "terraform"
// when Bin is empty.
func (e Env) binary() string {
	if e.Bin == "" {
		return "terraform"
	}
	return e.Bin
}

// infraDir returns InfraDir, defaulting to "infrastructure" when empty.
func (e Env) infraDir() string {
	if e.InfraDir == "" {
		return "infrastructure"
	}
	return e.InfraDir
}

// LiveDir is the Terraform root, shared by every environment.
func (e Env) LiveDir() string {
	return filepath.Join(e.infraDir(), "live")
}

// Dir is this environment's configuration directory.
func (e Env) Dir() string {
	return filepath.Join(e.infraDir(), "environments", e.Name)
}

// BackendConfig is this environment's backend configuration file, passed
// to `terraform init -backend-config`.
func (e Env) BackendConfig() string {
	return filepath.Join(e.Dir(), "backend.hcl")
}

// VarFile is this environment's variables file, passed to Terraform
// commands as `-var-file`.
func (e Env) VarFile() string {
	return filepath.Join(e.Dir(), "environment.tfvars")
}

// PlanFile is where a saved execution plan for this environment lives
// between `plan` and `apply`.
func (e Env) PlanFile() string {
	return filepath.Join(e.Dir(), "tfplan")
}
