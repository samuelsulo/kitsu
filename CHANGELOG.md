# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- Project scaffold: Go module `github.com/samuelsulo/kitsu`, Cobra-based
  command tree (`cmd/kitsu`, `internal/cli`), and a `version` command.
- `Makefile` with `build`, `install`, `run`, `test`, `fmt-check`, `lint`
  and `clean` targets; build-time version/commit/date injected via
  `-ldflags`.
- `CLAUDE.md` documenting project rules (language, documentation, README
  maintenance, changelog/versioning, commit conventions).
- `hooks install` command: points `core.hooksPath` at the repository's
  tracked hooks directory (`--dir`, default `.githooks`) and makes every
  hook file in it executable. Replaces the standalone
  `scripts/install-hooks.sh` used across projects.
- `terraform` command group: `init`, `validate`, `fmt`, `fmt-check`,
  `plan`, `show-plan`, `apply`, `apply-target`, `apply-auto`,
  `plan-destroy`, `refresh`, `destroy` and `destroy-target`, wrapping
  Terraform with the `<infra-dir>/live` + `<infra-dir>/environments/<env>`
  directory convention (`--env`, `--infra-dir`, `--terraform-bin`).
  Replaces the "TERRAFORM WORKFLOW" section of the per-project Terraform
  `Makefile` used across projects. `destroy`/`destroy-target` ask for
  confirmation before running Terraform's own (same double-confirmation
  as the original Makefile, since neither passes `-auto-approve`).

### Changed

- `.githooks/commit-msg`: commit scope is now validated against kitsu's
  own commands (`internal/cli/<scope>.go`) instead of the inherited
  `skills/<scope>` convention from a different project template.
