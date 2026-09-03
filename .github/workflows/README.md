# Workflows

GitHub Actions workflows for this repository.

## Available workflows

| Workflow | Trigger | What it does |
|---|---|---|
| [`release.yaml`](./release.yaml) | Push of a tag matching `v*.*.*` | Runs [GoReleaser](https://goreleaser.com/) (see [`.goreleaser.yaml`](../../.goreleaser.yaml)) to cross-compile `kitsu` for linux/darwin × amd64/arm64 and publish the resulting archives + checksums as a GitHub Release, named after the tag. |

## Adding a Workflow

A new `.yaml` file in this folder is automatically picked up by GitHub
Actions, with no further configuration. When adding one, update this
README with the workflow's name, trigger, and what it checks/does.
