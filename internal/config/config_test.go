package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// withConfigDir points the user config directory (and so Path()) at a
// fresh temp directory for the duration of the test, and moves the
// working directory out of whatever git repository (and its
// .kitsu.yaml) the test binary happens to run inside — so tests that
// don't care about the local, per-project config file see none.
func withConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Chdir(t.TempDir())
	return dir
}

// withProjectRepo git-inits a fresh temp directory, changes into it for
// the duration of the test, and returns its path — so ProjectPath()
// resolves inside it.
func withProjectRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := exec.Command("git", "init", "--quiet", dir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	t.Chdir(dir)
	return dir
}

func TestPath(t *testing.T) {
	dir := withConfigDir(t)

	got, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	want := filepath.Join(dir, "kitsu", "config.yaml")
	if got != want {
		t.Errorf("Path() = %q, want %q", got, want)
	}
}

func TestLoad_MissingFile(t *testing.T) {
	withConfigDir(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg != (Config{}) {
		t.Errorf("Load() with no config file = %+v, want zero value", cfg)
	}
}

func TestLoad_ValidFile(t *testing.T) {
	withConfigDir(t)
	writeConfig(t, `
terraform:
  catalog_repo: "git@example.com:org/catalog.git"
  role_arn_template: "arn:aws:iam::%s:role/AdminRole"
`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got, want := cfg.Terraform.CatalogRepo, "git@example.com:org/catalog.git"; got != want {
		t.Errorf("CatalogRepo = %q, want %q", got, want)
	}
	if got, want := cfg.Terraform.RoleARNTemplate, "arn:aws:iam::%s:role/AdminRole"; got != want {
		t.Errorf("RoleARNTemplate = %q, want %q", got, want)
	}
}

func TestLoad_MalformedFile(t *testing.T) {
	withConfigDir(t)
	writeConfig(t, "not: [valid: yaml")

	if _, err := Load(); err == nil {
		t.Error("Load: expected an error for a malformed config file, got nil")
	}
}

func TestProjectPath(t *testing.T) {
	repo := withProjectRepo(t)

	got, err := ProjectPath()
	if err != nil {
		t.Fatalf("ProjectPath: %v", err)
	}
	want := filepath.Join(repo, ".kitsu.yaml")
	if got != want {
		t.Errorf("ProjectPath() = %q, want %q", got, want)
	}
}

func TestProjectPath_NotAGitRepository(t *testing.T) {
	withConfigDir(t)

	if _, err := ProjectPath(); err == nil {
		t.Error("ProjectPath: expected an error outside a git repository, got nil")
	}
}

func TestLoadProject_MissingFile(t *testing.T) {
	withProjectRepo(t)

	cfg, err := LoadProject()
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	if cfg != (Config{}) {
		t.Errorf("LoadProject() with no local config file = %+v, want zero value", cfg)
	}
}

func TestLoadProject_NotAGitRepository(t *testing.T) {
	withConfigDir(t)

	// Unlike ProjectPath, LoadProject tolerates not being inside a git
	// repository, since it's meant to be folded into LoadMerged
	// unconditionally.
	cfg, err := LoadProject()
	if err != nil {
		t.Fatalf("LoadProject outside a git repository: %v", err)
	}
	if cfg != (Config{}) {
		t.Errorf("LoadProject() outside a git repository = %+v, want zero value", cfg)
	}
}

func TestLoadProject_ValidFile(t *testing.T) {
	withProjectRepo(t)
	writeProjectConfig(t, "skills:\n  repo: \"someone/skills\"\n")

	cfg, err := LoadProject()
	if err != nil {
		t.Fatalf("LoadProject: %v", err)
	}
	if got, want := cfg.Skills.Repo, "someone/skills"; got != want {
		t.Errorf("Repo = %q, want %q", got, want)
	}
}

func TestLoadMerged_LocalOverridesGlobal(t *testing.T) {
	withConfigDir(t)
	writeConfig(t, `
terraform:
  catalog_repo: "global-catalog"
  role_arn_template: "global-role"
`)
	withProjectRepo(t)
	writeProjectConfig(t, `
terraform:
  catalog_repo: "local-catalog"
`)

	cfg, err := LoadMerged()
	if err != nil {
		t.Fatalf("LoadMerged: %v", err)
	}
	if got, want := cfg.Terraform.CatalogRepo, "local-catalog"; got != want {
		t.Errorf("CatalogRepo = %q, want %q (local should override global)", got, want)
	}
	if got, want := cfg.Terraform.RoleARNTemplate, "global-role"; got != want {
		t.Errorf("RoleARNTemplate = %q, want %q (should fall back to global)", got, want)
	}
}

func TestResolveCatalogRepo(t *testing.T) {
	withConfigDir(t)
	writeConfig(t, "terraform:\n  catalog_repo: \"from-config\"\n")

	if got, err := ResolveCatalogRepo("from-flag"); err != nil || got != "from-flag" {
		t.Errorf("ResolveCatalogRepo(explicit) = %q, %v, want %q, nil", got, err, "from-flag")
	}
	if got, err := ResolveCatalogRepo(""); err != nil || got != "from-config" {
		t.Errorf("ResolveCatalogRepo(\"\") = %q, %v, want %q, nil", got, err, "from-config")
	}
}

func TestResolveCatalogRepo_NotConfigured(t *testing.T) {
	withConfigDir(t)

	if _, err := ResolveCatalogRepo(""); err == nil {
		t.Error("ResolveCatalogRepo(\"\") with no config: expected an error, got nil")
	}
}

func TestResolveRoleARNTemplate_NotConfigured(t *testing.T) {
	withConfigDir(t)

	if _, err := ResolveRoleARNTemplate(""); err == nil {
		t.Error("ResolveRoleARNTemplate(\"\") with no config: expected an error, got nil")
	}
}

func TestResolveSkillsRepo(t *testing.T) {
	withConfigDir(t)
	writeConfig(t, "skills:\n  repo: \"someone/skills\"\n")

	if got, err := ResolveSkillsRepo("someone-else/skills"); err != nil || got != "someone-else/skills" {
		t.Errorf("ResolveSkillsRepo(explicit) = %q, %v, want %q, nil", got, err, "someone-else/skills")
	}
	if got, err := ResolveSkillsRepo(""); err != nil || got != "someone/skills" {
		t.Errorf("ResolveSkillsRepo(\"\") = %q, %v, want %q, nil", got, err, "someone/skills")
	}
}

func TestResolveSkillsRepo_FallsBackToDefault(t *testing.T) {
	withConfigDir(t)

	got, err := ResolveSkillsRepo("")
	if err != nil {
		t.Fatalf("ResolveSkillsRepo(\"\") with no config: %v", err)
	}
	if got != DefaultSkillsRepo {
		t.Errorf("ResolveSkillsRepo(\"\") = %q, want the default %q", got, DefaultSkillsRepo)
	}
}

func TestSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "config.yaml")

	want := Config{Skills: SkillsConfig{Repo: "someone/skills"}}
	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := LoadAt(path)
	if err != nil {
		t.Fatalf("LoadAt: %v", err)
	}
	if got != want {
		t.Errorf("LoadAt(Save(cfg)) = %+v, want %+v", got, want)
	}
}

