package website

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/samuelsulo/kitsu/internal/git"
)

// resolveBucket finds the git repository root and initializes Terraform
// against env's backend, then reads the website_hosting module's S3
// bucket out of its state — the same lookup Deploy, Current and History
// all need before doing anything env-specific. progress receives the
// "reading infrastructure state" status line; callers that want a clean
// stdout (e.g. Current, meant to be captured in a script) should pass
// their stderr instead.
func resolveBucket(env, infraDir, terraformBin string, progress io.Writer) (repoRoot, liveDir, bucket string, err error) {
	repoRoot, err = git.Root("")
	if err != nil {
		return "", "", "", err
	}

	infraDir = defaultString(infraDir, "infrastructure")
	terraformBin = defaultString(terraformBin, "terraform")

	liveDir = filepath.Join(repoRoot, infraDir, "live")
	backendHCL := filepath.Join(repoRoot, infraDir, "environments", env, "backend.hcl")

	if _, err := os.Stat(backendHCL); err != nil {
		return "", "", "", fmt.Errorf("%s not found: environment %q does not exist yet", backendHCL, env)
	}

	fmt.Fprintf(progress, "==> [%s] Reading infrastructure endpoints from Terraform state\n", env)
	initCmd := exec.Command(terraformBin, "init", "-backend-config="+backendHCL, "-reconfigure")
	initCmd.Dir = liveDir
	if out, err := initCmd.CombinedOutput(); err != nil {
		return "", "", "", fmt.Errorf("terraform init: %w (%s)", err, strings.TrimSpace(string(out)))
	}

	bucket, err = terraformStateAttr(terraformBin, liveDir, "module.website_hosting.aws_s3_bucket.this", "id")
	if err != nil {
		return "", "", "", err
	}
	if bucket == "" {
		return "", "", "", fmt.Errorf(
			"could not read the S3 bucket from the %q Terraform state (has 'kitsu terraform apply --env %s' been run for the website_hosting module?)",
			env, env,
		)
	}

	return repoRoot, liveDir, bucket, nil
}
