package skills

import "testing"

func TestParseFrontmatter(t *testing.T) {
	content := []byte(`---
name: my-skill
description: Does a thing.
metadata:
  type: reference
---

Body content, not part of the frontmatter.
`)

	fm, err := parseFrontmatter(content)
	if err != nil {
		t.Fatalf("parseFrontmatter: %v", err)
	}
	if got, want := fm.Name, "my-skill"; got != want {
		t.Errorf("Name = %q, want %q", got, want)
	}
	if got, want := fm.Description, "Does a thing."; got != want {
		t.Errorf("Description = %q, want %q", got, want)
	}
}

func TestParseFrontmatter_MissingOpeningDelimiter(t *testing.T) {
	if _, err := parseFrontmatter([]byte("name: my-skill\n")); err == nil {
		t.Error("parseFrontmatter: expected an error for a missing opening '---', got nil")
	}
}

func TestParseFrontmatter_UnclosedDelimiter(t *testing.T) {
	if _, err := parseFrontmatter([]byte("---\nname: my-skill\n")); err == nil {
		t.Error("parseFrontmatter: expected an error for an unclosed frontmatter block, got nil")
	}
}

func TestParseFrontmatter_MalformedYAML(t *testing.T) {
	if _, err := parseFrontmatter([]byte("---\nname: [broken\n---\n")); err == nil {
		t.Error("parseFrontmatter: expected an error for malformed YAML, got nil")
	}
}
