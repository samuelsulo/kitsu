package terraform

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// Docs regenerates each module's README input/output tables with
// terraform-docs, run against every module directory two levels under the
// infrastructure directory's modules/ (e.g. modules/local/<name>,
// modules/vendor/<name>).
func (r Runner) Docs(docsBin string) error {
	if docsBin == "" {
		docsBin = "terraform-docs"
	}
	if _, err := exec.LookPath(docsBin); err != nil {
		return fmt.Errorf("terraform-docs not found (%s): install it, see https://terraform-docs.io/user-guide/installation/", docsBin)
	}

	dirs, err := filepath.Glob(filepath.Join(r.Env.infraDir(), "modules", "*", "*"))
	if err != nil {
		return fmt.Errorf("listing module directories: %w", err)
	}

	for _, dir := range dirs {
		info, err := os.Stat(dir)
		if err != nil {
			return err
		}
		if !info.IsDir() {
			continue
		}

		cmd := exec.Command(docsBin, "markdown", "table", "--output-file", "README.md", "--output-mode", "inject", dir)
		cmd.Stdout = r.Stdout
		cmd.Stderr = r.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("terraform-docs %s: %w", dir, err)
		}
	}

	return nil
}
