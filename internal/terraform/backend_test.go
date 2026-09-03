package terraform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStateBucketName(t *testing.T) {
	got := StateBucketName("123456789012", "eu-south-1")
	want := "terraform-state-123456789012-eu-south-1-an"
	if got != want {
		t.Errorf("StateBucketName() = %q, want %q", got, want)
	}
}

// withStubAWS puts a fake `aws` CLI first on PATH for the duration of the
// test: it answers `sts get-caller-identity` with accountID, `s3api
// head-bucket` with exit 0 if bucketExists (non-zero otherwise, like a
// real "bucket not found"), and logs every other invocation's arguments
// to logFile, always succeeding.
func withStubAWS(t *testing.T, accountID string, bucketExists bool, logFile string) {
	t.Helper()

	headBucketExit := "1"
	if bucketExists {
		headBucketExit = "0"
	}

	dir := t.TempDir()
	script := "#!/bin/sh\n" +
		"if [ \"$1 $2\" = \"sts get-caller-identity\" ]; then echo \"" + accountID + "\"; exit 0; fi\n" +
		"if [ \"$1 $2\" = \"s3api head-bucket\" ]; then exit " + headBucketExit + "; fi\n" +
		"{ echo \"ARGS=$*\"; echo '---'; } >> \"" + logFile + "\"\n" +
		"exit 0\n"
	if err := os.WriteFile(filepath.Join(dir, "aws"), []byte(script), 0o755); err != nil {
		t.Fatalf("write stub aws: %v", err)
	}

	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestBootstrapBackend_CreatesNewBucket(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	withStubAWS(t, "123456789012", false, logFile)

	var out captureWriter
	err := BootstrapBackend(BootstrapBackendOptions{
		Region:                          "eu-south-1",
		NoncurrentVersionExpirationDays: 90,
		Stdout:                          &out,
		Stderr:                          &out,
	})
	if err != nil {
		t.Fatalf("BootstrapBackend: %v", err)
	}

	invocations := readInvocations(t, logFile)
	var sawCreateBucket bool
	for _, inv := range invocations {
		if len(inv.args) > 1 && inv.args[0] == "s3api" && inv.args[1] == "create-bucket" {
			sawCreateBucket = true
		}
	}
	if !sawCreateBucket {
		t.Errorf("expected 's3api create-bucket' to be called; invocations: %+v", invocations)
	}

	if !strings.Contains(out.String(), "terraform-state-123456789012-eu-south-1-an") {
		t.Errorf("output = %q, want it to mention the bucket name", out.String())
	}
}

func TestBootstrapBackend_ExistingBucket_SkipsCreate(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	withStubAWS(t, "123456789012", true, logFile)

	var out captureWriter
	err := BootstrapBackend(BootstrapBackendOptions{
		Region:                          "eu-south-1",
		NoncurrentVersionExpirationDays: 90,
		Stdout:                          &out,
		Stderr:                          &out,
	})
	if err != nil {
		t.Fatalf("BootstrapBackend: %v", err)
	}

	for _, inv := range readInvocations(t, logFile) {
		if len(inv.args) > 1 && inv.args[0] == "s3api" && inv.args[1] == "create-bucket" {
			t.Error("did not expect 's3api create-bucket' to be called when the bucket already exists")
		}
	}
}
