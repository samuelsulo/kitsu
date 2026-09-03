package git

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRoot(t *testing.T) {
	repoRoot := t.TempDir()
	if err := exec.Command("git", "init", "--quiet", repoRoot).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	nested := filepath.Join(repoRoot, "a", "b")
	if err := exec.Command("mkdir", "-p", nested).Run(); err != nil {
		t.Fatalf("mkdir -p: %v", err)
	}

	got, err := Root(nested)
	if err != nil {
		t.Fatalf("Root(%q): %v", nested, err)
	}

	// Resolve symlinks (e.g. /tmp -> /private/tmp on macOS) on both sides
	// so the comparison isn't sensitive to the OS temp dir's real path.
	wantResolved, err := filepath.EvalSymlinks(repoRoot)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", repoRoot, err)
	}
	gotResolved, err := filepath.EvalSymlinks(got)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", got, err)
	}
	if gotResolved != wantResolved {
		t.Errorf("Root(%q) = %q, want %q", nested, gotResolved, wantResolved)
	}
}

func TestRoot_NotAGitRepository(t *testing.T) {
	dir := t.TempDir()

	if _, err := Root(dir); err == nil {
		t.Errorf("Root(%q): expected an error, got nil", dir)
	}
}
