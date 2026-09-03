package terraform

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunner_Clean(t *testing.T) {
	r := newTestRunner(t, filepath.Join(t.TempDir(), "log"))

	terraformCache := filepath.Join(r.Env.LiveDir(), ".terraform")
	lockFile := filepath.Join(r.Env.LiveDir(), ".terraform.lock.hcl")
	planFile := r.Env.PlanFile()

	if err := os.MkdirAll(filepath.Join(terraformCache, "providers"), 0o755); err != nil {
		t.Fatalf("mkdir %q: %v", terraformCache, err)
	}
	if err := os.WriteFile(lockFile, []byte("# lock\n"), 0o644); err != nil {
		t.Fatalf("write %q: %v", lockFile, err)
	}
	if err := os.WriteFile(planFile, []byte("fake plan"), 0o644); err != nil {
		t.Fatalf("write %q: %v", planFile, err)
	}

	if err := r.Clean(); err != nil {
		t.Fatalf("Clean: %v", err)
	}

	for _, path := range []string{terraformCache, lockFile, planFile} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("%q still exists after Clean: err=%v", path, err)
		}
	}
}

func TestRunner_Clean_NothingToRemove(t *testing.T) {
	r := newTestRunner(t, filepath.Join(t.TempDir(), "log"))

	// None of .terraform/, .terraform.lock.hcl or tfplan exist yet: Clean
	// must be a no-op, not an error.
	if err := r.Clean(); err != nil {
		t.Fatalf("Clean on an already-clean environment: %v", err)
	}
}
