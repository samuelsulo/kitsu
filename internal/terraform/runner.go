package terraform

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// Runner executes Terraform workflow steps for one Env, streaming the
// child process's I/O through Stdout/Stderr/Stdin.
type Runner struct {
	Env Env

	Stdout io.Writer
	Stderr io.Writer
	Stdin  io.Reader
}

// exec runs the terraform binary with args, in dir, streaming I/O through
// r's Stdout/Stderr/Stdin.
func (r Runner) exec(dir string, args ...string) error {
	cmd := exec.Command(r.Env.binary(), args...)
	cmd.Dir = dir
	cmd.Stdout = r.Stdout
	cmd.Stderr = r.Stderr
	cmd.Stdin = r.Stdin
	return cmd.Run()
}

// abs resolves path to an absolute one relative to the current working
// directory. Needed because Terraform commands run with dir set to
// Env.LiveDir(), so any relative path passed as an argument (-var-file,
// -backend-config, a saved plan file) would otherwise be resolved against
// the wrong directory.
func abs(path string) string {
	resolved, err := filepath.Abs(path)
	if err != nil {
		// filepath.Abs only fails if os.Getwd fails; fall back to the
		// original (possibly still relative) path rather than losing it.
		return path
	}
	return resolved
}

// Init initializes Terraform and configures the backend for r.Env.
func (r Runner) Init() error {
	return r.exec(r.Env.LiveDir(),
		"init",
		"-backend-config="+abs(r.Env.BackendConfig()),
		"-reconfigure",
	)
}

// Validate re-initializes (see Init) and validates the configuration.
func (r Runner) Validate() error {
	if err := r.Init(); err != nil {
		return err
	}
	return r.exec(r.Env.LiveDir(), "validate")
}

// Plan validates (see Validate) and generates and saves an execution plan
// to Env.PlanFile.
func (r Runner) Plan() error {
	if err := r.Validate(); err != nil {
		return err
	}
	return r.exec(r.Env.LiveDir(),
		"plan",
		"-var-file="+abs(r.Env.VarFile()),
		"-out="+abs(r.Env.PlanFile()),
	)
}

// ShowPlan shows the plan previously saved by Plan.
func (r Runner) ShowPlan() error {
	planFile := r.Env.PlanFile()
	if _, err := os.Stat(planFile); err != nil {
		return fmt.Errorf("no saved plan found at %s: run 'kitsu terraform plan' first", planFile)
	}
	return r.exec(r.Env.LiveDir(), "show", abs(planFile))
}

// Apply applies the plan previously saved by Plan, then removes it.
func (r Runner) Apply() error {
	planFile := r.Env.PlanFile()
	if _, err := os.Stat(planFile); err != nil {
		return fmt.Errorf("no saved plan found at %s: run 'kitsu terraform plan' first", planFile)
	}
	if err := r.exec(r.Env.LiveDir(), "apply", abs(planFile)); err != nil {
		return err
	}
	return os.Remove(planFile)
}

// ApplyTarget validates (see Validate) and applies changes to a single
// resource target, without requiring a saved plan.
func (r Runner) ApplyTarget(target string) error {
	if err := r.Validate(); err != nil {
		return err
	}
	return r.exec(r.Env.LiveDir(),
		"apply",
		"-var-file="+abs(r.Env.VarFile()),
		"-target="+target,
		"-auto-approve",
	)
}

// ApplyAuto validates (see Validate), then plans and applies in one step,
// without a saved plan or a confirmation prompt. Use with caution.
func (r Runner) ApplyAuto() error {
	if err := r.Validate(); err != nil {
		return err
	}
	return r.exec(r.Env.LiveDir(),
		"apply",
		"-var-file="+abs(r.Env.VarFile()),
		"-auto-approve",
	)
}

// PlanDestroy validates (see Validate) and previews what a destroy would
// remove, without applying anything.
func (r Runner) PlanDestroy() error {
	if err := r.Validate(); err != nil {
		return err
	}
	return r.exec(r.Env.LiveDir(),
		"plan",
		"-destroy",
		"-var-file="+abs(r.Env.VarFile()),
	)
}

// Refresh validates (see Validate) and reconciles the state with the real
// infrastructure, without changing either.
func (r Runner) Refresh() error {
	if err := r.Validate(); err != nil {
		return err
	}
	return r.exec(r.Env.LiveDir(),
		"plan",
		"-refresh-only",
		"-var-file="+abs(r.Env.VarFile()),
	)
}

// Destroy destroys every resource in r.Env. Callers are responsible for
// confirming this with the user first (see cli.confirm).
func (r Runner) Destroy() error {
	return r.exec(r.Env.LiveDir(), "destroy", "-var-file="+abs(r.Env.VarFile()))
}

// DestroyTarget destroys a single resource target in r.Env. Callers are
// responsible for confirming this with the user first (see cli.confirm).
func (r Runner) DestroyTarget(target string) error {
	return r.exec(r.Env.LiveDir(),
		"destroy",
		"-var-file="+abs(r.Env.VarFile()),
		"-target="+target,
	)
}
