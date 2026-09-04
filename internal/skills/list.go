package skills

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

// ListOptions configures List.
type ListOptions struct {
	// Local and Repo are the same as InstallOptions.
	Local bool
	Repo  string

	Stdout, Stderr io.Writer
}

// List prints the name of every skill folder under skills/ — one per
// line, sorted — without reading each one's SKILL.md (see Show for
// that). A folder counts as a skill if it contains a SKILL.md.
func List(opts ListOptions) error {
	root, cleanup, err := resolveSkillsRoot(opts.Local, opts.Repo)
	if err != nil {
		return err
	}
	defer cleanup()

	skillsDir := filepath.Join(root, "skills")
	entries, err := os.ReadDir(skillsDir)
	if err != nil {
		return fmt.Errorf("reading %s: %w", skillsDir, err)
	}

	var names []string
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if _, err := os.Stat(filepath.Join(skillsDir, e.Name(), "SKILL.md")); err != nil {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)

	for _, name := range names {
		fmt.Fprintln(opts.Stdout, name)
	}
	return nil
}
