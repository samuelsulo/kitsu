package website

import (
	"os"
	"path/filepath"
	"testing"
)

// newStubTerraformStateShow writes a fake terraform binary whose `state
// show <address>` prints output, when address == wantAddress (mimicking
// real `terraform state show` formatting), and fails otherwise (as it
// does for a resource address not present in the state). Any other
// subcommand is a no-op success.
func newStubTerraformStateShow(t *testing.T, wantAddress, output string) string {
	t.Helper()

	stubPath := filepath.Join(t.TempDir(), "terraform")
	script := "#!/bin/sh\n" +
		"if [ \"$1 $2\" = \"state show\" ]; then\n" +
		"  if [ \"$3\" = \"" + wantAddress + "\" ]; then\n" +
		"    cat <<'EOF'\n" + output + "\nEOF\n" +
		"    exit 0\n" +
		"  fi\n" +
		"  exit 1\n" +
		"fi\n" +
		"exit 0\n"
	if err := os.WriteFile(stubPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write stub terraform: %v", err)
	}
	return stubPath
}

func TestTerraformStateAttr_Found(t *testing.T) {
	stub := newStubTerraformStateShow(t, "module.website_hosting.aws_s3_bucket.this", `# module.website_hosting.aws_s3_bucket.this:
resource "aws_s3_bucket" "this" {
    id     = "acme-website-bucket"
    bucket = "acme-website-bucket"
}`)

	got, err := terraformStateAttr(stub, t.TempDir(), "module.website_hosting.aws_s3_bucket.this", "id")
	if err != nil {
		t.Fatalf("terraformStateAttr: %v", err)
	}
	if want := "acme-website-bucket"; got != want {
		t.Errorf("terraformStateAttr() = %q, want %q", got, want)
	}
}

func TestTerraformStateAttr_ResourceNotInState(t *testing.T) {
	stub := newStubTerraformStateShow(t, "module.website_hosting.aws_s3_bucket.this", "irrelevant")

	// A different address than the one the stub knows about: mimics a
	// resource that isn't in the state (e.g. the optional contact_api
	// module before it's ever been applied). Must be tolerated, not an
	// error.
	got, err := terraformStateAttr(stub, t.TempDir(), "module.contact_api.aws_api_gateway_domain_name.this", "domain_name")
	if err != nil {
		t.Fatalf("terraformStateAttr: %v", err)
	}
	if got != "" {
		t.Errorf("terraformStateAttr() = %q, want \"\" for a resource not in state", got)
	}
}

func TestTerraformStateAttr_AttrNotFound(t *testing.T) {
	stub := newStubTerraformStateShow(t, "module.website_hosting.aws_s3_bucket.this", `resource "aws_s3_bucket" "this" {
    bucket = "acme-website-bucket"
}`)

	got, err := terraformStateAttr(stub, t.TempDir(), "module.website_hosting.aws_s3_bucket.this", "id")
	if err != nil {
		t.Fatalf("terraformStateAttr: %v", err)
	}
	if got != "" {
		t.Errorf("terraformStateAttr() = %q, want \"\" when the attribute isn't present", got)
	}
}
