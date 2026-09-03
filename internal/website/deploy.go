// Package website builds and deploys a Vue-based website project
// following the company's vue-webapp-standard convention (a Vite build
// under website/, versioned via VITE_APP_VERSION and the footer
// SiteVersion component) to the S3 bucket + CloudFront distribution of
// one environment, read from that environment's Terraform state.
package website

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// DeployOptions configures Deploy.
type DeployOptions struct {
	// Env is "sandbox" or "production".
	Env string
	// Tag is the release tag to deploy, required and validated as
	// "website/vX.Y.Z" for production; must be empty for every other
	// environment.
	Tag string
	// Force skips the already-deployed/downgrade guards that otherwise
	// protect a production deploy.
	Force bool

	// InfraDir and WebsiteDir are relative to the git repository root
	// (found via git.Root, regardless of the current working
	// directory), matching the "infrastructure"/"website" convention.
	// Empty means the default.
	InfraDir   string
	WebsiteDir string
	// TerraformBin is the terraform binary to invoke. Empty means
	// "terraform".
	TerraformBin string

	Stdout, Stderr io.Writer
}

const (
	versionMarkerPrefix = "_deploy-versions"
	currentMarkerKey    = versionMarkerPrefix + "/current"
)

// productionTagPattern matches a production release tag, capturing its
// bare "X.Y.Z" version.
var productionTagPattern = regexp.MustCompile(`^website/v(\d+\.\d+\.\d+)$`)

