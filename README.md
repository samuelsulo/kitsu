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

## Commands

| Command         | Description                                              |
|-----------------|-----------------------------------------------------------|
| `version`       | Print kitsu's version info.                              |
| `hooks install` | Point git at the repository's tracked hooks (`--dir`, default `.githooks`) and make them executable. |
| `terraform ...` | Run Terraform against the `infrastructure/<env>` convention. See [Terraform workflow](#terraform-workflow). |

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

Example:

```sh
kitsu terraform plan --env production
kitsu terraform apply --env production
```
