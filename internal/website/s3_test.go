package website

import (
	"os"
	"path/filepath"
	"testing"
)

// withStubAWS puts a fake `aws` CLI first on PATH for the duration of
// the test: `s3api head-object` exits 0 if headObjectExists, non-zero
// otherwise; `s3 cp <src> -` prints cpContent if cpExists, non-zero
// otherwise (matching a real "no such key" failure).
func withStubAWS(t *testing.T, headObjectExists bool, cpExists bool, cpContent string) {
	t.Helper()

	headExit := "1"
	if headObjectExists {
		headExit = "0"
	}
	cpExit := "1"
	if cpExists {
		cpExit = "0"
	}

	dir := t.TempDir()
	script := "#!/bin/sh\n" +
		"if [ \"$1 $2\" = \"s3api head-object\" ]; then exit " + headExit + "; fi\n" +
		"if [ \"$1 $2\" = \"s3 cp\" ]; then\n" +
		"  if [ " + cpExit + " -eq 0 ]; then printf '%s' '" + cpContent + "'; fi\n" +
		"  exit " + cpExit + "\n" +
		"fi\n" +
		"exit 0\n"
	if err := os.WriteFile(filepath.Join(dir, "aws"), []byte(script), 0o755); err != nil {
		t.Fatalf("write stub aws: %v", err)
	}

	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestS3ObjectExists(t *testing.T) {
	withStubAWS(t, true, false, "")

	got, err := s3ObjectExists("my-bucket", "_deploy-versions/website/v1.0.0")
	if err != nil {
		t.Fatalf("s3ObjectExists: %v", err)
	}
	if !got {
		t.Error("s3ObjectExists() = false, want true")
	}
}

func TestS3ObjectExists_NotFound(t *testing.T) {
	withStubAWS(t, false, false, "")

	got, err := s3ObjectExists("my-bucket", "_deploy-versions/website/v1.0.0")
	if err != nil {
		t.Fatalf("s3ObjectExists: %v", err)
	}
	if got {
		t.Error("s3ObjectExists() = true, want false")
	}
}

func TestS3ObjectContent(t *testing.T) {
	withStubAWS(t, false, true, "website/v1.0.0")

	got, err := s3ObjectContent("my-bucket", currentMarkerKey)
	if err != nil {
		t.Fatalf("s3ObjectContent: %v", err)
	}
	if want := "website/v1.0.0"; got != want {
		t.Errorf("s3ObjectContent() = %q, want %q", got, want)
	}
}

func TestS3ObjectContent_Missing(t *testing.T) {
	withStubAWS(t, false, false, "")

	got, err := s3ObjectContent("my-bucket", currentMarkerKey)
	if err != nil {
		t.Fatalf("s3ObjectContent: %v", err)
	}
	if got != "" {
		t.Errorf("s3ObjectContent() = %q, want \"\" for a missing object", got)
	}
}
