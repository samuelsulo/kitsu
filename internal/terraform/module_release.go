package terraform

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/samuelsulo/kitsu/internal/git"
)

// ReleaseModuleOptions configures ReleaseModule.
type ReleaseModuleOptions struct {
	// Module is the module's name, e.g. "website-hosting".
	Module string
	// Version is the release version, as "vX.Y.Z" (matching the
	// "<module>/vX.Y.Z" tag convention used by CatalogVendor/
	// CatalogVersions).
	Version string
	// ModulesDir is the catalog repository's modules directory,
	// relative to its root. Empty means "modules".
	ModulesDir string
	// Push also pushes the current branch and the new tag to origin.
	// Without it, ReleaseModule only prints the commands to run.
	Push bool

	Stdout, Stderr io.Writer
}

// moduleVersionPattern matches a release version given as "vX.Y.Z".
var moduleVersionPattern = regexp.MustCompile(`^v(\d+\.\d+\.\d+)$`)

// unreleasedHeading is the exact line a module's CHANGELOG.md carries
// until it's released — no brackets, unlike a resolved "## [X.Y.Z]"
// heading. This is the module catalog's own changelog convention,
// distinct from (and not to be confused with) a project's own
// top-level CHANGELOG.md, which keeps "## [Unreleased]" bracketed.
const unreleasedHeading = "## Unreleased"

