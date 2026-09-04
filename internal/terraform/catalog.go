package terraform

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// tagRefPattern matches a `git ls-remote` ref line's tag name, e.g.
// "refs/tags/website-hosting/v1.0.1".
var tagRefPattern = regexp.MustCompile(`^refs/tags/(.+)$`)

// listCatalogTags returns every tag in repo matching refspec (e.g.
// "<module>/*", or "" for every tag), stripped of the "refs/tags/"
// prefix.
func listCatalogTags(repo, refspec string) ([]string, error) {
	args := []string{"ls-remote", "--tags", repo}
	if refspec != "" {
		args = append(args, refspec)
	}

	cmd := exec.Command("git", args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("git ls-remote --tags %s: %w", repo, err)
	}

	var tags []string
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		// For an annotated tag, ls-remote lists two lines: the tag ref
		// itself, and a second, "peeled" refs/tags/<name>^{} line for the
		// commit it points to. Skip the peeled one — the tag ref alone
		// already gives us the name — or every annotated tag's version
		// would be listed (and so printed) twice.
		if strings.HasSuffix(fields[1], "^{}") {
			continue
		}
		if m := tagRefPattern.FindStringSubmatch(fields[1]); m != nil {
			tags = append(tags, m[1])
		}
	}
	return tags, nil
}

// CatalogList lists the distinct module names available in repo — every
// tag's name up to its last "/version" segment — sorted.
func (r Runner) CatalogList(repo string) error {
	tags, err := listCatalogTags(repo, "")
	if err != nil {
		return err
	}

	modules := map[string]struct{}{}
	for _, tag := range tags {
		if i := strings.LastIndex(tag, "/"); i >= 0 {
			modules[tag[:i]] = struct{}{}
		}
	}
	if len(modules) == 0 {
		return fmt.Errorf("no modules found in %s", repo)
	}

	names := make([]string, 0, len(modules))
	for name := range modules {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		fmt.Fprintln(r.Stdout, name)
	}
	return nil
}

// CatalogVersions lists the available versions of module in repo, newest
// first.
func (r Runner) CatalogVersions(repo, module string) error {
	tags, err := listCatalogTags(repo, module+"/*")
	if err != nil {
		return err
	}

	prefix := module + "/"
	var versions []string
	for _, tag := range tags {
		if v, ok := strings.CutPrefix(tag, prefix); ok {
			versions = append(versions, v)
		}
	}
	if len(versions) == 0 {
		return fmt.Errorf("no tagged versions found for module %q", module)
	}

	sort.Slice(versions, func(i, j int) bool { return compareVersions(versions[i], versions[j]) > 0 })

	for _, v := range versions {
		fmt.Fprintln(r.Stdout, v)
	}
	return nil
}

// compareVersions compares two "vX.Y.Z" strings, returning <0, 0 or >0
// as a<b, a==b, a>b. Missing or non-numeric segments compare as 0.
func compareVersions(a, b string) int {
	pa, pb := versionParts(a), versionParts(b)
	for i := range pa {
		if pa[i] != pb[i] {
			return pa[i] - pb[i]
		}
	}
	return 0
}

func versionParts(v string) [3]int {
	var parts [3]int
	for i, s := range strings.SplitN(strings.TrimPrefix(v, "v"), ".", 3) {
		if i >= len(parts) {
			break
		}
		parts[i], _ = strconv.Atoi(s)
	}
	return parts
}

// CatalogVendor copies module at version from repo into
// <infra-dir>/modules/vendor/<module>, replacing anything already there,
// and records provenance in a VENDORED.md file.
func (r Runner) CatalogVendor(repo, module, version string) error {
	ref := module + "/" + version

	tmpDir, err := os.MkdirTemp("", "kitsu-vendor-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpDir)

	cloneCmd := exec.Command("git", "clone", "--quiet", "--depth", "1", "--branch", ref, repo, tmpDir)
	if out, err := cloneCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("cloning %s at ref %q: %w (%s)", repo, ref, err, strings.TrimSpace(string(out)))
	}

	srcDir := filepath.Join(tmpDir, "modules", module)
	if info, err := os.Stat(srcDir); err != nil || !info.IsDir() {
		return fmt.Errorf("module %q not found in catalog at ref %q", module, ref)
	}

	commitCmd := exec.Command("git", "-C", tmpDir, "rev-parse", "HEAD")
	var commitOut bytes.Buffer
	commitCmd.Stdout = &commitOut
	if err := commitCmd.Run(); err != nil {
		return fmt.Errorf("resolving commit for %s: %w", ref, err)
	}
	commit := strings.TrimSpace(commitOut.String())

	destDir := filepath.Join(r.Env.infraDir(), "modules", "vendor", module)
	if err := os.RemoveAll(destDir); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destDir), 0o755); err != nil {
		return err
	}
	if err := os.CopyFS(destDir, os.DirFS(srcDir)); err != nil {
		return fmt.Errorf("copying %s: %w", srcDir, err)
	}
	// Defensive: the module itself shouldn't carry a nested .git, but
	// strip one if it somehow does (e.g. vendored from a submodule).
	if err := os.RemoveAll(filepath.Join(destDir, ".git")); err != nil {
		return err
	}

	vendoredMD := fmt.Sprintf(
		"# Vendored module\n\nCopied verbatim from the catalog. Do not modify directly: re-run\n`kitsu terraform catalog vendor %s <version>` instead.\n\n"+
			"| | |\n|---|---|\n| Source | %s |\n| Module | %s |\n| Ref | %s |\n| Commit | %s |\n| Vendored on | %s |\n",
		module, repo, module, ref, commit, time.Now().UTC().Format("2006-01-02"),
	)
	if err := os.WriteFile(filepath.Join(destDir, "VENDORED.md"), []byte(vendoredMD), 0o644); err != nil {
		return err
	}

	fmt.Fprintf(r.Stdout, "✓ modules/vendor/%s/ vendored from %s (commit %s)\n", module, ref, commit)
	return nil
}
