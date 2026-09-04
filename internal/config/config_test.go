package config

import (
	"os"
	"path/filepath"
	"testing"
)

// withConfigDir points the user config directory (and so Path()) at a
// fresh temp directory for the duration of the test.
func withConfigDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
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
	writeConfig(t, "not: [valid: yaml")

	if _, err := Load(); err == nil {
		t.Error("Load: expected an error for a malformed config file, got nil")
	}
}

func TestResolveCatalogRepo(t *testing.T) {
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

// writeConfig points Path() at a temp config dir and writes content as
// its config.yaml.
func writeConfig(t *testing.T, content string) {
	t.Helper()
	dir := withConfigDir(t)

	path := filepath.Join(dir, "kitsu", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %q: %v", path, err)
	}
}