// ReleaseModule finalizes one module's release inside the current git
// repository — expected to be a checkout of the Terraform module
// catalog itself (see CatalogVendor/CatalogList/CatalogVersions for the
// consumer side of that same catalog). It turns the module's own
// CHANGELOG.md "## Unreleased" section into "## [X.Y.Z] - <date>" (or
// leaves an already-prepared entry for that version as-is, so
// preparing the changelog by hand first and re-running this is
// idempotent), updates its version cell in <modules-dir>/README.md,
// commits both, and creates the annotated "<module>/vX.Y.Z" tag.
//
// Refuses to run if any other tracked change is pending outside the
// module's own directory and the README index, so nothing unrelated
// gets swept into the release commit.
func ReleaseModule(opts ReleaseModuleOptions) error {
	m := moduleVersionPattern.FindStringSubmatch(opts.Version)
	if m == nil {
		return fmt.Errorf(`version must be given as "vX.Y.Z", got %q`, opts.Version)
	}
	bareVersion := m[1]

	repoRoot, err := git.Root("")
	if err != nil {
		return err
	}

	modulesDir := defaultString(opts.ModulesDir, "modules")
	moduleRelDir := filepath.Join(modulesDir, opts.Module)
	moduleDir := filepath.Join(repoRoot, moduleRelDir)
	changelogFile := filepath.Join(moduleDir, "CHANGELOG.md")
	readmeIndexRel := filepath.Join(modulesDir, "README.md")
	readmeIndex := filepath.Join(repoRoot, readmeIndexRel)
	tag := opts.Module + "/" + opts.Version

	if info, err := os.Stat(moduleDir); err != nil || !info.IsDir() {
		return fmt.Errorf("no module at %s", moduleRelDir)
	}
	if _, err := os.Stat(changelogFile); err != nil {
		return fmt.Errorf("%s does not exist", filepath.Join(moduleRelDir, "CHANGELOG.md"))
	}

	readmeContent, err := os.ReadFile(readmeIndex)
	if err != nil {
		return fmt.Errorf("reading %s: %w", readmeIndexRel, err)
	}
	moduleLink := fmt.Sprintf("[%s](", opts.Module)
	if !strings.Contains(string(readmeContent), moduleLink) {
		return fmt.Errorf("no row for %q in %s", opts.Module, readmeIndexRel)
	}

	if err := gitRun(repoRoot, "rev-parse", tag); err == nil {
		return fmt.Errorf("tag %s already exists", tag)
	}

	if err := checkNoUnrelatedChanges(repoRoot, moduleRelDir, readmeIndexRel); err != nil {
		return err
	}

	newHeading := fmt.Sprintf("## [%s] - %s", bareVersion, time.Now().Format("2006-01-02"))
	versionedHeadingPattern := regexp.MustCompile(`(?m)^## \[` + regexp.QuoteMeta(bareVersion) + `\] - \d{4}-\d{2}-\d{2}$`)

	changelog, err := os.ReadFile(changelogFile)
	if err != nil {
		return fmt.Errorf("reading %s: %w", filepath.Join(moduleRelDir, "CHANGELOG.md"), err)
	}
	switch {
	case containsLine(string(changelog), unreleasedHeading):
		updated := replaceFirstLine(string(changelog), unreleasedHeading, newHeading)
		if err := os.WriteFile(changelogFile, []byte(updated), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(opts.Stdout, "✓ Turned %q into %q in %s\n", unreleasedHeading, newHeading, filepath.Join(moduleRelDir, "CHANGELOG.md"))
	case versionedHeadingPattern.MatchString(string(changelog)):
		fmt.Fprintf(opts.Stdout, "✓ %s already has an entry for %s, leaving it as-is.\n", filepath.Join(moduleRelDir, "CHANGELOG.md"), opts.Version)
	default:
		return fmt.Errorf(
			"%s has neither an %q section nor an entry for %s: add the release notes first, then re-run",
			filepath.Join(moduleRelDir, "CHANGELOG.md"), unreleasedHeading, opts.Version,
		)
	}

	// Replace only this module's row's version cell, wherever it
	// currently sits in the table — no assumption about line numbers.
	cellPattern := regexp.MustCompile(`\| v\d+\.\d+\.\d+ \|$`)
	lines := strings.Split(string(readmeContent), "\n")
	for i, line := range lines {
		if strings.Contains(line, moduleLink) {
			lines[i] = cellPattern.ReplaceAllString(line, "| "+opts.Version+" |")
		}
	}
	if err := os.WriteFile(readmeIndex, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(opts.Stdout, "✓ Updated %s: %s -> %s.\n", readmeIndexRel, opts.Module, opts.Version)

	if err := gitRunOutput(repoRoot, opts.Stderr, "add", moduleRelDir, readmeIndexRel); err != nil {
		return err
	}
	if err := gitRun(repoRoot, "diff", "--cached", "--quiet"); err == nil {
		return fmt.Errorf("nothing to commit: has %s already been released?", opts.Version)
	}

	commitMessage := fmt.Sprintf("docs(%s): release %s", opts.Module, opts.Version)
	if err := gitRunOutput(repoRoot, opts.Stderr, "commit", "-q", "-m", commitMessage); err != nil {
		return err
	}
	fmt.Fprintf(opts.Stdout, "✓ Committed: %s\n", commitMessage)

	tagMessage := opts.Module + " " + opts.Version
	if err := gitRunOutput(repoRoot, opts.Stderr, "tag", "-a", tag, "-m", tagMessage); err != nil {
		return err
	}
	fmt.Fprintf(opts.Stdout, "✓ Tagged %s\n", tag)

	branch, err := gitOutputTrimmed(repoRoot, "symbolic-ref", "--short", "HEAD")
	if err != nil {
		return err
	}

	if opts.Push {
		if err := gitRunOutput(repoRoot, opts.Stderr, "push", "origin", branch); err != nil {
			return err
		}
		if err := gitRunOutput(repoRoot, opts.Stderr, "push", "origin", tag); err != nil {
			return err
		}
		fmt.Fprintf(opts.Stdout, "✓ Pushed %s and %s to origin.\n", branch, tag)
	} else {
		fmt.Fprintln(opts.Stdout)
		fmt.Fprintln(opts.Stdout, "Not pushed. Run when ready:")
		fmt.Fprintf(opts.Stdout, "  git push origin %s\n", branch)
		fmt.Fprintf(opts.Stdout, "  git push origin %s\n", tag)
	}

	return nil
}

// defaultString returns v, or fallback if v is empty.
func defaultString(v, fallback string) string {
	if v == "" {
		return fallback
	}
	return v
}

// checkNoUnrelatedChanges errors out if any tracked change is pending
// outside moduleRelDir/ and readmeIndexRel — the two paths a release
// commit is expected to carry — so nothing else gets swept into it.
// Untracked clutter elsewhere in the working tree is not this check's
// concern.
func checkNoUnrelatedChanges(repoRoot, moduleRelDir, readmeIndexRel string) error {
	cmd := exec.Command("git", "-C", repoRoot, "status", "--porcelain", "--untracked-files=no", "--", ".")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git status: %w", err)
	}

	modulePrefix := filepath.ToSlash(moduleRelDir) + "/"
	readmePath := filepath.ToSlash(readmeIndexRel)

	var unrelated []string
	for _, line := range strings.Split(strings.TrimRight(out.String(), "\n"), "\n") {
		if line == "" {
			continue
		}
		// Porcelain format: two status chars, a space, then the path.
		if len(line) < 4 {
			continue
		}
		path := filepath.ToSlash(strings.TrimSpace(line[3:]))
		if strings.HasPrefix(path, modulePrefix) || path == readmePath {
			continue
		}
		unrelated = append(unrelated, line)
	}

	if len(unrelated) > 0 {
		return fmt.Errorf(
			"tracked changes outside %s/ and %s would be swept into this release commit:\n%s",
			moduleRelDir, readmeIndexRel, strings.Join(unrelated, "\n"),
		)
	}
	return nil
}

// containsLine reports whether text has line exactly matching one of
// its own lines (trimming only the trailing newline convention, not
// surrounding whitespace).
func containsLine(text, line string) bool {
	for _, l := range strings.Split(text, "\n") {
		if l == line {
			return true
		}
	}
	return false
}

// replaceFirstLine replaces the first line in text equal to old with
// new, leaving every other line (including any later occurrence of
// old) untouched.
func replaceFirstLine(text, old, new string) string {
	lines := strings.Split(text, "\n")
	for i, l := range lines {
		if l == old {
			lines[i] = new
			break
		}
	}
	return strings.Join(lines, "\n")
}

// gitRun runs `git -C repoRoot <args...>`, discarding its output,
// returning an error only if it exits non-zero.
func gitRun(repoRoot string, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
	return cmd.Run()
}

// gitRunOutput runs `git -C repoRoot <args...>`, streaming stderr
// through stderr, and returns an error naming the command on failure.
func gitRunOutput(repoRoot string, stderr io.Writer, args ...string) error {
	cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
	cmd.Stderr = stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

// gitOutputTrimmed runs `git -C repoRoot <args...>` and returns its
// trimmed stdout.
func gitOutputTrimmed(repoRoot string, args ...string) (string, error) {
	cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(out.String()), nil
}
