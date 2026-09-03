// Package config loads kitsu's per-user configuration file, holding
// personal defaults that vary by user but not by project (e.g. which
// Terraform module catalog to vendor from) — as opposed to project-level
// conventions (e.g. the infrastructure/<env> directory layout), which
// stay as command flags instead.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config is kitsu's per-user configuration, loaded from the file at
// Path().
type Config struct {
	Terraform TerraformConfig `yaml:"terraform"`
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

// Path returns the config file's path: <user config dir>/kitsu/config.yaml
// (see os.UserConfigDir — $XDG_CONFIG_HOME/kitsu/config.yaml on Linux).
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locating the user config directory: %w", err)
	}
	return filepath.Join(dir, "kitsu", "config.yaml"), nil
}

// Load reads the config file, returning a zero-value Config — not an
// error — if it doesn't exist yet.
func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}

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

// ResolveCatalogRepo returns explicit if non-empty, otherwise the config
// file's terraform.catalog_repo, erroring with setup guidance if neither
// is set.
func ResolveCatalogRepo(explicit string) (string, error) {
	return resolve(explicit, func(c Config) string { return c.Terraform.CatalogRepo },
		"--catalog-repo", "terraform.catalog_repo")
}

// ResolveRoleARNTemplate returns explicit if non-empty, otherwise the
// config file's terraform.role_arn_template, erroring with setup
// guidance if neither is set.
func ResolveRoleARNTemplate(explicit string) (string, error) {
	return resolve(explicit, func(c Config) string { return c.Terraform.RoleARNTemplate },
		"--role-arn-template", "terraform.role_arn_template")
}

// resolve returns explicit if non-empty, otherwise the value get reads
// from the loaded config file. If both are empty, it errors, naming flag
// and key as the two ways to set it.
func resolve(explicit string, get func(Config) string, flag, key string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}

	cfg, err := Load()
	if err != nil {
		return "", err
	}
	if v := get(cfg); v != "" {
		return v, nil
	}

	path, _ := Path()
	return "", fmt.Errorf("not configured: pass %s, or set %s in %s", flag, key, path)
}
