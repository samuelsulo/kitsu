package githooks

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstall(t *testing.T) {
	repoRoot := t.TempDir()
	if err := exec.Command("git", "init", "--quiet", repoRoot).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	hooksDir := filepath.Join(repoRoot, ".githooks")
	if err := os.Mkdir(hooksDir, 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", hooksDir, err)
	}
	hookFile := filepath.Join(hooksDir, "commit-msg")
	if err := os.WriteFile(hookFile, []byte("#!/bin/sh\nexit 0\n"), 0o644); err != nil {
		t.Fatalf("write %q: %v", hookFile, err)
	}

	if err := Install(repoRoot, ".githooks"); err != nil {
		t.Fatalf("Install: %v", err)
	}

	out, err := exec.Command("git", "-C", repoRoot, "config", "core.hooksPath").Output()
	if err != nil {
		t.Fatalf("git config core.hooksPath: %v", err)
	}
	if got := strings.TrimSpace(string(out)); got != ".githooks" {
		t.Errorf("core.hooksPath = %q, want %q", got, ".githooks")
	}

	info, err := os.Stat(hookFile)
	if err != nil {
		t.Fatalf("stat %q: %v", hookFile, err)
	}
	if info.Mode()&executableBits == 0 {
		t.Errorf("hook file %q is not executable: mode=%v", hookFile, info.Mode())
	}
}

func TestInstall_MissingHooksDir(t *testing.T) {
	repoRoot := t.TempDir()
	if err := exec.Command("git", "init", "--quiet", repoRoot).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}

	if err := Install(repoRoot, ".githooks"); err == nil {
		t.Error("Install: expected an error for a missing hooks directory, got nil")
	}
}
