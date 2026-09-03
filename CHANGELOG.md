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
- `terraform` state management and inspection commands: `import`,
  `state list`/`show`/`rm`, `unlock`, `taint`, `untaint`, `console`,
  `providers`, `version` and `upgrade`, ported from the "STATE
  MANAGEMENT" and "INSPECTION" sections of the same Makefile.
  `state rm` and `unlock` ask for confirmation, matching the originals.
- `terraform docs`, `output`, `output-json` and `clean` commands, ported
  from the "DOCS", "OUTPUT" and "CLEANUP" sections of the same Makefile.
  `docs` runs `terraform-docs` (`--terraform-docs-bin`, default
  `terraform-docs`) against every module directory under
  `<infra-dir>/modules/*/*`.
- `terraform scaffold environment`/`scaffold module` and
  `terraform catalog list`/`versions`/`vendor`, completing the port of
  the "SCAFFOLDING" section of the same Makefile. `catalog` nests
  `list`/`versions`/`vendor` under one group (rather than flat
  `catalog-modules`/`catalog-versions`/`vendor-module`, since all three
  operate on the same catalog repository), and `vendor` takes module and
  version as two separate arguments instead of one concatenated
  `<module>/vX.Y.Z` ref.
- Per-user config file (`$XDG_CONFIG_HOME/kitsu/config.yaml`,
  `internal/config`) for personal defaults that don't belong as project
  conventions: `terraform.catalog_repo` and
  `terraform.role_arn_template`, each overridable per invocation with
  `--catalog-repo`/`--role-arn-template`.
- `terraform bootstrap-backend` command: creates (once per AWS account,
  idempotently) the S3 bucket for the Terraform state shared by every
  project in that account — versioning, encryption, public access
  block, TLS-only policy, lifecycle on noncurrent versions. Ported from
  the standalone `scripts/bootstrap-terraform-backend.sh` used across
  projects; always operates on the account of the currently active AWS
  credentials, never an account id passed by hand, matching the
  original.
- `website deploy` command (`internal/website`): builds the website and
  syncs it to the S3 bucket + CloudFront distribution of the given
  environment, read from that environment's Terraform state. Ported
  from the standalone `scripts/deploy-website.sh` used across projects,
  with the same version-tracking rules: production deploys an explicit
  `website/vX.Y.Z` tag in an isolated git worktree, tracked via S3
  markers, refusing to redeploy an already-deployed tag or downgrade to
  an older one without `--force`; every other environment deploys the
  currently checked-out commit, versioned by its short SHA. The
  `contact_api` Terraform module stays optional, matching the original.

### Changed

- `.githooks/commit-msg`: commit scope is now validated against kitsu's
  own commands (`internal/cli/<scope>.go`) instead of the inherited
  `skills/<scope>` convention from a different project template.