// Deploy builds the website and syncs it to the S3 bucket + CloudFront
// distribution of the given environment, read from that environment's
// Terraform state (never guessed or hardcoded).
//
// In production, it deploys the commit pointed at by an explicit
// "website/vX.Y.Z" tag (opts.Tag), not necessarily the currently
// checked-out commit: the version is injected into the build via
// VITE_APP_VERSION and, once deployed, recorded as an empty marker
// object in S3 (one per ever-released tag) plus a "_deploy-versions/
// current" object naming the currently live tag. Redeploying an
// already-deployed tag, or deploying one older than the current live
// tag, is refused unless opts.Force is set. Every other environment
// deploys whatever commit is currently checked out, versioned by its
// short SHA, and accepts neither Tag nor Force.
func Deploy(opts DeployOptions) error {
	version, err := validateDeployArgs(opts.Env, opts.Tag, opts.Force)
	if err != nil {
		return err
	}

	websiteDirName := defaultString(opts.WebsiteDir, "website")
	terraformBin := defaultString(opts.TerraformBin, "terraform")

	repoRoot, liveDir, bucket, err := resolveBucket(opts.Env, opts.InfraDir, terraformBin, opts.Stdout)
	if err != nil {
		return err
	}

	if opts.Env != "production" {
		sha, err := gitOutput(repoRoot, "rev-parse", "--short", "HEAD")
		if err != nil {
			return err
		}
		version = strings.TrimSpace(sha)
	}

	distributionID, err := terraformStateAttr(terraformBin, liveDir, "module.website_hosting.aws_cloudfront_distribution.this", "id")
	if err != nil {
		return err
	}
	if distributionID == "" {
		return fmt.Errorf(
			"could not read the CloudFront distribution from the %q Terraform state (has 'kitsu terraform apply --env %s' been run for the website_hosting module?)",
			opts.Env, opts.Env,
		)
	}

	// Bail out here, before resolving the tag's commit or doing any
	// build work, if this tag was already released or is older than
	// what's currently live: the Terraform state read above is the only
	// thing that has to happen first, since it's where the bucket to
	// check comes from.
	if opts.Env == "production" {
		done, err := checkProductionDeployGuards(opts, bucket, version)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}

	var deployCommit string
	if opts.Env == "production" {
		fmt.Fprintf(opts.Stdout, "==> [%s] Resolving tag %s\n", opts.Env, opts.Tag)
		if _, err := gitOutput(repoRoot, "rev-parse", "-q", "--verify", "refs/tags/"+opts.Tag); err != nil {
			return fmt.Errorf("tag %q not found locally: fetch it first, e.g. git fetch origin tag %s", opts.Tag, opts.Tag)
		}
		out, err := gitOutput(repoRoot, "rev-list", "-n1", opts.Tag)
		if err != nil {
			return err
		}
		deployCommit = strings.TrimSpace(out)
	} else {
		out, err := gitOutput(repoRoot, "rev-parse", "HEAD")
		if err != nil {
			return err
		}
		deployCommit = strings.TrimSpace(out)
	}

	// The contact_api module is optional: an environment without it yet
	// still builds and deploys, just with the contact form shipping
	// disabled.
	contactAPIDomain, err := terraformStateAttr(terraformBin, liveDir, "module.contact_api.aws_api_gateway_domain_name.this", "domain_name")
	if err != nil {
		return err
	}

	var buildEnv []string
	if contactAPIDomain != "" {
		fmt.Fprintf(opts.Stdout, "==> [%s] Contact API endpoint: https://%s\n", opts.Env, contactAPIDomain)
		buildEnv = append(buildEnv, "VITE_CONTACT_API_URL=https://"+contactAPIDomain)
	} else {
		fmt.Fprintf(opts.Stderr, "⚠ [%s] contact_api module not found in Terraform state: the contact form will ship disabled.\n", opts.Env)
	}

	buildDir := repoRoot
	if opts.Env == "production" {
		tmpDir, err := os.MkdirTemp("", "kitsu-website-deploy-")
		if err != nil {
			return err
		}
		defer os.RemoveAll(tmpDir)

		fmt.Fprintf(opts.Stdout, "==> [%s] Checking out tag %s (commit %s) into a clean worktree\n", opts.Env, opts.Tag, shortCommit(deployCommit))
		if out, err := gitOutput(repoRoot, "worktree", "add", "--detach", "--quiet", tmpDir, deployCommit); err != nil {
			return fmt.Errorf("git worktree add: %w (%s)", err, out)
		}
		defer func() {
			_ = exec.Command("git", "-C", repoRoot, "worktree", "remove", "--force", tmpDir).Run()
		}()

		buildDir = tmpDir
	}
	websiteDir := filepath.Join(buildDir, websiteDirName)

	fmt.Fprintf(opts.Stdout, "==> [%s] Installing dependencies\n", opts.Env)
	if err := runIn(websiteDir, opts, nil, "npm", "ci"); err != nil {
		return err
	}

	fmt.Fprintf(opts.Stdout, "==> [%s] Running unit tests\n", opts.Env)
	if err := runIn(websiteDir, opts, nil, "npm", "run", "test"); err != nil {
		return err
	}

	fmt.Fprintf(opts.Stdout, "==> [%s] Building production bundle (version %s)\n", opts.Env, version)
	if err := runIn(websiteDir, opts, append(buildEnv, "VITE_APP_VERSION="+version), "npm", "run", "build"); err != nil {
		return err
	}

	distDir := filepath.Join(websiteDir, "dist")
	fmt.Fprintf(opts.Stdout, "==> [%s] Syncing dist/ to s3://%s\n", opts.Env, bucket)
	// Hashed assets are cached forever (safe: the filename changes on
	// every content change); index.html is never cached, so a deploy is
	// visible immediately. Version markers live outside dist/, excluded
	// from the --delete sync so past releases are never pruned by it.
	if err := runIn("", opts, nil, "aws", "s3", "sync", distDir+"/", "s3://"+bucket+"/", "--delete",
		"--cache-control", "public, max-age=31536000, immutable",
		"--exclude", "index.html",
		"--exclude", versionMarkerPrefix+"/*",
	); err != nil {
		return err
	}
	if err := runIn("", opts, nil, "aws", "s3", "cp", filepath.Join(distDir, "index.html"), "s3://"+bucket+"/index.html",
		"--cache-control", "public, max-age=0, must-revalidate",
	); err != nil {
		return err
	}

	fmt.Fprintf(opts.Stdout, "==> [%s] Invalidating CloudFront distribution %s\n", opts.Env, distributionID)
	invalidationID, err := commandOutput("aws", "cloudfront", "create-invalidation",
		"--distribution-id", distributionID, "--paths", "/*", "--query", "Invalidation.Id", "--output", "text")
	if err != nil {
		return err
	}
	invalidationID = strings.TrimSpace(invalidationID)

	if opts.Env == "production" {
		if err := recordProductionDeploy(opts, bucket); err != nil {
			return err
		}
	}

	fmt.Fprintf(opts.Stdout, "==> Done. Deployed version %s (commit %s) to %s at %s (invalidation %s).\n",
		version, shortCommit(deployCommit), opts.Env, time.Now().UTC().Format("2006-01-02T15:04:05Z"), invalidationID)

	return nil
}

