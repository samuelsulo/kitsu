package skills

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newTestSkillsRepoWithFrontmatter is like newTestSkillsRepo, but each
// skill's SKILL.md carries real frontmatter (name + description), for
// List/Show tests.
func newTestSkillsRepoWithFrontmatter(t *testing.T, skills map[string]string) string {
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

	for name, description := range skills {
		skillDir := filepath.Join(repo, "skills", name)
		if err := os.MkdirAll(skillDir, 0o755); err != nil {
			t.Fatalf("mkdir %q: %v", skillDir, err)
		}
		content := fmt.Sprintf("---\nname: %s\ndescription: %s\n---\n\nBody.\n", name, description)
		if err := os.WriteFile(filepath.Join(skillDir, "SKILL.md"), []byte(content), 0o644); err != nil {
			t.Fatalf("write SKILL.md: %v", err)
		}
	}

	run("add", ".")
	run("commit", "--quiet", "-m", "seed skills")

	return repo
}

func TestList(t *testing.T) {
	repo := newTestSkillsRepoWithFrontmatter(t, map[string]string{
		"skill-b": "The second skill.",
		"skill-a": "The first skill.",
	})
	// A folder without a SKILL.md must not be listed as a skill.
	if err := os.MkdirAll(filepath.Join(repo, "skills", "not-a-skill"), 0o755); err != nil {
		t.Fatalf("mkdir not-a-skill: %v", err)
	}

	t.Chdir(repo)

	var out, errOut bytes.Buffer
	if err := List(ListOptions{Local: true, Stdout: &out, Stderr: &errOut}); err != nil {
		t.Fatalf("List: %v (stderr: %s)", err, errOut.String())
	}

	got := strings.TrimSpace(out.String())
	want := "skill-a\nskill-b"
	if got != want {
		t.Errorf("List output = %q, want %q (sorted, no not-a-skill)", got, want)
	}
}

func TestShow(t *testing.T) {
	repo := newTestSkillsRepoWithFrontmatter(t, map[string]string{
		"my-skill": "Does a thing.",
	})
	t.Chdir(repo)

	var out, errOut bytes.Buffer
	if err := Show(ShowOptions{Skill: "my-skill", Local: true, Stdout: &out, Stderr: &errOut}); err != nil {
		t.Fatalf("Show: %v (stderr: %s)", err, errOut.String())
	}

	want := "name: my-skill\ndescription: Does a thing.\n"
	if out.String() != want {
		t.Errorf("Show output = %q, want %q", out.String(), want)
	}
}

func TestShow_NameMismatch(t *testing.T) {
	repo := newTestSkillsRepoWithFrontmatter(t, map[string]string{
		"my-skill": "Does a thing.",
	})
	// Rename the folder without updating the frontmatter's name.
	oldDir := filepath.Join(repo, "skills", "my-skill")
	newDir := filepath.Join(repo, "skills", "renamed-skill")
	if err := os.Rename(oldDir, newDir); err != nil {
		t.Fatalf("rename skill dir: %v", err)
	}
	t.Chdir(repo)

	var out, errOut bytes.Buffer
	err := Show(ShowOptions{Skill: "renamed-skill", Local: true, Stdout: &out, Stderr: &errOut})
	if err == nil {
		t.Error("Show: expected an error when frontmatter name doesn't match the folder name, got nil")
	}
}

func TestShow_UnknownSkill(t *testing.T) {
	repo := newTestSkillsRepoWithFrontmatter(t, map[string]string{
		"my-skill": "Does a thing.",
	})
	t.Chdir(repo)

	var out, errOut bytes.Buffer
	err := Show(ShowOptions{Skill: "does-not-exist", Local: true, Stdout: &out, Stderr: &errOut})
	if err == nil {
		t.Error("Show: expected an error for an unknown skill, got nil")
	}
}
