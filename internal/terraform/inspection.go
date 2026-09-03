package terraform

// Console opens an interactive console to evaluate expressions against
// the current state.
func (r Runner) Console() error {
	return r.exec(r.Env.LiveDir(), "console", "-var-file="+abs(r.Env.VarFile()))
}

// Providers shows the provider requirements and versions in use.
func (r Runner) Providers() error {
	return r.exec(r.Env.LiveDir(), "providers")
}

// TerraformVersion shows the Terraform and provider versions in use.
// Named to avoid colliding with kitsu's own "version" concept; it's
// exposed on the CLI as `kitsu terraform version`.
func (r Runner) TerraformVersion() error {
	return r.exec(r.Env.LiveDir(), "version")
}

// Upgrade re-initializes and upgrades provider/module versions to the
// latest allowed.
func (r Runner) Upgrade() error {
	return r.exec(r.Env.LiveDir(),
		"init",
		"-backend-config="+abs(r.Env.BackendConfig()),
		"-upgrade",
	)
}
