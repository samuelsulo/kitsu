package website

import "testing"

func TestValidateDeployArgs(t *testing.T) {
	cases := []struct {
		name        string
		env, tag    string
		force       bool
		wantVersion string
		wantErr     bool
	}{
		{name: "sandbox, no tag/force", env: "sandbox", wantVersion: "", wantErr: false},
		{name: "production with valid tag", env: "production", tag: "website/v1.2.3", wantVersion: "1.2.3", wantErr: false},
		{name: "production with valid tag and force", env: "production", tag: "website/v1.2.3", force: true, wantVersion: "1.2.3", wantErr: false},
		{name: "unknown env", env: "staging", wantErr: true},
		{name: "production without tag", env: "production", wantErr: true},
		{name: "production with malformed tag", env: "production", tag: "v1.2.3", wantErr: true},
		{name: "production with malformed tag (missing patch)", env: "production", tag: "website/v1.2", wantErr: true},
		{name: "sandbox with tag", env: "sandbox", tag: "website/v1.2.3", wantErr: true},
		{name: "sandbox with force", env: "sandbox", force: true, wantErr: true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			version, err := validateDeployArgs(c.env, c.tag, c.force)
			if c.wantErr {
				if err == nil {
					t.Errorf("validateDeployArgs(%q, %q, %v): expected an error, got nil", c.env, c.tag, c.force)
				}
				return
			}
			if err != nil {
				t.Fatalf("validateDeployArgs(%q, %q, %v): %v", c.env, c.tag, c.force, err)
			}
			if version != c.wantVersion {
				t.Errorf("version = %q, want %q", version, c.wantVersion)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int // sign only
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.1.0", "1.0.9", 1},
		{"2.0.0", "1.9.9", 1},
	}

	for _, c := range cases {
		got := compareVersions(c.a, c.b)
		gotSign := sign(got)
		if gotSign != c.want {
			t.Errorf("compareVersions(%q, %q) = %d (sign %d), want sign %d", c.a, c.b, got, gotSign, c.want)
		}
	}
}

func sign(n int) int {
	switch {
	case n < 0:
		return -1
	case n > 0:
		return 1
	default:
		return 0
	}
}

func TestShortCommit(t *testing.T) {
	if got, want := shortCommit("abcdef1234567890"), "abcdef1"; got != want {
		t.Errorf("shortCommit(long) = %q, want %q", got, want)
	}
	if got, want := shortCommit("abc"), "abc"; got != want {
		t.Errorf("shortCommit(short) = %q, want %q", got, want)
	}
}

func TestDefaultString(t *testing.T) {
	if got, want := defaultString("", "fallback"), "fallback"; got != want {
		t.Errorf("defaultString(\"\", ...) = %q, want %q", got, want)
	}
	if got, want := defaultString("explicit", "fallback"), "explicit"; got != want {
		t.Errorf("defaultString(explicit, ...) = %q, want %q", got, want)
	}
}
