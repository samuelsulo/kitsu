package terraform

// Import imports an existing resource, identified by its provider-specific
// id, into the Terraform state as address.
func (r Runner) Import(address, id string) error {
	return r.exec(r.Env.LiveDir(), "import", "-var-file="+abs(r.Env.VarFile()), address, id)
}

// StateList lists every resource in the Terraform state.
func (r Runner) StateList() error {
	return r.exec(r.Env.LiveDir(), "state", "list")
}

// StateShow shows the state attributes of a single resource.
func (r Runner) StateShow(address string) error {
	return r.exec(r.Env.LiveDir(), "state", "show", address)
}

// StateRemove removes a resource from the Terraform state without
// destroying it. Callers are responsible for confirming this with the
// user first (see cli.confirm).
func (r Runner) StateRemove(address string) error {
	return r.exec(r.Env.LiveDir(), "state", "rm", address)
}

// Unlock force-releases a stuck state lock. Callers are responsible for
// confirming this with the user first (see cli.confirm).
func (r Runner) Unlock(lockID string) error {
	return r.exec(r.Env.LiveDir(), "force-unlock", lockID)
}

// Taint marks a resource for recreation on the next apply.
func (r Runner) Taint(address string) error {
	return r.exec(r.Env.LiveDir(), "taint", address)
}

// Untaint undoes a previous Taint.
func (r Runner) Untaint(address string) error {
	return r.exec(r.Env.LiveDir(), "untaint", address)
}
