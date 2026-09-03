package cli

import (
	"io"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestReadLine_DoesNotOverconsume(t *testing.T) {
	// Two lines available upfront, as when input is piped rather than
	// typed live: readLine must return only the first and leave the
	// second untouched for a later, unrelated reader of the same stream
	// (e.g. Terraform's own destroy confirmation prompt).
	r := strings.NewReader("yes\nyes\n")

	first, err := readLine(r)
	if err != nil {
		t.Fatalf("first readLine: %v", err)
	}
	if first != "yes" {
		t.Errorf("first = %q, want %q", first, "yes")
	}

	rest, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("reading the rest: %v", err)
	}
	if string(rest) != "yes\n" {
		t.Errorf("remaining input = %q, want %q (readLine must not consume past its own line)", rest, "yes\n")
	}
}

func TestConfirm(t *testing.T) {
	cases := map[string]struct {
		input string
		want  bool
	}{
		"yes":                    {"yes\n", true},
		"no":                     {"no\n", false},
		"empty line":             {"\n", false},
		"EOF, no input":          {"", false},
		"surrounding whitespace": {" yes \n", true}, // trimmed, like bash's `read` splitting on IFS
	}

	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			cmd := &cobra.Command{}
			cmd.SetOut(io.Discard)
			cmd.SetIn(strings.NewReader(c.input))

			got, err := confirm(cmd, "Destroying stuff.")
			if err != nil {
				t.Fatalf("confirm: %v", err)
			}
			if got != c.want {
				t.Errorf("confirm(%q) = %v, want %v", c.input, got, c.want)
			}
		})
	}
}
