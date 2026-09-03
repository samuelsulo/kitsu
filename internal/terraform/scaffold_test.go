package terraform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newScaffoldTestRunner(t *testing.T) Runner {
	t.Helper()

	root := t.TempDir()
	env := Env{InfraDir: filepath.Join(root, "infrastructure"), Name: "production"}
	if err := os.MkdirAll(env.LiveDir(), 0o755); err != nil {
		t.Fatalf("mkdir live dir: %v", err)
	}

	return Runner{Env: env, Stdout: &captureWriter{}}
}

func TestRunner_ScaffoldEnvironment(t *testing.T) {
	r := newScaffoldTestRunner(t)

	projectTFVars := filepath.Join(r.Env.LiveDir(), "project.auto.tfvars")
	if err := os.WriteFile(projectTFVars, []byte(`project    = "acme-website"
aws_region = "eu-south-1"
`), 0o644); err != nil {
		t.Fatalf("write %q: %v", projectTFVars, err)
	}

	if err := r.ScaffoldEnvironment("123456789012", "arn:aws:iam::%s:role/AdminRole"); err != nil {
		t.Fatalf("ScaffoldEnvironment: %v", err)
	}

	environmentTFVars, err := os.ReadFile(filepath.Join(r.Env.Dir(), "environment.tfvars"))
	if err != nil {
		t.Fatalf("read environment.tfvars: %v", err)
	}
	wantEnvironmentTFVars := "environment    = \"production\"\n" +
		"aws_account_id = \"123456789012\"\n" +
		"aws_role_arn   = \"arn:aws:iam::123456789012:role/AdminRole\"\n"
	if string(environmentTFVars) != wantEnvironmentTFVars {
		t.Errorf("environment.tfvars =\n%s\nwant:\n%s", environmentTFVars, wantEnvironmentTFVars)
	}

	backendHCL, err := os.ReadFile(filepath.Join(r.Env.Dir(), "backend.hcl"))
	if err != nil {
		t.Fatalf("read backend.hcl: %v", err)
	}
	wantBackendHCL := "bucket = \"terraform-state-123456789012-eu-south-1-an\"\n" +
		"key = \"acme-website/terraform.tfstate\"\n" +
		"region = \"eu-south-1\"\n" +
		"use_lockfile = true\n" +
		"encrypt = true\n"
	if string(backendHCL) != wantBackendHCL {
		t.Errorf("backend.hcl =\n%s\nwant:\n%s", backendHCL, wantBackendHCL)
	}
}

func TestRunner_ScaffoldEnvironment_MissingProjectTFVars(t *testing.T) {
	r := newScaffoldTestRunner(t)

	if err := r.ScaffoldEnvironment("123456789012", "arn:aws:iam::%s:role/AdminRole"); err == nil {
		t.Error("ScaffoldEnvironment: expected an error when project.auto.tfvars is missing, got nil")
	}
}

func TestRunner_ScaffoldEnvironment_IncompleteProjectTFVars(t *testing.T) {
	r := newScaffoldTestRunner(t)

	// Missing aws_region.
	projectTFVars := filepath.Join(r.Env.LiveDir(), "project.auto.tfvars")
	if err := os.WriteFile(projectTFVars, []byte(`project = "acme-website"`+"\n"), 0o644); err != nil {
		t.Fatalf("write %q: %v", projectTFVars, err)
	}

	if err := r.ScaffoldEnvironment("123456789012", "arn:aws:iam::%s:role/AdminRole"); err == nil {
		t.Error("ScaffoldEnvironment: expected an error when aws_region is missing, got nil")
	}
}

func TestRunner_ScaffoldModule(t *testing.T) {
	r := newScaffoldTestRunner(t)

	if err := r.ScaffoldModule("website-hosting"); err != nil {
		t.Fatalf("ScaffoldModule: %v", err)
	}

	dir := filepath.Join(r.Env.infraDir(), "modules", "local", "website-hosting")
	for _, f := range scaffoldModuleFiles {
		if _, err := os.Stat(filepath.Join(dir, f)); err != nil {
			t.Errorf("expected %q to be created: %v", f, err)
		}
	}
}

func TestRunner_ScaffoldModule_SkipsExistingFiles(t *testing.T) {
	r := newScaffoldTestRunner(t)

	dir := filepath.Join(r.Env.infraDir(), "modules", "local", "website-hosting")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", dir, err)
	}
	existing := filepath.Join(dir, "main.tf")
	if err := os.WriteFile(existing, []byte("# already here\n"), 0o644); err != nil {
		t.Fatalf("write %q: %v", existing, err)
	}

	if err := r.ScaffoldModule("website-hosting"); err != nil {
		t.Fatalf("ScaffoldModule: %v", err)
	}

	content, err := os.ReadFile(existing)
	if err != nil {
		t.Fatalf("read %q: %v", existing, err)
	}
	if string(content) != "# already here\n" {
		t.Errorf("existing main.tf was overwritten: got %q", content)
	}

	out, ok := r.Stdout.(*captureWriter)
	if !ok {
		t.Fatal("r.Stdout is not a *captureWriter")
	}
	if !strings.Contains(out.String(), "already exists, skipping") {
		t.Errorf("output = %q, want it to report main.tf as skipped", out.String())
	}
}
