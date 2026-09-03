package terraform

import (
	"path/filepath"
	"testing"
)

func TestEnv_Paths(t *testing.T) {
	e := Env{InfraDir: "infrastructure", Name: "sandbox"}

	cases := map[string]struct {
		got  string
		want string
	}{
		"LiveDir":       {e.LiveDir(), filepath.Join("infrastructure", "live")},
		"Dir":           {e.Dir(), filepath.Join("infrastructure", "environments", "sandbox")},
		"BackendConfig": {e.BackendConfig(), filepath.Join("infrastructure", "environments", "sandbox", "backend.hcl")},
		"VarFile":       {e.VarFile(), filepath.Join("infrastructure", "environments", "sandbox", "environment.tfvars")},
		"PlanFile":      {e.PlanFile(), filepath.Join("infrastructure", "environments", "sandbox", "tfplan")},
	}

	for name, c := range cases {
		if c.got != c.want {
			t.Errorf("%s = %q, want %q", name, c.got, c.want)
		}
	}
}

func TestEnv_Defaults(t *testing.T) {
	e := Env{Name: "sandbox"}

	if got, want := e.binary(), "terraform"; got != want {
		t.Errorf("binary() = %q, want %q", got, want)
	}
	if got, want := e.infraDir(), "infrastructure"; got != want {
		t.Errorf("infraDir() = %q, want %q", got, want)
	}
	if got, want := e.LiveDir(), filepath.Join("infrastructure", "live"); got != want {
		t.Errorf("LiveDir() = %q, want %q", got, want)
	}
}