// validateDeployArgs checks the env/tag/force combination, mirroring the
// original script's argument validation. For "production" it also
// returns the bare "X.Y.Z" version extracted from tag; for every other
// environment it returns "" (Deploy fills that in from git afterwards).
func validateDeployArgs(env, tag string, force bool) (version string, err error) {
	if env != "sandbox" && env != "production" {
		return "", fmt.Errorf("env must be either \"sandbox\" or \"production\", got %q", env)
	}

	if env == "production" {
		m := productionTagPattern.FindStringSubmatch(tag)
		if m == nil {
			return "", fmt.Errorf("tag must match \"website/vX.Y.Z\", got %q", tag)
		}
		return m[1], nil
	}

	if tag != "" || force {
		return "", fmt.Errorf("a tag/--force is only accepted for the \"production\" environment (got env=%q tag=%q force=%v)", env, tag, force)
	}
	return "", nil
}

// checkProductionDeployGuards enforces the already-deployed/downgrade
// rules for a production deploy of opts.Tag (version). It returns
// done=true when Deploy should stop (already deployed, opts.Force not
// set), or an error when the deploy is a refused downgrade.
func checkProductionDeployGuards(opts DeployOptions, bucket, version string) (done bool, err error) {
	alreadyDeployed, err := s3ObjectExists(bucket, versionMarkerPrefix+"/"+opts.Tag)
	if err != nil {
		return false, err
	}

	currentTag, err := s3ObjectContent(bucket, currentMarkerKey)
	if err != nil {
		return false, err
	}

	isDowngrade := false
	var currentVersion string
	if currentTag != "" && currentTag != opts.Tag {
		if m := productionTagPattern.FindStringSubmatch(currentTag); m != nil {
			currentVersion = m[1]
			isDowngrade = compareVersions(version, currentVersion) < 0
		}
	}

	switch {
	case opts.Force:
		fmt.Fprintf(opts.Stdout, "==> [%s] --force: skipping the already-deployed / downgrade checks for %s\n", opts.Env, opts.Tag)
		return false, nil
	case alreadyDeployed:
		fmt.Fprintf(opts.Stdout, "==> [%s] Tag %s (version %s) is already deployed.\n", opts.Env, opts.Tag, version)
		fmt.Fprintln(opts.Stdout, "    Nothing to do: pass a new 'website/vX.Y.Z' tag to release again, or")
		fmt.Fprintln(opts.Stdout, "    re-run with --force to redeploy it anyway (e.g. a rollback).")
		return true, nil
	case isDowngrade:
		return false, fmt.Errorf(
			"[%s] %s (version %s) is older than the currently deployed %s (version %s); re-run with --force to deploy it anyway (e.g. a deliberate rollback)",
			opts.Env, opts.Tag, version, currentTag, currentVersion,
		)
	default:
		return false, nil
	}
}

