# CLAUDE.md

Project rules for `kitsu`, a personal automation CLI written in Go. These
rules apply to every change in this repository, human- or AI-authored.

## 1. Language

Everything in this repository is written in English: code, identifiers,
comments, commit messages, documentation, issue/PR templates — no
exceptions. This keeps the project consistent and usable outside a single
native-language context.

## 2. Documentation

Every unit of behavior is documented, not just the notable ones:

- Every new command (and subcommand) gets a `Short`/`Long` description in
  its Cobra definition, plus a usage entry in `README.md`.
- Every exported Go identifier (package, type, func, const) gets a doc
  comment following standard Go conventions.
- Non-obvious internal logic gets an inline comment explaining *why*, not
  just *what*.

Undocumented code is treated as incomplete, not as a follow-up task.

## 3. README

`README.md` is the entry point for using and understanding this project.
It must be updated in the same change that introduces the thing it
describes — a new command, a changed flag, a new build/install step, a
changed requirement. A change is not done until `README.md` reflects it.

## 4. Changelog & Versioning

This project follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
and [Semantic Versioning](https://semver.org/) (`vMAJOR.MINOR.PATCH`).

- `CHANGELOG.md` has an `[Unreleased]` section at the top, under
  `Added`/`Changed`/`Fixed`/`Removed`/etc. headings as needed.
- Every change worth telling a user about gets an entry under
  `[Unreleased]` in the same commit that makes the change — not batched
  later.
- Cutting a release means: move `[Unreleased]`'s content under a new
  `## [vX.Y.Z] - YYYY-MM-DD` section, leave a fresh empty `[Unreleased]`
  on top, and tag the resulting commit `vX.Y.Z` (annotated git tag,
  matching the changelog heading exactly).
- Version bump follows SemVer: breaking CLI/flag/behavior changes →
  MAJOR (or MINOR pre-1.0.0), new backward-compatible commands/flags →
  MINOR, fixes/docs/internal cleanup → PATCH.

## 5. Commits

Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/),
enforced by `.githooks/commit-msg`:

```
<type>(<scope>): <description>
```

Allowed types: `feat`, `fix`, `refactor`, `docs`, `chore`, `test`, `style`,
`perf`, `build`, `ci`, `revert`.

`<scope>` is either the name of the affected command — matching an
existing `internal/cli/<scope>.go` file, excluding `root.go` — or `repo`
for changes not tied to one specific command (e.g. this file, the
`Makefile`, CI config).
