package website

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newStubAWSForHistory puts a fake `aws` CLI first on PATH: `s3api
// list-objects-v2` prints listOutput (raw JSON) verbatim, and `s3 cp
// <src> -` prints currentContent if non-empty (else fails, like a
// missing key).
func newStubAWSForHistory(t *testing.T, listOutput, currentContent string) {
	t.Helper()

	cpExit := "1"
	if currentContent != "" {
		cpExit = "0"
	}

	dir := t.TempDir()
	script := "#!/bin/sh\n" +
		"if [ \"$1 $2\" = \"s3api list-objects-v2\" ]; then cat <<'EOF'\n" + listOutput + "\nEOF\nexit 0; fi\n" +
		"if [ \"$1 $2\" = \"s3 cp\" ]; then\n" +
		"  if [ " + cpExit + " -eq 0 ]; then printf '%s' '" + currentContent + "'; fi\n" +
		"  exit " + cpExit + "\n" +
		"fi\n" +
		"exit 0\n"
	if err := os.WriteFile(filepath.Join(dir, "aws"), []byte(script), 0o755); err != nil {
		t.Fatalf("write stub aws: %v", err)
	}

	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

const historyListOutput = `{
  "Contents": [
    {"Key": "_deploy-versions/current", "LastModified": "2026-09-05T09:10:00.000Z"},
    {"Key": "_deploy-versions/website/v1.0.0", "LastModified": "2026-09-01T18:00:00.000Z"},
    {"Key": "_deploy-versions/website/v1.1.0", "LastModified": "2026-09-05T09:10:00.000Z"},
    {"Key": "_deploy-versions/website/v1.2.0", "LastModified": "2026-09-10T14:32:00.000Z"}
  ]
}`

func TestHistoryForBucket(t *testing.T) {
	newStubAWSForHistory(t, historyListOutput, "website/v1.1.0")

	var out bytes.Buffer
	if err := historyForBucket("acme-bucket", "production", &out); err != nil {
		t.Fatalf("historyForBucket: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3: %q", len(lines), out.String())
	}

	// Newest deploy first, regardless of version number ordering.
	if !strings.HasPrefix(lines[0], "website/v1.2.0") {
		t.Errorf("line 0 = %q, want it to start with website/v1.2.0 (most recently deployed)", lines[0])
	}
	if !strings.Contains(lines[1], "website/v1.1.0") || !strings.Contains(lines[1], "(current)") {
		t.Errorf("line 1 = %q, want website/v1.1.0 marked (current)", lines[1])
	}
	if !strings.HasPrefix(lines[2], "website/v1.0.0") {
		t.Errorf("line 2 = %q, want it to start with website/v1.0.0 (oldest deploy)", lines[2])
	}
	if strings.Contains(lines[0], "(current)") || strings.Contains(lines[2], "(current)") {
		t.Errorf("only the current version should be marked: %q", out.String())
	}
}

func TestHistoryForBucket_NoDeployments(t *testing.T) {
	newStubAWSForHistory(t, `{"Contents": []}`, "")

	var out bytes.Buffer
	if err := historyForBucket("acme-bucket", "production", &out); err != nil {
		t.Fatalf("historyForBucket: %v", err)
	}
	if !strings.Contains(out.String(), "No versions have been deployed") {
		t.Errorf("output = %q, want a 'no versions' message", out.String())
	}
}

func TestCurrentForBucket(t *testing.T) {
	newStubAWSForHistory(t, "", "website/v1.1.0")

	var out bytes.Buffer
	if err := currentForBucket("acme-bucket", "production", &out); err != nil {
		t.Fatalf("currentForBucket: %v", err)
	}
	if got, want := strings.TrimSpace(out.String()), "website/v1.1.0"; got != want {
		t.Errorf("stdout = %q, want %q", got, want)
	}
}

func TestCurrentForBucket_NoDeployments(t *testing.T) {
	newStubAWSForHistory(t, "", "")

	var out bytes.Buffer
	err := currentForBucket("acme-bucket", "production", &out)
	if err == nil {
		t.Error("currentForBucket: expected an error when no version has been deployed, got nil")
	}
	if out.String() != "" {
		t.Errorf("stdout = %q, want empty on error", out.String())
	}
}
