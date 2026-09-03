package terraform

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// invocations parses the log file written by the stub terraform binary
// (see newStubTerraform) into one entry per invocation, each holding the
// working directory and arguments it was called with.
type invocation struct {
	dir  string
	args []string
}

// newStubTerraform writes a fake terraform binary that appends its
// working directory and arguments to logFile on every invocation, instead
// of doing anything real. It returns the stub's path.
func newStubTerraform(t *testing.T, logFile string) string {
	t.Helper()

	stubPath := filepath.Join(t.TempDir(), "terraform")
	script := "#!/bin/sh\n" +
		"{ echo \"DIR=$PWD\"; echo \"ARGS=$*\"; echo '---'; } >> \"" + logFile + "\"\n"
	if err := os.WriteFile(stubPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write stub terraform binary: %v", err)
	}
	return stubPath
}

func readInvocations(t *testing.T, logFile string) []invocation {
	t.Helper()

	data, err := os.ReadFile(logFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read log file: %v", err)
	}

	var invocations []invocation
	for _, block := range strings.Split(strings.TrimSpace(string(data)), "---") {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		var inv invocation
		for _, line := range strings.Split(block, "\n") {
			switch {
			case strings.HasPrefix(line, "DIR="):
				inv.dir = strings.TrimPrefix(line, "DIR=")
			case strings.HasPrefix(line, "ARGS="):
				args := strings.TrimPrefix(line, "ARGS=")
				if args != "" {
					inv.args = strings.Split(args, " ")
				}
			}
		}
		invocations = append(invocations, inv)
	}
	return invocations
}

// newTestRunner sets up an Env rooted at a temp directory, with the
// environment.hcl/environment.tfvars files Runner's methods expect to
// find already in place, and a stub terraform binary logging to logFile.
func newTestRunner(t *testing.T, logFile string) Runner {
	t.Helper()

	root := t.TempDir()
	env := Env{
		Bin:      newStubTerraform(t, logFile),
		InfraDir: filepath.Join(root, "infrastructure"),
		Name:     "sandbox",
	}

	if err := os.MkdirAll(env.LiveDir(), 0o755); err != nil {
		t.Fatalf("mkdir live dir: %v", err)
	}
	if err := os.MkdirAll(env.Dir(), 0o755); err != nil {
		t.Fatalf("mkdir env dir: %v", err)
	}
	if err := os.WriteFile(env.BackendConfig(), []byte("bucket = \"test\"\n"), 0o644); err != nil {
		t.Fatalf("write backend config: %v", err)
	}
	if err := os.WriteFile(env.VarFile(), []byte("environment = \"sandbox\"\n"), 0o644); err != nil {
		t.Fatalf("write var file: %v", err)
	}

	return Runner{Env: env}
}

func TestRunner_Init(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	r := newTestRunner(t, logFile)

	if err := r.Init(); err != nil {
		t.Fatalf("Init: %v", err)
	}

	invocations := readInvocations(t, logFile)
	if len(invocations) != 1 {
		t.Fatalf("got %d invocations, want 1: %+v", len(invocations), invocations)
	}

	inv := invocations[0]
	if want := abs(r.Env.LiveDir()); inv.dir != want {
		t.Errorf("dir = %q, want %q", inv.dir, want)
	}
	wantArgs := []string{"init", "-backend-config=" + abs(r.Env.BackendConfig()), "-reconfigure"}
	if strings.Join(inv.args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("args = %v, want %v", inv.args, wantArgs)
	}
}

func TestRunner_Validate_RunsInitFirst(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	r := newTestRunner(t, logFile)

	if err := r.Validate(); err != nil {
		t.Fatalf("Validate: %v", err)
	}

	invocations := readInvocations(t, logFile)
	if len(invocations) != 2 {
		t.Fatalf("got %d invocations, want 2 (init, validate): %+v", len(invocations), invocations)
	}
	if invocations[0].args[0] != "init" {
		t.Errorf("first invocation = %v, want it to start with 'init'", invocations[0].args)
	}
	if invocations[1].args[0] != "validate" {
		t.Errorf("second invocation = %v, want it to start with 'validate'", invocations[1].args)
	}
}

func TestRunner_Plan(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	r := newTestRunner(t, logFile)

	if err := r.Plan(); err != nil {
		t.Fatalf("Plan: %v", err)
	}

	invocations := readInvocations(t, logFile)
	if len(invocations) != 3 {
		t.Fatalf("got %d invocations, want 3 (init, validate, plan): %+v", len(invocations), invocations)
	}

	last := invocations[2]
	wantArgs := []string{"plan", "-var-file=" + abs(r.Env.VarFile()), "-out=" + abs(r.Env.PlanFile())}
	if strings.Join(last.args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("args = %v, want %v", last.args, wantArgs)
	}
}

func TestRunner_ApplyTarget(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	r := newTestRunner(t, logFile)

	if err := r.ApplyTarget("aws_s3_bucket.this"); err != nil {
		t.Fatalf("ApplyTarget: %v", err)
	}

	invocations := readInvocations(t, logFile)
	last := invocations[len(invocations)-1]
	wantArgs := []string{"apply", "-var-file=" + abs(r.Env.VarFile()), "-target=aws_s3_bucket.this", "-auto-approve"}
	if strings.Join(last.args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("args = %v, want %v", last.args, wantArgs)
	}
}

func TestRunner_Destroy(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	r := newTestRunner(t, logFile)

	if err := r.Destroy(); err != nil {
		t.Fatalf("Destroy: %v", err)
	}

	invocations := readInvocations(t, logFile)
	if len(invocations) != 1 {
		t.Fatalf("got %d invocations, want 1 (no Validate/Init dependency): %+v", len(invocations), invocations)
	}
	wantArgs := []string{"destroy", "-var-file=" + abs(r.Env.VarFile())}
	if strings.Join(invocations[0].args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("args = %v, want %v", invocations[0].args, wantArgs)
	}
}

func TestRunner_ShowPlan_NoSavedPlan(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	r := newTestRunner(t, logFile)

	if err := r.ShowPlan(); err == nil {
		t.Error("ShowPlan: expected an error when no plan was saved, got nil")
	}
	if invocations := readInvocations(t, logFile); len(invocations) != 0 {
		t.Errorf("expected terraform not to be invoked, got %d invocations", len(invocations))
	}
}

func TestRunner_Apply_NoSavedPlan(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	r := newTestRunner(t, logFile)

	if err := r.Apply(); err == nil {
		t.Error("Apply: expected an error when no plan was saved, got nil")
	}
}

func TestRunner_Apply_RemovesPlanFileOnSuccess(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	r := newTestRunner(t, logFile)

	if err := os.WriteFile(r.Env.PlanFile(), []byte("fake plan"), 0o644); err != nil {
		t.Fatalf("write fake plan file: %v", err)
	}

	if err := r.Apply(); err != nil {
		t.Fatalf("Apply: %v", err)
	}

	if _, err := os.Stat(r.Env.PlanFile()); !os.IsNotExist(err) {
		t.Errorf("plan file still exists after Apply: err=%v", err)
	}
}
