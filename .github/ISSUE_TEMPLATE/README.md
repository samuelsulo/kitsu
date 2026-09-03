# Issue Template

Templates for opening GitHub issues during the development of a client
project, to track work uniformly from the high-level objective down to
the individual bug.

## Available Templates

| Template | File | Use | Label |
|---|---|---|---|
| 🚀 Epic | [`epic.md`](./epic.md) | High-level feature, to be broken down into User Story, Technical Story or Spike | `epic` |
| 💡 User Story | [`user-story.md`](./user-story.md) | Feature from the end user's point of view | `user-story` |
| ⚙️ Technical Story | [`technical-story.md`](./technical-story.md) | Feature or task from a technical point of view (implementation, configuration, infrastructure) | `technical-story` |
| 🔍 Spike | [`spike.md`](./spike.md) | Research or technical exploration activity with no immediate deliverable (PoC, tool evaluation) | `spike` |
| 🐛 Bug Report | [`bug-report.md`](./bug-report.md) | Malfunction or unexpected behavior | `bug` |

## Hierarchy

```
Epic
├── User Story        (value for the end user)
├── Technical Story   (technical work not directly visible to the user)
└── Spike             (preliminary research, opens subsequent Story/Task items)

Bug Report            (independent, not part of an Epic)
```

Every Epic lists in its own issue the Story and Spike items that compose
it ("Breakdown" table); every Story/Spike references, when relevant, the
Epic it belongs to under "References".

## Conventions

- Issue title with the prefix already set by the template (e.g.
  `[Epic] `, `[Story] `, `[Bug] `): do not remove it, it is there to
  recognize the issue type at a glance in the list.
- Every template already sets `labels` and `type`: do not add alternative
  labels for the same concept (e.g. do not use `feature` alongside
  `user-story`).
- Acceptance criteria and the Definition of Done must be filled in before
  starting the work, not after the fact: they are the basis for closing
  the issue.
- Sections in square brackets (e.g. `[Describe...]`) are placeholders to
  be replaced, not text to leave in the published issue.

## Adding a Template

A new `.md` file in this folder with the YAML front matter (`name`,
`type`, `about`, `title`, `labels`, `assignees`) automatically appears
among the templates offered when creating an issue, with no further
configuration needed.
