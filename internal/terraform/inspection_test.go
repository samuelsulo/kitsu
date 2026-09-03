package terraform

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRunner_Console(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	r := newTestRunner(t, logFile)

	if err := r.Console(); err != nil {
		t.Fatalf("Console: %v", err)
	}

	invocations := readInvocations(t, logFile)
	wantArgs := []string{"console", "-var-file=" + abs(r.Env.VarFile())}
	if strings.Join(invocations[0].args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("args = %v, want %v", invocations[0].args, wantArgs)
	}
}

func TestRunner_Providers(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	r := newTestRunner(t, logFile)

	if err := r.Providers(); err != nil {
		t.Fatalf("Providers: %v", err)
	}

	invocations := readInvocations(t, logFile)
	wantArgs := []string{"providers"}
	if strings.Join(invocations[0].args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("args = %v, want %v", invocations[0].args, wantArgs)
	}
}

func TestRunner_TerraformVersion(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	r := newTestRunner(t, logFile)

	if err := r.TerraformVersion(); err != nil {
		t.Fatalf("TerraformVersion: %v", err)
	}

	invocations := readInvocations(t, logFile)
	wantArgs := []string{"version"}
	if strings.Join(invocations[0].args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("args = %v, want %v", invocations[0].args, wantArgs)
	}
}

func TestRunner_Upgrade(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	r := newTestRunner(t, logFile)

	if err := r.Upgrade(); err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	invocations := readInvocations(t, logFile)
	wantArgs := []string{"init", "-backend-config=" + abs(r.Env.BackendConfig()), "-upgrade"}
	if strings.Join(invocations[0].args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("args = %v, want %v", invocations[0].args, wantArgs)
	}
}
