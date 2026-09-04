package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/samuelsulo/kitsu/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

// configScope selects which config file a config subcommand reads or
// writes.
type configScope int

const (
	// scopeMerged reads the effective config: the local (project) file's
	// values, falling back to the global (user) file's. It's only valid
	// for read commands (get/show/path) — writes always target one file.
	scopeMerged configScope = iota
	scopeGlobal
	scopeLocal
)

// configScopeFlags adds the --global/--local flags shared by every
// config subcommand and returns a resolver for them. required means the
// command writes to a single file, so exactly one of the two must be
// passed; otherwise neither is allowed too, meaning "the effective,
// merged config" (see scopeMerged).
func configScopeFlags(cmd *cobra.Command, required bool) (resolve func() (configScope, error)) {
	var global, local bool
	cmd.Flags().BoolVar(&global, "global", false, "Use the global, per-user config file")
	cmd.Flags().BoolVar(&local, "local", false, "Use the local, per-project config file (the git repository's .kitsu.yaml)")

	return func() (configScope, error) {
		switch {
		case global && local:
			return 0, fmt.Errorf("--global and --local are mutually exclusive")
		case global:
			return scopeGlobal, nil
		case local:
			return scopeLocal, nil
		case required:
			return 0, fmt.Errorf("specify --global or --local")
		default:
			return scopeMerged, nil
		}
	}
}

// loadConfigForScope reads the config file(s) named by scope: the
// merged, effective config (see config.LoadMerged) for scopeMerged, or
// one file's raw content for scopeGlobal/scopeLocal.
func loadConfigForScope(scope configScope) (config.Config, error) {
	switch scope {
	case scopeGlobal:
		return config.Load()
	case scopeLocal:
		path, err := config.ProjectPath()
		if err != nil {
			return config.Config{}, err
		}
		return config.LoadAt(path)
	default:
		return config.LoadMerged()
	}
}

// writableConfigPath returns the file path scope addresses (scopeGlobal
// or scopeLocal only — set/unset/edit require one of them via
// configScopeFlags(cmd, true)).
func writableConfigPath(scope configScope) (string, error) {
	if scope == scopeGlobal {
		return config.Path()
	}
	return config.ProjectPath()
}

// newConfigCmd builds the "config" command group.
func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Read and edit kitsu's configuration files",
		Long: `config manages kitsu's two configuration files: the
global, per-user one (personal defaults that don't vary by project —
see 'kitsu config path') and the local, per-project one (a
.kitsu.yaml at the root of the current git repository, checked into
it so it applies to everyone working there). Where both set the same
key, the local file wins.

Every other kitsu command that reads a config value (e.g.
'terraform scaffold environment') already reads this merged,
effective config — these commands are for inspecting and editing the
files directly.`,
	}

	cmd.AddCommand(
		newConfigGetCmd(),
		newConfigSetCmd(),
		newConfigUnsetCmd(),
		newConfigShowCmd(),
		newConfigPathCmd(),
		newConfigEditCmd(),
	)

	return cmd
}

func newConfigGetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Print one config key's value",
		Long: `get prints the value of <key>, a dot-separated path such
as "terraform.catalog_repo" (see 'kitsu config show' for the full
list of keys).

Without --global/--local, it prints the effective value: the local
(project) file's, or the global (user) file's if the local one
doesn't set it. --global or --local prints that one file's raw value
instead, ignoring the other.`,
		Args: cobra.ExactArgs(1),
	}

	resolveScope := configScopeFlags(cmd, false)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		scope, err := resolveScope()
		if err != nil {
			return err
		}
		cfg, err := loadConfigForScope(scope)
		if err != nil {
			return err
		}
		value, err := config.Get(cfg, args[0])
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), value)
		return nil
	}

	return cmd
}

func newConfigSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set one config key's value",
		Long: `set writes <value> for <key> (see 'kitsu config show'
for the full list of keys) into one config file. --global or --local
is required: writing the wrong one by mistake means either leaking a
personal value into a shared repository, or a value meant for the
whole project silently only applying to you.`,
		Args: cobra.ExactArgs(2),
	}

	resolveScope := configScopeFlags(cmd, true)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		scope, err := resolveScope()
		if err != nil {
			return err
		}
		path, err := writableConfigPath(scope)
		if err != nil {
			return err
		}
		cfg, err := config.LoadAt(path)
		if err != nil {
			return err
		}
		if err := config.Set(&cfg, args[0], args[1]); err != nil {
			return err
		}
		return config.Save(path, cfg)
	}

	return cmd
}

func newConfigUnsetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unset <key>",
		Short: "Clear one config key's value",
		Long: `unset clears <key> (equivalent to 'set <key> ""') in one
config file. --global or --local is required, for the same reason as
'set'.`,
		Args: cobra.ExactArgs(1),
	}

	resolveScope := configScopeFlags(cmd, true)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		scope, err := resolveScope()
		if err != nil {
			return err
		}
		path, err := writableConfigPath(scope)
		if err != nil {
			return err
		}
		cfg, err := config.LoadAt(path)
		if err != nil {
			return err
		}
		if err := config.Unset(&cfg, args[0]); err != nil {
			return err
		}
		return config.Save(path, cfg)
	}

	return cmd
}

func newConfigShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Print the whole config as YAML",
		Long: `show prints the whole config as YAML.

Without --global/--local, it prints the effective config: the local
(project) file's values override the global (user) file's, key by
key. --global or --local prints that one file's raw content instead,
ignoring the other.`,
		Args: cobra.NoArgs,
	}

	resolveScope := configScopeFlags(cmd, false)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		scope, err := resolveScope()
		if err != nil {
			return err
		}
		cfg, err := loadConfigForScope(scope)
		if err != nil {
			return err
		}
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("encoding config: %w", err)
		}
		_, err = cmd.OutOrStdout().Write(data)
		return err
	}

	return cmd
}

func newConfigPathCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "path",
		Short: "Print the config file path(s)",
		Long: `path prints the global config file's path and, if run
inside a git repository, the local (project) config file's path too
— one per line, global first. Neither file needs to exist yet.
--global or --local restricts the output to just one, erroring for
--local outside a git repository.`,
		Args: cobra.NoArgs,
	}

	resolveScope := configScopeFlags(cmd, false)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		scope, err := resolveScope()
		if err != nil {
			return err
		}

		out := cmd.OutOrStdout()

		if scope != scopeLocal {
			path, err := config.Path()
			if err != nil {
				return err
			}
			fmt.Fprintln(out, path)
		}

		if scope != scopeGlobal {
			path, err := config.ProjectPath()
			if err != nil {
				if scope == scopeLocal {
					return err
				}
				// scope == scopeMerged: not being inside a git repository
				// just means there's no local file to print, same as
				// get/show's tolerant merge behavior.
			} else {
				fmt.Fprintln(out, path)
			}
		}

		return nil
	}

	return cmd
}

func newConfigEditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "edit",
		Short: "Open a config file in $EDITOR",
		Long: `edit opens the config file named by --global or --local
(required, for the same reason as 'set') in $EDITOR (defaulting to
"vi" if unset), creating an empty file first if it doesn't exist
yet.`,
		Args: cobra.NoArgs,
	}

	resolveScope := configScopeFlags(cmd, true)

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		scope, err := resolveScope()
		if err != nil {
			return err
		}
		path, err := writableConfigPath(scope)
		if err != nil {
			return err
		}

		if _, err := os.Stat(path); os.IsNotExist(err) {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				return fmt.Errorf("creating %s: %w", filepath.Dir(path), err)
			}
			if err := os.WriteFile(path, nil, 0o644); err != nil {
				return fmt.Errorf("creating %s: %w", path, err)
			}
		} else if err != nil {
			return fmt.Errorf("checking %s: %w", path, err)
		}

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "vi"
		}
		editCmd := exec.Command(editor, path)
		editCmd.Stdin = cmd.InOrStdin()
		editCmd.Stdout = cmd.OutOrStdout()
		editCmd.Stderr = cmd.ErrOrStderr()
		return editCmd.Run()
	}

	return cmd
}
