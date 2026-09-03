package terraform

import (
	"os"
	"path/filepath"
)

// Clean removes the local Terraform cache (shared by every environment)
// and this environment's saved plan, if any.
func (r Runner) Clean() error {
	if err := os.RemoveAll(filepath.Join(r.Env.LiveDir(), ".terraform")); err != nil {
		return err
	}
	if err := removeIfExists(filepath.Join(r.Env.LiveDir(), ".terraform.lock.hcl")); err != nil {
		return err
	}
	return removeIfExists(r.Env.PlanFile())
}

// removeIfExists removes path, treating it already being gone as success.
func removeIfExists(path string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
