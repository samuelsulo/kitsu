package terraform

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// newTestModuleCatalogRepo creates a local git repository shaped like the
// Terraform module catalog ReleaseModule operates on: one module with a
// CHANGELOG.md carrying "## Unreleased", and a modules/README.md index
// row for it.
func newTestModuleCatalogRepo(t *testing.T, module, currentVersion string) string {
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

	moduleDir := filepath.Join(repo, "modules", module)
	if err := os.MkdirAll(moduleDir, 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", moduleDir, err)
	}
	changelog := "# Changelog\n\n## Unreleased\n\n### Added\n\n- Something.\n"
	if err := os.WriteFile(filepath.Join(moduleDir, "CHANGELOG.md"), []byte(changelog), 0o644); err != nil {
		t.Fatalf("write CHANGELOG.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(moduleDir, "main.tf"), []byte("# placeholder\n"), 0o644); err != nil {
		t.Fatalf("write main.tf: %v", err)
	}

	readme := "# Modules\n\n| Module | Version |\n|---|---|\n" +
		"| [" + module + "](modules/" + module + ") | " + currentVersion + " |\n"
	if err := os.WriteFile(filepath.Join(repo, "modules", "README.md"), []byte(readme), 0o644); err != nil {
		t.Fatalf("write README.md: %v", err)
	}

	run("add", ".")
	run("commit", "--quiet", "-m", "seed module")

	return repo
}

func TestReleaseModule(t *testing.T) {
	repo := newTestModuleCatalogRepo(t, "website-hosting", "v1.0.0")
	t.Chdir(repo)

	var out, errOut bytes.Buffer
	err := ReleaseModule(ReleaseModuleOptions{
		Module:  "website-hosting",
		Version: "v1.1.0",
		Stdout:  &out,
		Stderr:  &errOut,
	})
	if err != nil {
		t.Fatalf("ReleaseModule: %v (stderr: %s)", err, errOut.String())
	}

	changelog, err := os.ReadFile(filepath.Join(repo, "modules", "website-hosting", "CHANGELOG.md"))
	if err != nil {
		t.Fatalf("read CHANGELOG.md: %v", err)
	}
	if !regexp.MustCompile(`(?m)^## \[1\.1\.0\] - \d{4}-\d{2}-\d{2}$`).MatchString(string(changelog)) {
		t.Errorf("CHANGELOG.md = %q, want a '## [1.1.0] - <date>' heading", changelog)
	}
	if strings.Contains(string(changelog), "## Unreleased") {
		t.Error("CHANGELOG.md still contains '## Unreleased'")
	}

	readme, err := os.ReadFile(filepath.Join(repo, "modules", "README.md"))
	if err != nil {
		t.Fatalf("read README.md: %v", err)
	}
	if !strings.Contains(string(readme), "| v1.1.0 |") {
		t.Errorf("README.md = %q, want it to contain '| v1.1.0 |'", readme)
	}

	// Working tree must be clean: everything staged and committed.
	status := gitOutput(t, repo, "status", "--porcelain")
	if status != "" {
		t.Errorf("git status not clean after ReleaseModule: %q", status)
	}

	subject := gitOutput(t, repo, "log", "-1", "--format=%s")
	if want := "docs(website-hosting): release v1.1.0"; subject != want {
		t.Errorf("commit subject = %q, want %q", subject, want)
	}

	tagType := gitOutput(t, repo, "cat-file", "-t", "website-hosting/v1.1.0")
	if tagType != "tag" {
		t.Errorf("website-hosting/v1.1.0 is not an annotated tag (cat-file -t = %q)", tagType)
	}
}

func TestReleaseModule_AlreadyPreparedEntry(t *testing.T) {
	repo := newTestModuleCatalogRepo(t, "website-hosting", "v1.0.0")
	changelogPath := filepath.Join(repo, "modules", "website-hosting", "CHANGELOG.md")
	prepared := "# Changelog\n\n## [1.1.0] - 2020-01-01\n\n### Added\n\n- Something.\n"
	if err := os.WriteFile(changelogPath, []byte(prepared), 0o644); err != nil {
		t.Fatalf("write prepared CHANGELOG.md: %v", err)
	}
	amendChangelog(t, repo, "prepare changelog by hand")

	t.Chdir(repo)

	var out, errOut bytes.Buffer
	err := ReleaseModule(ReleaseModuleOptions{
		Module:  "website-hosting",
		Version: "v1.1.0",
		Stdout:  &out,
		Stderr:  &errOut,
	})
	if err != nil {
		t.Fatalf("ReleaseModule: %v (stderr: %s)", err, errOut.String())
	}

	changelog, err := os.ReadFile(changelogPath)
	if err != nil {
		t.Fatalf("read CHANGELOG.md: %v", err)
	}
	if string(changelog) != prepared {
		t.Errorf("CHANGELOG.md was modified, want it left exactly as prepared:\ngot:  %q\nwant: %q", changelog, prepared)
	}
}

func TestReleaseModule_MissingChangelogSection(t *testing.T) {
	repo := newTestModuleCatalogRepo(t, "website-hosting", "v1.0.0")
	changelogPath := filepath.Join(repo, "modules", "website-hosting", "CHANGELOG.md")
	if err := os.WriteFile(changelogPath, []byte("# Changelog\n\nNothing here.\n"), 0o644); err != nil {
		t.Fatalf("write CHANGELOG.md: %v", err)
	}
	amendChangelog(t, repo, "remove unreleased section")

	t.Chdir(repo)

	var out, errOut bytes.Buffer
	err := ReleaseModule(ReleaseModuleOptions{Module: "website-hosting", Version: "v1.1.0", Stdout: &out, Stderr: &errOut})
	if err == nil {
		t.Error("ReleaseModule: expected an error when neither Unreleased nor a matching entry exists, got nil")
	}
}

func TestReleaseModule_UnrelatedDirtyChanges(t *testing.T) {
	repo := newTestModuleCatalogRepo(t, "website-hosting", "v1.0.0")
	other := filepath.Join(repo, "unrelated.txt")
	if err := os.WriteFile(other, []byte("v1\n"), 0o644); err != nil {
		t.Fatalf("write unrelated.txt: %v", err)
	}
	runGitT(t, repo, "add", "unrelated.txt")
	runGitT(t, repo, "commit", "--quiet", "-m", "add unrelated file")
	if err := os.WriteFile(other, []byte("v2\n"), 0o644); err != nil {
		t.Fatalf("modify unrelated.txt: %v", err)
	}
	runGitT(t, repo, "add", "unrelated.txt")

	t.Chdir(repo)

	var out, errOut bytes.Buffer
	err := ReleaseModule(ReleaseModuleOptions{Module: "website-hosting", Version: "v1.1.0", Stdout: &out, Stderr: &errOut})
	if err == nil {
		t.Error("ReleaseModule: expected an error for an unrelated staged change, got nil")
	} else if !strings.Contains(err.Error(), "unrelated.txt") {
		t.Errorf("error = %q, want it to mention unrelated.txt", err)
	}
}

func TestReleaseModule_TagAlreadyExists(t *testing.T) {
	repo := newTestModuleCatalogRepo(t, "website-hosting", "v1.0.0")
	runGitT(t, repo, "tag", "-a", "website-hosting/v1.1.0", "-m", "already tagged")

	t.Chdir(repo)

	var out, errOut bytes.Buffer
	err := ReleaseModule(ReleaseModuleOptions{Module: "website-hosting", Version: "v1.1.0", Stdout: &out, Stderr: &errOut})
	if err == nil {
		t.Error("ReleaseModule: expected an error when the tag already exists, got nil")
	}
}

func TestReleaseModule_ModuleNotFound(t *testing.T) {
	repo := newTestModuleCatalogRepo(t, "website-hosting", "v1.0.0")
	t.Chdir(repo)

	var out, errOut bytes.Buffer
	err := ReleaseModule(ReleaseModuleOptions{Module: "does-not-exist", Version: "v1.1.0", Stdout: &out, Stderr: &errOut})
	if err == nil {
		t.Error("ReleaseModule: expected an error for an unknown module, got nil")
	}
}

func TestReleaseModule_InvalidVersion(t *testing.T) {
	repo := newTestModuleCatalogRepo(t, "website-hosting", "v1.0.0")
	t.Chdir(repo)

	var out, errOut bytes.Buffer
	err := ReleaseModule(ReleaseModuleOptions{Module: "website-hosting", Version: "1.1.0", Stdout: &out, Stderr: &errOut})
	if err == nil {
		t.Error("ReleaseModule: expected an error for a version without a 'v' prefix, got nil")
	}
}

func TestReleaseModule_Push(t *testing.T) {
	repo := newTestModuleCatalogRepo(t, "website-hosting", "v1.0.0")

	remote := t.TempDir()
	runGitT(t, remote, "init", "--quiet", "--bare")
	runGitT(t, repo, "remote", "add", "origin", remote)
	// A bare remote has no branches yet: push the initial state first, so
	// ReleaseModule's own push has something to fast-forward from.
	branch := gitOutput(t, repo, "symbolic-ref", "--short", "HEAD")
	runGitT(t, repo, "push", "origin", branch)

	t.Chdir(repo)

	var out, errOut bytes.Buffer
	err := ReleaseModule(ReleaseModuleOptions{
		Module: "website-hosting", Version: "v1.1.0", Push: true,
		Stdout: &out, Stderr: &errOut,
	})
	if err != nil {
		t.Fatalf("ReleaseModule: %v (stderr: %s)", err, errOut.String())
	}

	remoteTag := gitOutput(t, remote, "tag", "-l", "website-hosting/v1.1.0")
	if remoteTag != "website-hosting/v1.1.0" {
		t.Errorf("tag not found on remote: %q", remoteTag)
	}
}

// gitOutput runs `git -C repo <args...>` and returns its trimmed stdout,
// failing the test on error.
func gitOutput(t *testing.T, repo string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// runGitT runs `git -C repo <args...>`, failing the test on error.
func runGitT(t *testing.T, repo string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", repo}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// amendChangelog stages every change and commits it, for tests that need
// to seed a specific CHANGELOG.md content on top of newTestModuleCatalogRepo.
func amendChangelog(t *testing.T, repo, message string) {
	t.Helper()
	runGitT(t, repo, "add", ".")
	runGitT(t, repo, "commit", "--quiet", "-m", message)
}
