package website

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
	"time"
)

// InspectOptions configures both Current and History: just enough to
// resolve the environment's website bucket, nothing deploy-specific.
type InspectOptions struct {
	// Env is the environment to inspect, e.g. "production". Version
	// tracking only exists for environments Deploy records markers for
	// (currently just "production" — see Deploy); inspecting any other
	// environment reports no versions.
	Env string

	InfraDir     string
	TerraformBin string

	Stdout, Stderr io.Writer
}

// Current prints the tag currently live in opts.Env — and nothing else,
// so it's easy to use in a script, e.g.:
//
//	if [ "$(kitsu website current --env production)" = "$TAG" ]; then ...
func Current(opts InspectOptions) error {
	_, _, bucket, err := resolveBucket(opts.Env, opts.InfraDir, opts.TerraformBin, opts.Stderr)
	if err != nil {
		return err
	}
	return currentForBucket(bucket, opts.Env, opts.Stdout)
}

// currentForBucket is Current's logic once the bucket is known, split
// out so it's testable without a real git repo/Terraform state (which
// resolveBucket needs) to get there.
func currentForBucket(bucket, env string, stdout io.Writer) error {
	current, err := s3ObjectContent(bucket, currentMarkerKey)
	if err != nil {
		return err
	}
	if current == "" {
		return fmt.Errorf("no version has been deployed to %q yet", env)
	}

	fmt.Fprintln(stdout, current)
	return nil
}

// History lists every version ever deployed to opts.Env (see Deploy),
// most recently deployed first, marking whichever one is currently
// live. Ordered by deploy time rather than version number, so a
// rollback (an older version deployed more recently) is visible as
// such rather than looking like a gap.
func History(opts InspectOptions) error {
	_, _, bucket, err := resolveBucket(opts.Env, opts.InfraDir, opts.TerraformBin, opts.Stderr)
	if err != nil {
		return err
	}
	return historyForBucket(bucket, opts.Env, opts.Stdout)
}

// historyForBucket is History's logic once the bucket is known, split
// out for the same reason as currentForBucket.
func historyForBucket(bucket, env string, stdout io.Writer) error {
	objects, err := s3ListObjects(bucket, versionMarkerPrefix+"/")
	if err != nil {
		return err
	}

	current, err := s3ObjectContent(bucket, currentMarkerKey)
	if err != nil {
		return err
	}

	type deployment struct {
		tag          string
		lastModified time.Time
	}

	var deployments []deployment
	for _, obj := range objects {
		if obj.Key == currentMarkerKey {
			continue
		}
		deployments = append(deployments, deployment{
			tag:          strings.TrimPrefix(obj.Key, versionMarkerPrefix+"/"),
			lastModified: obj.LastModified,
		})
	}
	if len(deployments) == 0 {
		fmt.Fprintf(stdout, "No versions have been deployed to %q yet.\n", env)
		return nil
	}

	sort.Slice(deployments, func(i, j int) bool {
		return deployments[i].lastModified.After(deployments[j].lastModified)
	})

	for _, d := range deployments {
		marker := ""
		if d.tag == current {
			marker = "  (current)"
		}
		fmt.Fprintf(stdout, "%-24s %s%s\n", d.tag, d.lastModified.UTC().Format("2006-01-02 15:04 MST"), marker)
	}
	return nil
}

// s3Object is the subset of `aws s3api list-objects-v2`'s output this
// package needs.
type s3Object struct {
	Key          string    `json:"Key"`
	LastModified time.Time `json:"LastModified"`
}

// s3ListObjects lists every object in bucket whose key starts with
// prefix. Returns an empty slice (not an error) if none match.
func s3ListObjects(bucket, prefix string) ([]s3Object, error) {
	cmd := exec.Command("aws", "s3api", "list-objects-v2", "--bucket", bucket, "--prefix", prefix, "--output", "json")
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("aws s3api list-objects-v2: %s", msg)
	}

	if strings.TrimSpace(out.String()) == "" {
		return nil, nil
	}

	var parsed struct {
		Contents []s3Object `json:"Contents"`
	}
	if err := json.Unmarshal(out.Bytes(), &parsed); err != nil {
		return nil, fmt.Errorf("parsing list-objects-v2 output: %w", err)
	}
	return parsed.Contents, nil
}
