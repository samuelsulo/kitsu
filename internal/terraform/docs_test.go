package terraform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newStubBinary writes a fake executable named name that appends its
// arguments to logFile on every invocation. It returns the stub's path.
func newStubBinary(t *testing.T, name, logFile string) string {
	t.Helper()

	stubPath := filepath.Join(t.TempDir(), name)
	script := "#!/bin/sh\n" +
		"{ echo \"ARGS=$*\"; echo '---'; } >> \"" + logFile + "\"\n"
	if err := os.WriteFile(stubPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write stub binary %q: %v", name, err)
	}
	return stubPath
}

func TestRunner_Docs(t *testing.T) {
	infraDir := t.TempDir()
	for _, dir := range []string{
		filepath.Join(infraDir, "modules", "local", "website-hosting"),
		filepath.Join(infraDir, "modules", "vendor", "contact-api"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("mkdir %q: %v", dir, err)
		}
	}

	logFile := filepath.Join(t.TempDir(), "log")
	docsBin := newStubBinary(t, "terraform-docs", logFile)

	r := Runner{Env: Env{InfraDir: infraDir}}
	if err := r.Docs(docsBin); err != nil {
		t.Fatalf("Docs: %v", err)
	}

	invocations := readInvocations(t, logFile)
	if len(invocations) != 2 {
		t.Fatalf("got %d invocations, want 2 (one per module dir): %+v", len(invocations), invocations)
	}

	var gotDirs []string
	for _, inv := range invocations {
		wantPrefix := []string{"markdown", "table", "--output-file", "README.md", "--output-mode", "inject"}
		if strings.Join(inv.args[:len(wantPrefix)], " ") != strings.Join(wantPrefix, " ") {
			t.Errorf("args = %v, want it to start with %v", inv.args, wantPrefix)
		}
		gotDirs = append(gotDirs, inv.args[len(inv.args)-1])
	}

	for _, want := range []string{
		filepath.Join(infraDir, "modules", "local", "website-hosting"),
		filepath.Join(infraDir, "modules", "vendor", "contact-api"),
	} {
		found := false
		for _, got := range gotDirs {
			if got == want {
				found = true
			}
		}
		if !found {
			t.Errorf("module dir %q was not passed to terraform-docs (got dirs: %v)", want, gotDirs)
		}
	}
}

func TestRunner_Docs_MissingBinary(t *testing.T) {
	r := Runner{Env: Env{InfraDir: t.TempDir()}}

	if err := r.Docs(filepath.Join(t.TempDir(), "no-such-terraform-docs")); err == nil {
		t.Error("Docs: expected an error when terraform-docs isn't found, got nil")
	}
}
