package terraform

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// requireTerraform skips the test if a real terraform binary isn't
// available: Fmt/FmtCheck rely on `terraform fmt`'s actual formatting
// logic, which isn't worth reimplementing with a stub.
func requireTerraform(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("terraform"); err != nil {
		t.Skip("terraform binary not found in PATH, skipping")
	}
}

const unformattedTF = `resource "aws_s3_bucket" "this" {
    bucket  =  "example"
}
`

const unformattedHCL = `bucket   = "example"
region = "eu-south-1"
`

func TestRunner_Fmt(t *testing.T) {
	requireTerraform(t)

	infraDir := t.TempDir()
	tfFile := filepath.Join(infraDir, "live", "main.tf")
	hclFile := filepath.Join(infraDir, "environments", "sandbox", "backend.hcl")
	testHCLFile := filepath.Join(infraDir, "live", "fixtures.tftest.hcl")

	for _, f := range []string{tfFile, hclFile, testHCLFile} {
		if err := os.MkdirAll(filepath.Dir(f), 0o755); err != nil {
			t.Fatalf("mkdir for %q: %v", f, err)
		}
	}
	if err := os.WriteFile(tfFile, []byte(unformattedTF), 0o644); err != nil {
		t.Fatalf("write %q: %v", tfFile, err)
	}
	if err := os.WriteFile(hclFile, []byte(unformattedHCL), 0o644); err != nil {
		t.Fatalf("write %q: %v", hclFile, err)
	}
	if err := os.WriteFile(testHCLFile, []byte(unformattedHCL), 0o644); err != nil {
		t.Fatalf("write %q: %v", testHCLFile, err)
	}

	r := Runner{Env: Env{InfraDir: infraDir}, Stderr: os.Stderr}

	if err := r.Fmt(); err != nil {
		t.Fatalf("Fmt: %v", err)
	}

	if err := r.FmtCheck(); err != nil {
		t.Errorf("FmtCheck after Fmt: %v (files should now be formatted)", err)
	}

	// .tftest.hcl is skipped by the generic-HCL pass, but only because
	// Terraform's own `fmt -recursive` (the first step above) already
	// formats it: it should still come out formatted, not untouched.
	got, err := os.ReadFile(testHCLFile)
	if err != nil {
		t.Fatalf("read %q: %v", testHCLFile, err)
	}
	if string(got) == unformattedHCL {
		t.Error(".tftest.hcl was left unformatted, want it formatted by terraform fmt -recursive")
	}
}

func TestRunner_FmtCheck_DetectsUnformattedFile(t *testing.T) {
	requireTerraform(t)

	infraDir := t.TempDir()
	hclFile := filepath.Join(infraDir, "environments", "sandbox", "backend.hcl")
	if err := os.MkdirAll(filepath.Dir(hclFile), 0o755); err != nil {
		t.Fatalf("mkdir for %q: %v", hclFile, err)
	}
	if err := os.WriteFile(hclFile, []byte(unformattedHCL), 0o644); err != nil {
		t.Fatalf("write %q: %v", hclFile, err)
	}

	r := Runner{Env: Env{InfraDir: infraDir}, Stderr: os.Stderr}

	if err := r.FmtCheck(); err == nil {
		t.Error("FmtCheck: expected an error for an unformatted file, got nil")
	}
}
