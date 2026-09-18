package terraform

import (
	"os"
	"os/exec"
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
	wantEnvironmentTFVars := "environment             = \"production\"\n" +
		"aws_account_id          = \"123456789012\"\n" +
		"aws_assume_role_enabled = true\n"
	if string(environmentTFVars) != wantEnvironmentTFVars {
		t.Errorf("environment.tfvars =\n%s\nwant:\n%s", environmentTFVars, wantEnvironmentTFVars)
	}

	backendHCL, err := os.ReadFile(filepath.Join(r.Env.Dir(), "backend.hcl"))
	if err != nil {
		t.Fatalf("read backend.hcl: %v", err)
	}
	wantBackendHCL := "bucket       = \"terraform-state-123456789012-eu-south-1-an\"\n" +
		"key          = \"acme-website/production/terraform.tfstate\"\n" +
		"region       = \"eu-south-1\"\n" +
		"use_lockfile = true\n" +
		"encrypt      = true\n" +
		"\n" +
		"assume_role = {\n" +
		"  role_arn = \"arn:aws:iam::123456789012:role/AdminRole\"\n" +
		"}\n"
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

// newTestInfraRepo creates a local git repository, seeded with two
// commits (the second tagged "v2"), that stands in for an existing
// project's Terraform tree: git clone works the same against a local
// path as against a real remote URL.
func newTestInfraRepo(t *testing.T) string {
	t.Helper()

	repo := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}

	run("init", "--quiet")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")

	write := func(path, content string) {
		full := filepath.Join(repo, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir for %q: %v", full, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %q: %v", full, err)
		}
	}

	write("live/main.tf", "# v1\n")
	run("add", ".")
	run("commit", "--quiet", "-m", "v1")
	run("tag", "-a", "v1", "-m", "v1")

	write("live/main.tf", "# v2\n")
	run("add", ".")
	run("commit", "--quiet", "-m", "v2")
	run("tag", "-a", "v2", "-m", "v2")

	return repo
}

func TestRunner_ScaffoldInfra_CreatesFreshDir(t *testing.T) {
	repo := newTestInfraRepo(t)
	infraDir := filepath.Join(t.TempDir(), "infrastructure")
	r := Runner{Env: Env{InfraDir: infraDir}, Stdout: &captureWriter{}}

	if err := r.ScaffoldInfra(repo, "", false); err != nil {
		t.Fatalf("ScaffoldInfra: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(infraDir, "live", "main.tf"))
	if err != nil {
		t.Fatalf("read live/main.tf: %v", err)
	}
	if string(content) != "# v2\n" {
		t.Errorf("live/main.tf = %q, want the v2 content", content)
	}
	if _, err := os.Stat(filepath.Join(infraDir, ".git")); !os.IsNotExist(err) {
		t.Errorf(".git should not have been copied into %s", infraDir)
	}
}

func TestRunner_ScaffoldInfra_RefusesNonEmptyWithoutForce(t *testing.T) {
	repo := newTestInfraRepo(t)
	infraDir := t.TempDir()
	existing := filepath.Join(infraDir, "environments", "production", "environment.tfvars")
	if err := os.MkdirAll(filepath.Dir(existing), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(existing, []byte("environment = \"production\"\n"), 0o644); err != nil {
		t.Fatalf("write %q: %v", existing, err)
	}
	r := Runner{Env: Env{InfraDir: infraDir}, Stdout: &captureWriter{}}

	if err := r.ScaffoldInfra(repo, "", false); err == nil {
		t.Error("ScaffoldInfra: expected an error for a non-empty infra-dir without --force, got nil")
	}

	if _, err := os.Stat(existing); err != nil {
		t.Errorf("existing content should not have been touched: %v", err)
	}
}

func TestRunner_ScaffoldInfra_ForceReplaces(t *testing.T) {
	repo := newTestInfraRepo(t)
	infraDir := t.TempDir()
	stale := filepath.Join(infraDir, "environments", "production", "environment.tfvars")
	if err := os.MkdirAll(filepath.Dir(stale), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(stale, []byte("environment = \"production\"\n"), 0o644); err != nil {
		t.Fatalf("write %q: %v", stale, err)
	}
	r := Runner{Env: Env{InfraDir: infraDir}, Stdout: &captureWriter{}}

	if err := r.ScaffoldInfra(repo, "", true); err != nil {
		t.Fatalf("ScaffoldInfra: %v", err)
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("expected %s to have been wiped by --force", stale)
	}
	if _, err := os.Stat(filepath.Join(infraDir, "live", "main.tf")); err != nil {
		t.Errorf("expected live/main.tf to have been copied: %v", err)
	}
}

func TestRunner_ScaffoldInfra_Ref(t *testing.T) {
	repo := newTestInfraRepo(t)
	infraDir := t.TempDir()
	r := Runner{Env: Env{InfraDir: infraDir}, Stdout: &captureWriter{}}

	if err := r.ScaffoldInfra(repo, "v1", true); err != nil {
		t.Fatalf("ScaffoldInfra: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(infraDir, "live", "main.tf"))
	if err != nil {
		t.Fatalf("read live/main.tf: %v", err)
	}
	if string(content) != "# v1\n" {
		t.Errorf("live/main.tf = %q, want the v1 content (pinned by --ref), not HEAD's", content)
	}
}
