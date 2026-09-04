package skills

import (
	"archive/zip"
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// newTestSkillsRepo creates a local git repository with one skill
// folder under skills/, that stands in for a real skills repository:
// git clone works the same against a local path as against a remote
// URL.
func newTestSkillsRepo(t *testing.T, skill string) string {
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

	skillDir := filepath.Join(repo, "skills", skill)
	if err := os.MkdirAll(skillDir, 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", skillDir, err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte("# "+skill+"\n"), 0o644); err != nil {
		t.Fatalf("write SKILL.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(skillDir, "reference.md"), []byte("details\n"), 0o644); err != nil {
		t.Fatalf("write reference.md: %v", err)
	}

	run("add", ".")
	run("commit", "--quiet", "-m", "seed skill")

	return repo
}

// withFakeHome points $HOME at a fresh temp directory for the duration
// of the test, so Install's ~/.claude/skills/ write is isolated.
func withFakeHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	return home
}

func TestInstall_Local(t *testing.T) {
	repo := newTestSkillsRepo(t, "my-skill")
	home := withFakeHome(t)
	t.Chdir(repo)

	var out, errOut bytes.Buffer
	err := Install(InstallOptions{Skill: "my-skill", Local: true, Stdout: &out, Stderr: &errOut})
	if err != nil {
		t.Fatalf("Install: %v (stderr: %s)", err, errOut.String())
	}

	installed := filepath.Join(home, ".claude", "skills", "my-skill")
	if _, err := os.Stat(filepath.Join(installed, "SKILL.md")); err != nil {
		t.Errorf("SKILL.md not installed at %q: %v", installed, err)
	}
	if _, err := os.Stat(filepath.Join(installed, "reference.md")); err != nil {
		t.Errorf("reference.md not installed at %q: %v", installed, err)
	}
}

func TestInstall_Local_Overwrites(t *testing.T) {
	repo := newTestSkillsRepo(t, "my-skill")
	home := withFakeHome(t)

	installed := filepath.Join(home, ".claude", "skills", "my-skill")
	if err := os.MkdirAll(installed, 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", installed, err)
	}
	stale := filepath.Join(installed, "stale.md")
	if err := os.WriteFile(stale, []byte("old\n"), 0o644); err != nil {
		t.Fatalf("write stale file: %v", err)
	}

	t.Chdir(repo)
	var out, errOut bytes.Buffer
	if err := Install(InstallOptions{Skill: "my-skill", Local: true, Stdout: &out, Stderr: &errOut}); err != nil {
		t.Fatalf("Install: %v (stderr: %s)", err, errOut.String())
	}

	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale.md from a previous install still exists: err=%v", err)
	}
}

func TestInstall_Remote(t *testing.T) {
	repo := newTestSkillsRepo(t, "my-skill")
	home := withFakeHome(t)

	// Not --local: resolveSkillsRoot clones opts.Repo instead. A local
	// path is a valid git clone source too, so this exercises the real
	// clone path without touching the network.
	var out, errOut bytes.Buffer
	err := Install(InstallOptions{Skill: "my-skill", Repo: repo, Stdout: &out, Stderr: &errOut})
	if err != nil {
		t.Fatalf("Install: %v (stderr: %s)", err, errOut.String())
	}

	installed := filepath.Join(home, ".claude", "skills", "my-skill")
	if _, err := os.Stat(filepath.Join(installed, "SKILL.md")); err != nil {
		t.Errorf("SKILL.md not installed at %q: %v", installed, err)
	}
}

func TestInstall_UnknownSkill(t *testing.T) {
	repo := newTestSkillsRepo(t, "my-skill")
	withFakeHome(t)
	t.Chdir(repo)

	var out, errOut bytes.Buffer
	err := Install(InstallOptions{Skill: "does-not-exist", Local: true, Stdout: &out, Stderr: &errOut})
	if err == nil {
		t.Error("Install: expected an error for an unknown skill, got nil")
	}
}

func TestInstall_MissingSkillMD(t *testing.T) {
	repo := newTestSkillsRepo(t, "my-skill")
	if err := os.Remove(filepath.Join(repo, "skills", "my-skill", "SKILL.md")); err != nil {
		t.Fatalf("remove SKILL.md: %v", err)
	}
	withFakeHome(t)
	t.Chdir(repo)

	var out, errOut bytes.Buffer
	err := Install(InstallOptions{Skill: "my-skill", Local: true, Stdout: &out, Stderr: &errOut})
	if err == nil {
		t.Error("Install: expected an error when SKILL.md is missing, got nil")
	}
}

func TestPackage_Local(t *testing.T) {
	repo := newTestSkillsRepo(t, "my-skill")
	t.Chdir(repo)

	outDir := t.TempDir()
	var out, errOut bytes.Buffer
	zipPath, err := Package(PackageOptions{Skill: "my-skill", OutputDir: outDir, Local: true, Stdout: &out, Stderr: &errOut})
	if err != nil {
		t.Fatalf("Package: %v (stderr: %s)", err, errOut.String())
	}
	if want := filepath.Join(outDir, "my-skill.zip"); zipPath != want {
		t.Errorf("zipPath = %q, want %q", zipPath, want)
	}

	r, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer r.Close()

	names := map[string]bool{}
	for _, f := range r.File {
		names[f.Name] = true
	}
	for _, want := range []string{"my-skill/SKILL.md", "my-skill/reference.md"} {
		if !names[want] {
			t.Errorf("zip is missing %q; entries: %v", want, names)
		}
	}
}

func TestPackage_DefaultsToCurrentDirectory(t *testing.T) {
	repo := newTestSkillsRepo(t, "my-skill")
	workDir := t.TempDir()
	t.Chdir(workDir)

	var out, errOut bytes.Buffer
	zipPath, err := Package(PackageOptions{Skill: "my-skill", Local: false, Repo: repo, Stdout: &out, Stderr: &errOut})
	if err != nil {
		t.Fatalf("Package: %v (stderr: %s)", err, errOut.String())
	}
	if want := filepath.Join(workDir, "my-skill.zip"); zipPath != want {
		t.Errorf("zipPath = %q, want %q", zipPath, want)
	}
}

func TestPackage_UnknownSkill(t *testing.T) {
	repo := newTestSkillsRepo(t, "my-skill")
	t.Chdir(repo)

	var out, errOut bytes.Buffer
	_, err := Package(PackageOptions{Skill: "does-not-exist", OutputDir: t.TempDir(), Local: true, Stdout: &out, Stderr: &errOut})
	if err == nil {
		t.Error("Package: expected an error for an unknown skill, got nil")
	}
}
