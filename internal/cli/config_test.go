package cli

import (
	"bytes"
	"os/exec"
	"strings"
	"testing"
)

// withConfigDir points the user config directory at a fresh temp
// directory and moves the working directory out of whatever git
// repository the test binary happens to run inside, for the duration
// of the test — mirroring internal/config's own test helper, since
// these tests exercise the same environment (XDG_CONFIG_HOME, cwd) but
// can't import internal/config's unexported helper.
func withConfigDir(t *testing.T) {
	t.Helper()
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Chdir(t.TempDir())
}

// withProjectRepo git-inits a fresh temp directory and changes into it
// for the duration of the test.
func withProjectRepo(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	if err := exec.Command("git", "init", "--quiet", dir).Run(); err != nil {
		t.Fatalf("git init: %v", err)
	}
	t.Chdir(dir)
}

// runConfig runs "kitsu config <args...>" and returns its stdout,
// failing the test on error.
func runConfig(t *testing.T, args ...string) string {
	t.Helper()
	cmd := newConfigCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetArgs(args)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("kitsu config %s: %v", strings.Join(args, " "), err)
	}
	return out.String()
}

// runConfigErr runs "kitsu config <args...>" expecting it to fail, and
// returns the error.
func runConfigErr(t *testing.T, args ...string) error {
	t.Helper()
	cmd := newConfigCmd()
	cmd.SetOut(bytes.NewBuffer(nil))
	cmd.SetArgs(args)
	err := cmd.Execute()
	if err == nil {
		t.Fatalf("kitsu config %s: expected an error, got nil", strings.Join(args, " "))
	}
	return err
}

func TestConfigSetGet_Global(t *testing.T) {
	withConfigDir(t)

	runConfig(t, "set", "--global", "skills.repo", "someone/skills")

	got := strings.TrimSpace(runConfig(t, "get", "skills.repo"))
	if got != "someone/skills" {
		t.Errorf("get skills.repo = %q, want %q", got, "someone/skills")
	}
}

func TestConfigSetGet_LocalOverridesGlobal(t *testing.T) {
	withConfigDir(t)
	runConfig(t, "set", "--global", "terraform.catalog_repo", "global-catalog")

	withProjectRepo(t)
	runConfig(t, "set", "--local", "terraform.catalog_repo", "local-catalog")

	got := strings.TrimSpace(runConfig(t, "get", "terraform.catalog_repo"))
	if got != "local-catalog" {
		t.Errorf("get terraform.catalog_repo (merged) = %q, want %q", got, "local-catalog")
	}

	got = strings.TrimSpace(runConfig(t, "get", "--global", "terraform.catalog_repo"))
	if got != "global-catalog" {
		t.Errorf("get --global terraform.catalog_repo = %q, want %q", got, "global-catalog")
	}
}

func TestConfigUnset(t *testing.T) {
	withConfigDir(t)
	runConfig(t, "set", "--global", "skills.repo", "someone/skills")

	runConfig(t, "unset", "--global", "skills.repo")

	got := strings.TrimSpace(runConfig(t, "get", "skills.repo"))
	if got != "" {
		t.Errorf("get skills.repo after unset = %q, want empty", got)
	}
}

func TestConfigShow(t *testing.T) {
	withConfigDir(t)
	runConfig(t, "set", "--global", "skills.repo", "someone/skills")

	got := runConfig(t, "show", "--global")
	if !strings.Contains(got, "repo: someone/skills") {
		t.Errorf("show --global = %q, want it to contain %q", got, "repo: someone/skills")
	}
}

func TestConfigPath(t *testing.T) {
	withConfigDir(t)

	got := strings.TrimSpace(runConfig(t, "path", "--global"))
	if !strings.HasSuffix(got, "kitsu/config.yaml") && !strings.HasSuffix(got, "kitsu\\config.yaml") {
		t.Errorf("path --global = %q, want it to end with kitsu/config.yaml", got)
	}
}

func TestConfigPath_LocalOutsideGitRepository(t *testing.T) {
	withConfigDir(t)

	runConfigErr(t, "path", "--local")
}

func TestConfigSet_RequiresScope(t *testing.T) {
	withConfigDir(t)

	runConfigErr(t, "set", "skills.repo", "someone/skills")
}

func TestConfigSet_GlobalAndLocalMutuallyExclusive(t *testing.T) {
	withConfigDir(t)

	runConfigErr(t, "set", "--global", "--local", "skills.repo", "someone/skills")
}

func TestConfigGet_UnknownKey(t *testing.T) {
	withConfigDir(t)

	runConfigErr(t, "get", "nope.nope")
}
