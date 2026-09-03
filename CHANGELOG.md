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

### Changed

- `.githooks/commit-msg`: commit scope is now validated against kitsu's
  own commands (`internal/cli/<scope>.go`) instead of the inherited
  `skills/<scope>` convention from a different project template.
