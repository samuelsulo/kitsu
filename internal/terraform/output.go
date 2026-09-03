package terraform

// Output shows the outputs of the environment's current state.
func (r Runner) Output() error {
	return r.exec(r.Env.LiveDir(), "output")
}

// OutputJSON shows the outputs of the environment's current state, in
// JSON format.
func (r Runner) OutputJSON() error {
	return r.exec(r.Env.LiveDir(), "output", "-json")
}
