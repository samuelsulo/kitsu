package terraform

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRunner_Import(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	r := newTestRunner(t, logFile)

	if err := r.Import("aws_s3_bucket.this", "my-bucket"); err != nil {
		t.Fatalf("Import: %v", err)
	}

	invocations := readInvocations(t, logFile)
	wantArgs := []string{"import", "-var-file=" + abs(r.Env.VarFile()), "aws_s3_bucket.this", "my-bucket"}
	if strings.Join(invocations[0].args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("args = %v, want %v", invocations[0].args, wantArgs)
	}
}

func TestRunner_StateList(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	r := newTestRunner(t, logFile)

	if err := r.StateList(); err != nil {
		t.Fatalf("StateList: %v", err)
	}

	invocations := readInvocations(t, logFile)
	wantArgs := []string{"state", "list"}
	if strings.Join(invocations[0].args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("args = %v, want %v", invocations[0].args, wantArgs)
	}
}

func TestRunner_StateShow(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	r := newTestRunner(t, logFile)

	if err := r.StateShow("aws_s3_bucket.this"); err != nil {
		t.Fatalf("StateShow: %v", err)
	}

	invocations := readInvocations(t, logFile)
	wantArgs := []string{"state", "show", "aws_s3_bucket.this"}
	if strings.Join(invocations[0].args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("args = %v, want %v", invocations[0].args, wantArgs)
	}
}

func TestRunner_StateRemove(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	r := newTestRunner(t, logFile)

	if err := r.StateRemove("aws_s3_bucket.this"); err != nil {
		t.Fatalf("StateRemove: %v", err)
	}

	invocations := readInvocations(t, logFile)
	wantArgs := []string{"state", "rm", "aws_s3_bucket.this"}
	if strings.Join(invocations[0].args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("args = %v, want %v", invocations[0].args, wantArgs)
	}
}

func TestRunner_Unlock(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	r := newTestRunner(t, logFile)

	if err := r.Unlock("abc-123"); err != nil {
		t.Fatalf("Unlock: %v", err)
	}

	invocations := readInvocations(t, logFile)
	wantArgs := []string{"force-unlock", "abc-123"}
	if strings.Join(invocations[0].args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("args = %v, want %v", invocations[0].args, wantArgs)
	}
}

func TestRunner_Taint(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	r := newTestRunner(t, logFile)

	if err := r.Taint("aws_s3_bucket.this"); err != nil {
		t.Fatalf("Taint: %v", err)
	}

	invocations := readInvocations(t, logFile)
	wantArgs := []string{"taint", "aws_s3_bucket.this"}
	if strings.Join(invocations[0].args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("args = %v, want %v", invocations[0].args, wantArgs)
	}
}

func TestRunner_Untaint(t *testing.T) {
	logFile := filepath.Join(t.TempDir(), "log")
	r := newTestRunner(t, logFile)

	if err := r.Untaint("aws_s3_bucket.this"); err != nil {
		t.Fatalf("Untaint: %v", err)
	}

	invocations := readInvocations(t, logFile)
	wantArgs := []string{"untaint", "aws_s3_bucket.this"}
	if strings.Join(invocations[0].args, " ") != strings.Join(wantArgs, " ") {
		t.Errorf("args = %v, want %v", invocations[0].args, wantArgs)
	}
}
