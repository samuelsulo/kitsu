package skills

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// frontmatter is the subset of a SKILL.md's YAML frontmatter this
// package cares about.
type frontmatter struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

// parseFrontmatter extracts and parses the YAML frontmatter — the block
// between the file's first line (a bare "---") and the next line that's
// also a bare "---" — out of a SKILL.md's content.
func parseFrontmatter(content []byte) (frontmatter, error) {
	lines := strings.Split(string(content), "\n")

	if len(lines) == 0 || strings.TrimRight(lines[0], "\r") != "---" {
		return frontmatter{}, fmt.Errorf(`missing frontmatter: expected the file to start with a "---" line`)
	}

	end := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], "\r") == "---" {
			end = i
			break
		}
	}
	if end == -1 {
		return frontmatter{}, fmt.Errorf(`frontmatter not closed: expected a second "---" line`)
	}

	var fm frontmatter
	if err := yaml.Unmarshal([]byte(strings.Join(lines[1:end], "\n")), &fm); err != nil {
		return frontmatter{}, fmt.Errorf("parsing frontmatter: %w", err)
	}
	return fm, nil
}
