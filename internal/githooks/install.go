// Package githooks installs the git hooks tracked in a repository (under a
// hooks directory such as .githooks/) by pointing git's core.hooksPath at
// it. Git never tracks .git/hooks/ itself, so hooks meant to be shared with
// a repository must live elsewhere in the tree and be wired up explicitly
// after cloning.
package githooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// executableBits are OR'd into each hook file's existing permissions,
// mirroring `chmod +x`: git does not reliably preserve the executable bit
// across clones/checkouts on every platform, so hooks must be re-marked
// executable after installing them.
const executableBits = 0o111

// Install points repoRoot's core.hooksPath at hooksDir (given relative to
// repoRoot, e.g. ".githooks") and makes every file directly inside it
// executable.
func Install(repoRoot, hooksDir string) error {
	absHooksDir := filepath.Join(repoRoot, hooksDir)

	entries, err := os.ReadDir(absHooksDir)
	if err != nil {
		return fmt.Errorf("reading hooks directory %q: %w", absHooksDir, err)
	}

	cmd := exec.Command("git", "config", "core.hooksPath", hooksDir)
	cmd.Dir = repoRoot
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git config core.hooksPath %q: %w (%s)", hooksDir, err, out)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat %q: %w", entry.Name(), err)
		}

		path := filepath.Join(absHooksDir, entry.Name())
		if err := os.Chmod(path, info.Mode()|executableBits); err != nil {
			return fmt.Errorf("chmod +x %q: %w", path, err)
		}
	}

	return nil
}
