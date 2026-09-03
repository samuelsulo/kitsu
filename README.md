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

More commands will be added here as they're implemented.
