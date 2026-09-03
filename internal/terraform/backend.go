package terraform

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
)

// StateBucketName returns the S3 bucket name Terraform state for
// accountID in region is stored under: the AWS account-regional
// namespace suffix ("-an") makes the name permanently reserved for that
// account, unlike the global namespace. Shared by BootstrapBackend
// (which creates the bucket) and ScaffoldEnvironment (which writes it
// into backend.hcl), so both always agree on the same name.
func StateBucketName(accountID, region string) string {
	return fmt.Sprintf("terraform-state-%s-%s-an", accountID, region)
}

// BootstrapBackendOptions configures BootstrapBackend.
type BootstrapBackendOptions struct {
	// Region is the AWS region the state bucket lives in.
	Region string
	// NoncurrentVersionExpirationDays controls how long old state
	// versions are kept before the lifecycle rule deletes them.
	NoncurrentVersionExpirationDays int

	Stdout, Stderr io.Writer
}

// BootstrapBackend creates, once and idempotently, the S3 bucket for the
// Terraform state shared by every Terraform project of one AWS account:
// versioning, default encryption, a public access block, an HTTPS-only
// bucket policy, and a lifecycle rule expiring noncurrent versions. State
// locking uses the native S3 lockfile (backend.hcl's use_lockfile = true,
// Terraform >= 1.10): no DynamoDB table is created.
//
// It always operates on the AWS account of the currently active
// credentials (aws sts get-caller-identity) — never an account id passed
// by hand — so run it once per account, with that account's credentials
// active, before the first `kitsu terraform init` there. If the bucket
// already exists, it only verifies/realigns its configuration.
func BootstrapBackend(opts BootstrapBackendOptions) error {
	accountID, err := awsCallerAccountID()
	if err != nil {
		return err
	}

	bucket := StateBucketName(accountID, opts.Region)
	fmt.Fprintf(opts.Stdout, "==> Target bucket: s3://%s (account %s, region %s)\n", bucket, accountID, opts.Region)

	exists, err := s3BucketExists(bucket, opts.Region)
	if err != nil {
		return err
	}
	if exists {
		fmt.Fprintln(opts.Stdout, "  ⤳ Bucket already exists, realigning the configuration (idempotent).")
	} else {
		fmt.Fprintln(opts.Stdout, "==> Creating bucket (account regional namespace)")
		if err := awsRun(opts, "s3api", "create-bucket",
			"--bucket", bucket,
			"--bucket-namespace", "account-regional",
			"--region", opts.Region,
			"--create-bucket-configuration", "LocationConstraint="+opts.Region,
		); err != nil {
			return err
		}
	}

	fmt.Fprintln(opts.Stdout, "==> Public access block (mandatory, the state contains sensitive data)")
	if err := awsRun(opts, "s3api", "put-public-access-block",
		"--bucket", bucket,
		"--public-access-block-configuration",
		"BlockPublicAcls=true,IgnorePublicAcls=true,BlockPublicPolicy=true,RestrictPublicBuckets=true",
	); err != nil {
		return err
	}

	fmt.Fprintln(opts.Stdout, "==> Ownership controls (disables ACLs, bucket policy only)")
	if err := awsRun(opts, "s3api", "put-bucket-ownership-controls",
		"--bucket", bucket,
		"--ownership-controls", "Rules=[{ObjectOwnership=BucketOwnerEnforced}]",
	); err != nil {
		return err
	}

	fmt.Fprintln(opts.Stdout, "==> Versioning (protection against corruption/accidental loss of the state)")
	if err := awsRun(opts, "s3api", "put-bucket-versioning",
		"--bucket", bucket,
		"--versioning-configuration", "Status=Enabled",
	); err != nil {
		return err
	}

	fmt.Fprintln(opts.Stdout, "==> Default server-side encryption (AES256 + bucket key)")
	if err := awsRun(opts, "s3api", "put-bucket-encryption",
		"--bucket", bucket,
		"--server-side-encryption-configuration",
		`{"Rules":[{"ApplyServerSideEncryptionByDefault":{"SSEAlgorithm":"AES256"},"BucketKeyEnabled":true}]}`,
	); err != nil {
		return err
	}

	fmt.Fprintln(opts.Stdout, "==> Bucket policy: explicitly deny non-HTTPS traffic")
	policy := fmt.Sprintf(
		`{"Version":"2012-10-17","Statement":[{"Sid":"DenyInsecureTransport","Effect":"Deny","Principal":"*","Action":"s3:*","Resource":["arn:aws:s3:::%s","arn:aws:s3:::%s/*"],"Condition":{"Bool":{"aws:SecureTransport":"false"}}}]}`,
		bucket, bucket,
	)
	if err := awsRun(opts, "s3api", "put-bucket-policy", "--bucket", bucket, "--policy", policy); err != nil {
		return err
	}

	fmt.Fprintf(opts.Stdout, "==> Lifecycle: delete noncurrent versions after %d days and incomplete multipart uploads\n", opts.NoncurrentVersionExpirationDays)
	lifecycle := fmt.Sprintf(
		`{"Rules":[{"ID":"expire-noncurrent-state-versions","Status":"Enabled","Filter":{},"NoncurrentVersionExpiration":{"NoncurrentDays":%d},"AbortIncompleteMultipartUpload":{"DaysAfterInitiation":7}}]}`,
		opts.NoncurrentVersionExpirationDays,
	)
	if err := awsRun(opts, "s3api", "put-bucket-lifecycle-configuration", "--bucket", bucket, "--lifecycle-configuration", lifecycle); err != nil {
		return err
	}

	fmt.Fprintln(opts.Stdout, "==> Tags (Project, ManagedBy)")
	tagging := `{"TagSet":[{"Key":"Project","Value":"terraform-state"},{"Key":"ManagedBy","Value":"kitsu"}]}`
	if err := awsRun(opts, "s3api", "put-bucket-tagging", "--bucket", bucket, "--tagging", tagging); err != nil {
		return err
	}

	fmt.Fprintf(opts.Stdout, "\n✓ Bucket ready: s3://%s\n\n", bucket)
	fmt.Fprintln(opts.Stdout, "Values to use in the environment's backend.hcl (different key per project):")
	fmt.Fprintf(opts.Stdout, "  bucket       = %q\n", bucket)
	fmt.Fprintln(opts.Stdout, `  key          = "<project>/terraform.tfstate"`)
	fmt.Fprintf(opts.Stdout, "  region       = %q\n", opts.Region)
	fmt.Fprintln(opts.Stdout, "  use_lockfile = true")
	fmt.Fprintln(opts.Stdout, "  encrypt      = true")

	return nil
}

// awsCallerAccountID returns the AWS account id of the currently active
// credentials.
func awsCallerAccountID() (string, error) {
	cmd := exec.Command("aws", "sts", "get-caller-identity", "--query", "Account", "--output", "text")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("aws sts get-caller-identity: %w", err)
	}
	return strings.TrimSpace(out.String()), nil
}

// s3BucketExists reports whether bucket exists (and is accessible) in
// region.
func s3BucketExists(bucket, region string) (bool, error) {
	cmd := exec.Command("aws", "s3api", "head-bucket", "--bucket", bucket, "--region", region)
	err := cmd.Run()
	if err == nil {
		return true, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// head-bucket exits non-zero when the bucket doesn't exist (or
		// isn't accessible with the current credentials).
		return false, nil
	}
	return false, fmt.Errorf("aws s3api head-bucket: %w", err)
}

// awsRun runs `aws <args...>`, streaming its output through opts.
func awsRun(opts BootstrapBackendOptions, args ...string) error {
	cmd := exec.Command("aws", args...)
	cmd.Stdout = opts.Stdout
	cmd.Stderr = opts.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("aws %s: %w", strings.Join(args, " "), err)
	}
	return nil
}
