# Workflows

GitHub Actions workflows for automatic checks on the Terraform
infrastructure. Currently this folder does not contain any workflow yet:
the checks described in [`CLAUDE.md` §5](../../CLAUDE.md#5-terraform-rules)
(`terraform validate` and a clean `plan` before every `apply`) are today
run manually through the [`Makefile`](../../infrastructure/Makefile)
targets (`make validate`, `make fmt-check`, `make plan`, ...).

## Allowed Scope

A workflow introduced in this folder can:

- Run `make fmt-check` and `make validate ENV=sandbox` on every pull
  request, as a pre-merge check.
- Run `make plan ENV=sandbox` to show the predicted execution, without
  applying it.

A workflow must **never**:

- Run `make apply` (or `apply-auto`) on `environments/production`: every
  `apply` in production requires a manually reviewed `plan`, not only by
  the AI assistant (see [`CLAUDE.md` §5](../../CLAUDE.md#5-terraform-rules)
  and [§9](../../CLAUDE.md#9-changes-that-require-manual-review)).
- Contain AWS credentials in plain text in the workflow file: credentials
  must be read from secrets configured on GitHub, never committed.

Introducing a workflow that automates `apply` on `production`, even
behind manual approval within the workflow itself, is a change to the
deployment flow and must be flagged and agreed upon before being
implemented (see
[`CLAUDE.md` §9](../../CLAUDE.md#9-changes-that-require-manual-review)).

## Adding a Workflow

A new `.yaml` file in this folder is automatically picked up by GitHub
Actions, with no further configuration. When adding one, update this
README with the workflow's name, trigger (e.g. `pull_request`) and what
it checks.
