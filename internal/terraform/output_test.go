package terraform

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRunner_Output(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	r := newTestRunner(t, logFile)

	if err := r.Output(); err != nil {
		t.Fatalf("Output: %v", err)
	}

	invocations := readInvocations(t, logFile)
	wantArgs := []string{"output"}
	if strings.Join(invocations[0].args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("args = %v, want %v", invocations[0].args, wantArgs)
	}
}

func TestRunner_OutputJSON(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	r := newTestRunner(t, logFile)

	if err := r.OutputJSON(); err != nil {
		t.Fatalf("OutputJSON: %v", err)
	}

	invocations := readInvocations(t, logFile)
	wantArgs := []string{"output", "-json"}
	if strings.Join(invocations[0].args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("args = %v, want %v", invocations[0].args, wantArgs)
	}
}
