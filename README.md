# kitsu

A personal CLI to automate recurring tasks across projects, distributed
as a single Go binary.

## Requirements

- [Go](https://go.dev/) 1.27+

## Build

```sh
make build      # builds ./bin/kitsu
```

## Install

### Via mise (in other projects)

Released versions (tagged `vX.Y.Z`, published as
[GitHub Releases](https://github.com/samuelsulo/kitsu/releases) by
[`.github/workflows/release.yaml`](.github/workflows/release.yaml)) are
installable with [mise](https://mise.jdx.dev/), via its `github:`
backend — no plugin needed:

```sh
mise use -g github:samuelsulo/kitsu@1.0.0
```

Or pinned in a project's `mise.toml`:

```toml
[tools]
"github:samuelsulo/kitsu" = "1.0.0"
```

### From source

```sh
make install    # builds and installs kitsu into $GOBIN (or $GOPATH/bin)
```

## Usage

```sh
kitsu --help
kitsu version
```

Or, from source, without building first:

```sh
make run ARGS="version"
```

## Development

```sh
make test        # run tests
make lint        # gofmt + go vet
```

See [CLAUDE.md](CLAUDE.md) for project rules (language, documentation,
changelog/versioning, commit conventions) and
[CHANGELOG.md](CHANGELOG.md) for the history of changes.

### Releasing

1. Move `CHANGELOG.md`'s `[Unreleased]` content under a new
   `## [vX.Y.Z] - YYYY-MM-DD` heading, and leave a fresh empty
   `[Unreleased]` on top.
2. Commit that (`chore(repo): release vX.Y.Z`).
3. Tag the resulting commit — `git tag -a vX.Y.Z -m vX.Y.Z` — matching
   the changelog heading exactly, and push it: `git push origin vX.Y.Z`.
4. Pushing the tag triggers
   [`.github/workflows/release.yaml`](.github/workflows/release.yaml),
   which runs GoReleaser (see [`.goreleaser.yaml`](.goreleaser.yaml)) to
   build linux/darwin × amd64/arm64 binaries and publish them as a
   GitHub Release — installable right after via `mise` (see
   [Install](#install)).

## Commands

| Command         | Description                                              |
|-----------------|-----------------------------------------------------------|
| `version`       | Print kitsu's version info.                              |
| `hooks install` | Point git at the repository's tracked hooks (`--dir`, default `.githooks`) and make them executable. |
| `terraform ...` | Run Terraform against the `infrastructure/<env>` convention. See [Terraform workflow](#terraform-workflow). |
| `website ...`   | Build/deploy the project's website, or inspect what's deployed. See [Website deploy](#website-deploy). |
| `skills ...`     | Install or package a Claude Code skill. See [Skills](#skills). |

More commands will be added here as they're implemented.

## Terraform workflow

`kitsu terraform` wraps the Terraform CLI with the directory and flag
conventions shared across projects: one Terraform root at
`<infra-dir>/live`, configured per environment from
`<infra-dir>/environments/<env>/{backend.hcl,environment.tfvars}`. It
replaces a per-project Terraform `Makefile` used the same way.

Global flags, given before the subcommand:

| Flag              | Default          | Description                                                       |
|-------------------|------------------|--------------------------------------------------------------------|
| `--env`           | `sandbox`        | Environment name (a directory under `<infra-dir>/environments/`). |
| `--infra-dir`     | `infrastructure` | Infrastructure root directory.                                    |
| `--terraform-bin` | `terraform`      | Terraform binary to invoke.                                       |
| `--terraform-docs-bin` | `terraform-docs` | terraform-docs binary to invoke (used by `docs`).             |

Subcommands:

| Command                            | Description |
|-------------------------------------|--------------|
| `init`                              | Initialize Terraform and configure the backend. |
| `validate`                          | Initialize (see `init`) and validate the configuration. |
| `fmt`                               | Format Terraform (`.tf`/`.tfvars`) and generic HCL files (e.g. `backend.hcl`). |
| `fmt-check`                         | Check formatting without modifying files (useful in CI). |
| `fmt-staged <file>...`              | Format exactly the given files, not the whole tree — for a pre-commit hook that formats only staged files. |
| `plan`                              | Validate (see `validate`) and generate and save an execution plan. |
| `show-plan`                         | Show the plan previously saved by `plan`. |
| `apply`                             | Apply the plan previously saved by `plan`. |
| `apply-target --target=<address>`   | Validate and apply changes to a single resource target, without a saved plan. |
| `apply-auto`                        | Validate, then plan and apply in one step, without a saved plan. Use with caution. |
| `plan-destroy`                      | Preview what a destroy would remove, without applying anything. |
| `refresh`                           | Reconcile the state with the real infrastructure, without changing either. |
| `destroy`                           | Destroy every resource in the target environment. Asks for confirmation, then Terraform's own. |
| `destroy-target --target=<address>` | Destroy a single resource target. Same double confirmation as `destroy`. |
| `import <address> <id>`             | Import an existing resource, identified by its provider-specific id, into the Terraform state. |
| `state list`                        | List every resource in the Terraform state. |
| `state show <address>`              | Show the state attributes of a single resource. |
| `state rm <address>`                | Remove a resource from the state without destroying it. Asks for confirmation. |
| `unlock <lock-id>`                  | Force-release a stuck state lock. Asks for confirmation. |
| `taint <address>`                   | Mark a resource for recreation on the next apply. |
| `untaint <address>`                 | Undo a previous `taint`. |
| `console`                           | Open an interactive console to evaluate expressions against the current state. |
| `providers`                         | Show the provider requirements and versions in use. |
| `version`                           | Show the Terraform and provider versions (Terraform's own, not kitsu's — see `kitsu version`). |
| `upgrade`                           | Re-initialize and upgrade provider/module versions to the latest allowed. |
| `output`                            | Show the outputs of the environment's current state. |
| `output-json`                       | Show the outputs of the environment's current state, in JSON format. |
| `clean`                             | Remove the local Terraform cache and this environment's saved plan. |
| `docs`                              | Regenerate each module's README input/output tables with terraform-docs, for every module directory under `<infra-dir>/modules/*/*`. Requires `terraform-docs` on `PATH` (or `--terraform-docs-bin`). |
| `scaffold environment --account-id=<id>` | Scaffold `<infra-dir>/environments/<env>/{environment.tfvars,backend.hcl}` for a new AWS account, reading `project`/`aws_region` from `live/project.auto.tfvars`. `--env` must be `sandbox` or `production`. |
| `scaffold module <name>`            | Scaffold `<infra-dir>/modules/local/<name>/` with the standard empty module files (`main.tf`, `variables.tf`, `outputs.tf`, `versions.tf`, `README.md`), skipping any that already exist. |
| `catalog list`                      | List modules available in the module catalog. |
| `catalog versions <module>`         | List a catalog module's available versions, newest first. |
| `catalog vendor <module> <version>` | Copy a module from the catalog into `<infra-dir>/modules/vendor/<module>`, pinned to that version's tag, with provenance recorded in `VENDORED.md`. |
| `catalog release <module> <version> [--push]` | Run from *inside a checkout of the catalog itself* (not a project that vendors from it): finalizes that module's `CHANGELOG.md` entry, updates its version cell in `<modules-dir>/README.md` (default `modules`), commits both, and tags `<module>/vX.Y.Z`. Refuses if any other tracked change is pending, so nothing unrelated is swept into the release commit. |
| `bootstrap-backend`                 | Create (once per AWS account) the S3 bucket for the Terraform state shared by every project in that account: versioning, encryption, public access block, TLS-only policy, lifecycle on noncurrent versions. Idempotent. Operates on the account of the currently active AWS credentials — never an account id passed by hand. |

Example:

```sh
kitsu terraform plan --env production
kitsu terraform apply --env production
```

## Website deploy

`kitsu website deploy` builds `<website-dir>/` and syncs it to the S3
bucket + CloudFront distribution of the given environment, read from
that environment's Terraform state (never guessed or hardcoded) —
following the company's vue-webapp-standard convention (Vite build,
`VITE_APP_VERSION`, the footer `SiteVersion` component).

| Flag              | Default          | Description                                                       |
|-------------------|------------------|---------------------------------------------------------------------|
| `--env`           | *(required)*     | `sandbox` or `production`.                                        |
| `--tag`           | —                | Release tag to deploy, e.g. `website/v1.0.1`. Required for `production`, forbidden otherwise. |
| `--force`         | `false`          | Skip the already-deployed/downgrade guards (production only).     |
| `--infra-dir`     | `infrastructure` | Infrastructure root directory, relative to the repo root.         |
| `--website-dir`   | `website`        | Website project directory, relative to the repo root.             |
| `--terraform-bin` | `terraform`      | Terraform binary to invoke.                                       |

Version tracking:

- **production**: deploys the commit pointed at by the exact `--tag`
  given (not necessarily the currently checked-out commit), built in an
  isolated git worktree. The version is injected as `VITE_APP_VERSION`
  and, once deployed, recorded as a marker object in S3 (one per
  ever-released tag) plus a `_deploy-versions/current` object naming the
  live tag. Redeploying an already-deployed tag is a no-op; deploying an
  older tag than the current one is refused — both require `--force` to
  go through anyway (e.g. a deliberate rollback).
- **every other environment** (`sandbox`, ...): deploys whatever commit
  is currently checked out, versioned by its short SHA.

The `contact_api` Terraform module is optional: an environment without
it yet still builds and deploys, with the contact form shipping
disabled.

Example:

```sh
kitsu website deploy --env sandbox
kitsu website deploy --env production --tag website/v1.0.1
```

### Inspecting deployed versions

`kitsu website current --env <env>` and `kitsu website history --env <env>`
read the version tracking `deploy` writes (see above) — both take the
same `--infra-dir`/`--terraform-bin` flags, and only report anything
for an environment `deploy` actually records markers for (currently
just `production`).

| Command    | Output |
|------------|--------|
| `current`  | Just the tag currently live — nothing else, so it's easy to use in a script: `if [ "$(kitsu website current --env production)" = "$TAG" ]; then ...` |
| `history`  | Every version ever deployed, **most recently deployed first** (not highest version first — a rollback shows up as such, not as a gap), marking whichever one is currently live. |

```sh
$ kitsu website history --env production
website/v1.2.0           2026-09-10 14:32 UTC
website/v1.1.0           2026-09-05 09:10 UTC  (current)
website/v1.0.0           2026-09-01 18:00 UTC
```

(Here, `v1.2.0` was deployed and then rolled back to `v1.1.0` — `history`
makes that visible; `current` alone would just say `website/v1.1.0`.)

## Skills

`kitsu skills` manages a [Claude Code](https://claude.com/claude-code)
skill living at `skills/<skill>/` (a folder with a `SKILL.md`) in a
skills repository.

| Command                     | Does |
|------------------------------|------|
| `list`                       | Prints every skill's folder name, one per line, sorted. |
| `show <skill>`               | Prints the skill's `name` and `description`, read from its `SKILL.md` frontmatter. Errors if the frontmatter's `name` doesn't match the folder name — every other command addresses a skill by that folder name alone, so the two must agree. |
| `install <skill>`           | Copies `skills/<skill>/` into `~/.claude/skills/<skill>/` (overwritten if it already exists), so Claude Code picks it up. |
| `package <skill> [--output-dir <dir>]` | Zips `skills/<skill>/` into `<output-dir>/<skill>.zip` (default `.`), ready to upload to a new machine/Claude instance. |

All four default to cloning fresh from `--repo` (a GitHub `owner/repo`,
defaulting to `skills.repo` in the config files below, or
`samuelsulo/claude-skills` if neither is set) into a temp directory.
Pass `--local` to instead read `skills/<skill>/` from the current git
repository — e.g. while working inside a checkout of the skills repo
itself, to test a change before it's pushed (`--repo` and `--local`
are mutually exclusive).

```sh
kitsu skills list                                    # from the default/configured repo
kitsu skills show my-skill
kitsu skills install my-skill --repo someone/skills  # from a specific one
kitsu skills package my-skill --local --output-dir ./dist
```

## Configuration

Personal defaults that vary by user but not by project — not project
conventions, which stay as flags — live in two YAML files sharing the
same schema:

- **Global** (per-user): `$XDG_CONFIG_HOME/kitsu/config.yaml` (or the
  OS-appropriate equivalent of `~/.config/kitsu/config.yaml`).
- **Local** (per-project): `.kitsu.yaml` at the root of the current git
  repository — checked into it, so a value applies to everyone working
  on the project, not just you.

Where both set the same key, the local file wins. Every command that
reads a config value reads this merged, effective config.

```yaml
terraform:
  # Git URL of the Terraform module catalog used by 'catalog' (or pass
  # --catalog-repo explicitly on each command).
  catalog_repo: "git@github.com:<you>/terraform-aws-catalog.git"
  # IAM role ARN template used by 'scaffold environment', with %s
  # standing in for the AWS account id (or pass --role-arn-template).
  role_arn_template: "arn:aws:iam::%s:role/<YourAdminRole>"

skills:
  # GitHub "owner/repo" of the Claude Code skills repo used by
  # 'skills install'/'skills package' (or pass --repo explicitly).
  # Defaults to "samuelsulo/claude-skills" if omitted entirely.
  repo: "<you>/claude-skills"
```

Every value here can be overridden per invocation with the matching
flag; the config files only supply the default when the flag is
omitted. Most commands that need a value neither the flag nor either
config file provides fail with a message naming both ways to set it —
`skills.repo` is the one exception, falling back to
`samuelsulo/claude-skills` instead of erroring.

### `kitsu config`

`kitsu config` reads and edits the two files above directly, addressing
a value by its dot-separated key (e.g. `terraform.catalog_repo`, as
shown in the YAML above).

| Command                     | Does |
|------------------------------|------|
| `get <key>`                  | Prints `<key>`'s effective value (local, falling back to global) — or, with `--global`/`--local`, that one file's raw value. |
| `set <key> <value> (--global\|--local)` | Writes `<value>` for `<key>` into one file. `--global`/`--local` is required — the wrong one means either leaking a personal value into a shared repository, or a project setting silently only applying to you. |
| `unset <key> (--global\|--local)` | Clears `<key>` in one file (same scope requirement as `set`). |
| `show`                        | Prints the whole config as YAML — effective by default, or one file's raw content with `--global`/`--local`. |
| `path`                        | Prints the config file path(s): global first, then local if run inside a git repository. |
| `edit (--global\|--local)`    | Opens one file in `$EDITOR`, creating it first if it doesn't exist. |

```sh
kitsu config set --global skills.repo someone/claude-skills
kitsu config set --local terraform.catalog_repo git@github.com:acme/catalog.git  # this project only
kitsu config get terraform.catalog_repo   # effective value: local, falling back to global
kitsu config show
kitsu config edit --local
```