// recordProductionDeploy writes this deploy's version marker and updates
// the "current" pointer to opts.Tag.
func recordProductionDeploy(opts DeployOptions, bucket string) error {
	fmt.Fprintf(opts.Stdout, "==> [%s] Recording version marker %s/%s\n", opts.Env, versionMarkerPrefix, opts.Tag)
	if err := runIn("", opts, nil, "aws", "s3api", "put-object", "--bucket", bucket, "--key", versionMarkerPrefix+"/"+opts.Tag); err != nil {
		return err
	}

	fmt.Fprintf(opts.Stdout, "==> [%s] Updating %s to %s\n", opts.Env, currentMarkerKey, opts.Tag)
	markerFile, err := os.CreateTemp("", "kitsu-current-marker-")
	if err != nil {
		return err
	}
	defer os.Remove(markerFile.Name())
	if _, err := markerFile.WriteString(opts.Tag); err != nil {
		markerFile.Close()
		return err
	}
	if err := markerFile.Close(); err != nil {
		return err
	}

	return runIn("", opts, nil, "aws", "s3api", "put-object", "--bucket", bucket, "--key", currentMarkerKey, "--body", markerFile.Name())
}

// defaultString returns v, or fallback if v is empty.
func defaultString(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

// shortCommit returns commit's first 7 characters, or commit itself if
// shorter.
func shortCommit(commit string) string {
	if len(commit) > 7 {
		return commit[:7]
	}
	return commit
}

// compareVersions compares two bare "X.Y.Z" version strings, returning
// <0, 0 or >0 as a<b, a==b, a>b. Missing or non-numeric segments compare
// as 0.
func compareVersions(a, b string) int {
	pa, pb := versionParts(a), versionParts(b)
	for i := range pa {
		if pa[i] != pb[i] {
			return pa[i] - pb[i]
		}
	}
	return 0
}

func versionParts(v string) [3]int {
	var parts [3]int
	for i, s := range strings.SplitN(v, ".", 3) {
		if i >= len(parts) {
			break
		}
		parts[i], _ = strconv.Atoi(s)
	}
	return parts
}

// gitOutput runs `git -C repoRoot <args...>` and returns its stdout.
func gitOutput(repoRoot string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return out.String(), nil
}

// runIn runs `name <args...>`, in dir (unless empty, meaning inherit the
// current working directory), with extraEnv appended to the current
// environment, streaming its output through opts.Stdout/Stderr.
func runIn(dir string, opts DeployOptions, extraEnv []string, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if len(extraEnv) > 0 {
		cmd.Env = append(os.Environ(), extraEnv...)
	}
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%s %s: %w", name, strings.Join(args, " "), err)
	}
	return nil
}

// commandOutput runs `name <args...>` and returns its stdout.
func commandOutput(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("%s %s: %s", name, strings.Join(args, " "), msg)
	}
	return out.String(), nil
}

// stateAttrPattern matches a `terraform state show` attribute line, e.g.
// `  id = "bucket-name"`.
func stateAttrPattern(attr string) *regexp.Regexp {
	return regexp.MustCompile(`(?m)^\s*` + regexp.QuoteMeta(attr) + `\s*=\s*"([^"]*)"`)
}

// terraformStateAttr reads attr off address in the Terraform state shown
// from liveDir. It returns "" (not an error) if address isn't in the
// state, or attr isn't found on it — the same "optional lookup" contract
// used for e.g. the contact_api module, which may not exist yet.
func terraformStateAttr(terraformBin, liveDir, address, attr string) (string, error) {
	cmd := exec.Command(terraformBin, "state", "show", address)
	cmd.Dir = liveDir
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return "", nil
		}
		return "", fmt.Errorf("terraform state show %s: %w", address, err)
	}

	m := stateAttrPattern(attr).FindSubmatch(out)
	if m == nil {
		return "", nil
	}
	return string(m[1]), nil
}

// s3ObjectExists reports whether key exists in bucket.
func s3ObjectExists(bucket, key string) (bool, error) {
	err := exec.Command("aws", "s3api", "head-object", "--bucket", bucket, "--key", key).Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return false, nil
	}
	return false, fmt.Errorf("aws s3api head-object: %w", err)
}

// s3ObjectContent returns the content of key in bucket, or "" (not an
// error) if it doesn't exist.
func s3ObjectContent(bucket, key string) (string, error) {
	cmd := exec.Command("aws", "s3", "cp", "s3://"+bucket+"/"+key, "-")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err == nil {
		return out.String(), nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		return "", nil
	}
	return "", fmt.Errorf("aws s3 cp: %w", err)
}
