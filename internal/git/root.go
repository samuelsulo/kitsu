// Package git provides small helpers around the git CLI needed by kitsu's
// commands. It shells out to the git binary rather than reimplementing
// git's logic.
package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Root returns the absolute path of the top-level directory of the git
// repository containing dir. An empty dir means the current directory.
func Root(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if dir != "" {
		cmd.Dir = dir
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("not a git repository (or any parent): %s", msg)
	}

	return strings.TrimSpace(stdout.String()), nil
}
