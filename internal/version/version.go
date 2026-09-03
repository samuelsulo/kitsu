// Package version holds build-time metadata injected via -ldflags at
// release time (see the Makefile). Defaults are used for local `go run`/`go
// build` without ldflags.
package version

var (
	// Version is the kitsu release version (e.g. "v0.1.0").
	Version = "dev"
	// Commit is the git commit SHA the binary was built from.
	Commit = "none"
	// Date is the build timestamp (RFC3339).
	Date = "unknown"
)
