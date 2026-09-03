package terraform

import (
	"bytes"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Fmt formats every .tf/.tfvars/.tftest.hcl file (recursively, handled by
// Terraform's own `fmt` command) and every generic .hcl file (e.g.
// backend.hcl, which it doesn't) under the infrastructure directory, in
// place. Anything under a .terraform/ cache directory is left untouched.
func (r Runner) Fmt() error {
	if err := r.exec(r.Env.infraDir(), "fmt", "-recursive", "."); err != nil {
		return err
	}
	return r.walkHCLFiles(func(path string) error {
		return r.fmtHCLFile(path, false)
	})
}

// FmtCheck reports whether every .tf/.tfvars/.tftest.hcl/.hcl file under
// the infrastructure directory is already formatted, without modifying
// anything. Intended for CI.
func (r Runner) FmtCheck() error {
	if err := r.exec(r.Env.infraDir(), "fmt", "-recursive", "-check", "."); err != nil {
		return err
	}
	return r.walkHCLFiles(func(path string) error {
		return r.fmtHCLFile(path, true)
	})
}

// walkHCLFiles calls fn for every generic .hcl file under the
// infrastructure directory, in the same order as filepath.WalkDir,
// skipping .tftest.hcl files and .terraform/ cache directories.
func (r Runner) walkHCLFiles(fn func(path string) error) error {
	return filepath.WalkDir(r.Env.infraDir(), func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".terraform" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".hcl") || strings.HasSuffix(path, ".tftest.hcl") {
			return nil
		}
		return fn(path)
	})
}

// fmtHCLFile formats one generic HCL file by piping it through
// `terraform fmt -` (the sub-syntax .tf/.tfvars formatting doesn't cover
// on its own, e.g. backend.hcl). With checkOnly, it reports a formatting
// mismatch as an error instead of rewriting the file.
func (r Runner) fmtHCLFile(path string, checkOnly bool) error {
	original, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var out bytes.Buffer
	formatter := Runner{Env: r.Env, Stdout: &out, Stderr: r.Stderr, Stdin: bytes.NewReader(original)}
	if err := formatter.exec("", "fmt", "-"); err != nil {
		return fmt.Errorf("terraform fmt %s: %w", path, err)
	}

	if bytes.Equal(out.Bytes(), original) {
		return nil
	}

	if checkOnly {
		return fmt.Errorf("%s is not formatted (run 'kitsu terraform fmt')", path)
	}

	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	return os.WriteFile(path, out.Bytes(), info.Mode())
}