func TestKeys(t *testing.T) {
	keys := Keys()
	want := []string{"skills.repo", "terraform.catalog_repo", "terraform.role_arn_template"}
	if len(keys) != len(want) {
		t.Fatalf("Keys() = %v, want %v", keys, want)
	}
	for i, k := range want {
		if keys[i] != k {
			t.Errorf("Keys()[%d] = %q, want %q", i, keys[i], k)
		}
	}
}

func TestGetSetUnset(t *testing.T) {
	var cfg Config

	if err := Set(&cfg, "terraform.catalog_repo", "example"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got, err := Get(cfg, "terraform.catalog_repo")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "example" {
		t.Errorf("Get(terraform.catalog_repo) = %q, want %q", got, "example")
	}

	if err := Unset(&cfg, "terraform.catalog_repo"); err != nil {
		t.Fatalf("Unset: %v", err)
	}
	got, err = Get(cfg, "terraform.catalog_repo")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != "" {
		t.Errorf("Get(terraform.catalog_repo) after Unset = %q, want empty", got)
	}
}

func TestGetSetUnset_UnknownKey(t *testing.T) {
	var cfg Config

	if _, err := Get(cfg, "nope"); err == nil {
		t.Error("Get with an unknown key: expected an error, got nil")
	}
	if err := Set(&cfg, "nope", "x"); err == nil {
		t.Error("Set with an unknown key: expected an error, got nil")
	}
	if err := Unset(&cfg, "nope"); err == nil {
		t.Error("Unset with an unknown key: expected an error, got nil")
	}
}

// writeConfig points Path() at a temp config dir and writes content as
// its config.yaml. Must run after withConfigDir (or withProjectRepo,
// which doesn't touch XDG_CONFIG_HOME) has set up the environment for
// the test.
func writeConfig(t *testing.T, content string) {
	t.Helper()

	path, err := Path()
	if err != nil {
		t.Fatalf("Path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}

// writeProjectConfig writes content as the current git repository's
// .kitsu.yaml. Must run after withProjectRepo.
func writeProjectConfig(t *testing.T, content string) {
	t.Helper()

	path, err := ProjectPath()
	if err != nil {
		t.Fatalf("ProjectPath: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}
