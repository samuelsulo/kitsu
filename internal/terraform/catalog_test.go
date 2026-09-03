package terraform

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newTestCatalogRepo creates a local git repository, seeded with two
// tagged modules, that stands in for a real module catalog: git
// ls-remote/clone work the same against a local path as against a real
// remote URL.
func newTestCatalogRepo(t *testing.T) string {
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

	writeAndCommit := func(path, content, tag string) {
		full := filepath.Join(repo, path)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir for %q: %v", full, err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write %q: %v", full, err)
		}
		run("add", ".")
		run("commit", "--quiet", "-m", "add "+path)
		run("tag", tag)
	}

	writeAndCommit("modules/module-a/main.tf", "# v1.0.0\n", "module-a/v1.0.0")
	writeAndCommit("modules/module-a/main.tf", "# v1.2.0\n", "module-a/v1.2.0")
	writeAndCommit("modules/module-b/main.tf", "# v0.1.0\n", "module-b/v0.1.0")

	return repo
}

func TestRunner_CatalogList(t *testing.T) {
	repo := newTestCatalogRepo(t)
	var out captureWriter
	r := Runner{Stdout: &out}

	if err := r.CatalogList(repo); err != nil {
		t.Fatalf("CatalogList: %v", err)
	}

	got := out.String()
	for _, want := range []string{"module-a", "module-b"} {
		if !strings.Contains(got, want) {
			t.Errorf("CatalogList output = %q, want it to contain %q", got, want)
		}
	}
}

func TestRunner_CatalogVersions(t *testing.T) {
	repo := newTestCatalogRepo(t)
	var out captureWriter
	r := Runner{Stdout: &out}

	if err := r.CatalogVersions(repo, "module-a"); err != nil {
		t.Fatalf("CatalogVersions: %v", err)
	}

	want := "v1.2.0\nv1.0.0\n" // newest first
	if got := out.String(); got != want {
		t.Errorf("CatalogVersions output = %q, want %q", got, want)
	}
}

func TestRunner_CatalogVersions_UnknownModule(t *testing.T) {
	repo := newTestCatalogRepo(t)
	var out captureWriter
	r := Runner{Stdout: &out}

	if err := r.CatalogVersions(repo, "does-not-exist"); err == nil {
		t.Error("CatalogVersions: expected an error for an unknown module, got nil")
	}
}

func TestRunner_CatalogVendor(t *testing.T) {
	repo := newTestCatalogRepo(t)
	infraDir := t.TempDir()
	var out captureWriter
	r := Runner{Env: Env{InfraDir: infraDir}, Stdout: &out}

	if err := r.CatalogVendor(repo, "module-a", "v1.0.0"); err != nil {
		t.Fatalf("CatalogVendor: %v", err)
	}

	vendoredFile := filepath.Join(infraDir, "modules", "vendor", "module-a", "main.tf")
	content, err := os.ReadFile(vendoredFile)
	if err != nil {
		t.Fatalf("read %q: %v", vendoredFile, err)
	}
	if string(content) != "# v1.0.0\n" {
		t.Errorf("vendored main.tf = %q, want the v1.0.0 content", content)
	}

	if _, err := os.Stat(filepath.Join(infraDir, "modules", "vendor", "module-a", "VENDORED.md")); err != nil {
		t.Errorf("VENDORED.md not written: %v", err)
	}
}

func TestRunner_CatalogVendor_UnknownModule(t *testing.T) {
	repo := newTestCatalogRepo(t)
	var out captureWriter
	r := Runner{Env: Env{InfraDir: t.TempDir()}, Stdout: &out}

	if err := r.CatalogVendor(repo, "does-not-exist", "v1.0.0"); err == nil {
		t.Error("CatalogVendor: expected an error for an unknown module, got nil")
	}
}

// captureWriter is a minimal io.Writer that accumulates everything
// written to it, for asserting on command output in tests.
type captureWriter struct {
	data []byte
}

func (c *captureWriter) Write(p []byte) (int, error) {
	c.data = append(c.data, p...)
	return len(p), nil
}

func (c *captureWriter) String() string { return string(c.data) }
