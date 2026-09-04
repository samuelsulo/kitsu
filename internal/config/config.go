// Package config loads kitsu's configuration, which lives in two files
// sharing the same schema (Config): a global, per-user file holding
// personal defaults that vary by user but not by project (e.g. which
// Terraform module catalog to vendor from), and an optional local,
// per-project file — checked into a repository — that overrides the
// global file for anyone working on that project. Project conventions
// that vary by directory layout rather than by user or project identity
// (e.g. the infrastructure/<env> directory layout) stay as command
// flags instead of config keys.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/samuelsulo/kitsu/internal/git"
	"gopkg.in/yaml.v3"
)

// Config is kitsu's configuration schema, shared by both the global
// (Path) and local (ProjectPath) config files. Every field is a string,
// which Get/Set/Unset/Keys rely on to address it generically by its
// dot-separated yaml tag path (e.g. "terraform.catalog_repo"), rather
// than through a hand-maintained key registry that could drift from the
// struct.
type Config struct {
	Terraform TerraformConfig `yaml:"terraform"`
	Skills    SkillsConfig    `yaml:"skills"`
}

// TerraformConfig holds personal defaults for the `terraform` command
// group.
type TerraformConfig struct {
	// CatalogRepo is the git URL of the Terraform module catalog used by
	// `kitsu terraform catalog`.
	CatalogRepo string `yaml:"catalog_repo"`
	// RoleARNTemplate builds the cross-account IAM role ARN written by
	// `kitsu terraform scaffold environment`, with %s standing in for the
	// AWS account id (e.g. "arn:aws:iam::%s:role/MyAdminRole").
	RoleARNTemplate string `yaml:"role_arn_template"`
}

// SkillsConfig holds personal defaults for the `skills` command group.
type SkillsConfig struct {
	// Repo is the GitHub "owner/repo" of the Claude Code skills
	// repository used by `kitsu skills install`/`skills package`.
	Repo string `yaml:"repo"`
}

// DefaultSkillsRepo is used by ResolveSkillsRepo when neither an
// explicit flag, the local config file, nor the global one sets
// skills.repo.
const DefaultSkillsRepo = "samuelsulo/claude-skills"

// Path returns the global config file's path: <user config
// dir>/kitsu/config.yaml (see os.UserConfigDir —
// $XDG_CONFIG_HOME/kitsu/config.yaml on Linux).
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locating the user config directory: %w", err)
	}
	return filepath.Join(dir, "kitsu", "config.yaml"), nil
}

// ProjectPath returns the local config file's path: .kitsu.yaml at the
// root of the git repository containing the current directory. It
// errors if the current directory isn't inside a git repository.
func ProjectPath() (string, error) {
	root, err := git.Root("")
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".kitsu.yaml"), nil
}

// Load reads the global config file (see Path), returning a zero-value
// Config — not an error — if it doesn't exist yet.
func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}
	return LoadAt(path)
}

// LoadProject reads the local, per-project config file (see
// ProjectPath). Unlike Load and LoadAt, it tolerates not being inside a
// git repository — returning a zero-value Config, not an error — so
// callers that want project config folded in only when there is a
// project to have one (LoadMerged, in particular) don't need to special
// case it.
func LoadProject() (Config, error) {
	path, err := ProjectPath()
	if err != nil {
		return Config{}, nil
	}
	return LoadAt(path)
}

// LoadAt reads and parses the config file at path, returning a
// zero-value Config — not an error — if it doesn't exist yet.
func LoadAt(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, fmt.Errorf("reading %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parsing %s: %w", path, err)
	}
	return cfg, nil
}

// LoadMerged loads the global and local config files and combines them
// into the config kitsu actually uses: the local file's values win, key
// by key, falling back to the global file's where the local one leaves
// a key unset. This is what every Resolve* function reads.
func LoadMerged() (Config, error) {
	global, err := Load()
	if err != nil {
		return Config{}, err
	}
	project, err := LoadProject()
	if err != nil {
		return Config{}, err
	}
	return merge(global, project), nil
}

// Save writes cfg as YAML to path, creating its parent directory if
// needed.
func Save(path string, cfg Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("writing %s: %w", path, err)
	}
	return nil
}

// Keys returns every valid config key (e.g. "terraform.catalog_repo"),
// sorted, as accepted by Get, Set and Unset.
func Keys() []string {
	paths := fieldPaths()
	keys := make([]string, len(paths))
	for i, p := range paths {
		keys[i] = p.key
	}
	sort.Strings(keys)
	return keys
}

// Get returns key's value in cfg (e.g. "terraform.catalog_repo"),
// erroring if key isn't one of Keys().
func Get(cfg Config, key string) (string, error) {
	p, err := lookup(key)
	if err != nil {
		return "", err
	}
	return reflect.ValueOf(cfg).FieldByIndex(p.index).String(), nil
}

// Set writes value for key into cfg, erroring if key isn't one of
// Keys().
func Set(cfg *Config, key, value string) error {
	p, err := lookup(key)
	if err != nil {
		return err
	}
	reflect.ValueOf(cfg).Elem().FieldByIndex(p.index).SetString(value)
	return nil
}

// Unset clears key in cfg — equivalent to Set(cfg, key, "") — erroring
// if key isn't one of Keys().
func Unset(cfg *Config, key string) error {
	return Set(cfg, key, "")
}

// ResolveCatalogRepo returns explicit if non-empty, otherwise the
// merged config's terraform.catalog_repo (see LoadMerged), erroring
// with setup guidance if neither is set.
func ResolveCatalogRepo(explicit string) (string, error) {
	return resolve(explicit, func(c Config) string { return c.Terraform.CatalogRepo },
		"--catalog-repo", "terraform.catalog_repo")
}

// ResolveRoleARNTemplate returns explicit if non-empty, otherwise the
// merged config's terraform.role_arn_template (see LoadMerged), erroring
// with setup guidance if neither is set.
func ResolveRoleARNTemplate(explicit string) (string, error) {
	return resolve(explicit, func(c Config) string { return c.Terraform.RoleARNTemplate },
		"--role-arn-template", "terraform.role_arn_template")
}

// ResolveSkillsRepo returns explicit if non-empty, otherwise the merged
// config's skills.repo (see LoadMerged), otherwise DefaultSkillsRepo —
// unlike the other Resolve* functions, this one never errors: there's
// always a usable value.
func ResolveSkillsRepo(explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}

	cfg, err := LoadMerged()
	if err != nil {
		return "", err
	}
	if cfg.Skills.Repo != "" {
		return cfg.Skills.Repo, nil
	}
	return DefaultSkillsRepo, nil
}

// resolve returns explicit if non-empty, otherwise the value get reads
// from the merged config (see LoadMerged). If both are empty, it
// errors, naming the flag and key as the two ways to set it.
func resolve(explicit string, get func(Config) string, flag, key string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}

	cfg, err := LoadMerged()
	if err != nil {
		return "", err
	}
	if v := get(cfg); v != "" {
		return v, nil
	}

	path, _ := Path()
	return "", fmt.Errorf("not configured: pass %s, or set %s in %s (or a local .kitsu.yaml)", flag, key, path)
}

// fieldPath addresses one leaf string field of Config by the
// dot-separated path built from its yaml tags (e.g.
// "terraform.catalog_repo"), and the reflect.Value.FieldByIndex path
// needed to reach it.
type fieldPath struct {
	key   string
	index []int
}

// fieldPaths walks Config's fields, recursing into nested structs, and
// returns one fieldPath per leaf string field, built from their yaml
// tags. This makes Keys, Get, Set, Unset and merge generic over
// Config's schema: a new config key needs only a new struct field with
// a yaml tag, with no separate registry to keep in sync.
func fieldPaths() []fieldPath {
	var paths []fieldPath
	var walk func(t reflect.Type, prefix string, index []int)
	walk = func(t reflect.Type, prefix string, index []int) {
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			tag := strings.Split(f.Tag.Get("yaml"), ",")[0]
			if tag == "" || tag == "-" {
				continue
			}
			key := tag
			if prefix != "" {
				key = prefix + "." + tag
			}
			idx := append(append([]int{}, index...), i)

			if f.Type.Kind() == reflect.Struct {
				walk(f.Type, key, idx)
				continue
			}
			paths = append(paths, fieldPath{key: key, index: idx})
		}
	}
	walk(reflect.TypeOf(Config{}), "", nil)
	return paths
}

// lookup finds key's fieldPath, erroring with the full key list if key
// isn't one of Keys().
func lookup(key string) (fieldPath, error) {
	for _, p := range fieldPaths() {
		if p.key == key {
			return p, nil
		}
	}
	return fieldPath{}, fmt.Errorf("unknown config key %q (valid keys: %s)", key, strings.Join(Keys(), ", "))
}

// merge overlays override's non-empty string fields onto base,
// returning the result. Used by LoadMerged to combine the local
// (project) config over the global (user) one: the local file wins key
// by key where it sets a value, otherwise the global file's value
// stands.
func merge(base, override Config) Config {
	result := base
	for _, p := range fieldPaths() {
		v := reflect.ValueOf(override).FieldByIndex(p.index).String()
		if v != "" {
			reflect.ValueOf(&result).Elem().FieldByIndex(p.index).SetString(v)
		}
	}
	return result
}
