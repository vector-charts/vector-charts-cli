# Agent instructions

## Commit messages

When creating git commits in this repo, always use a conventional prefix:

```
<type>: <short description>
```

### Allowed types

- `feat:` - new user-facing capability or command
- `fix:` / `bugfix:` - bug fix
- `chore:` - maintenance, cleanup, tooling, non-user-facing tweaks
- `docs:` - documentation only
- `refactor:` - code restructure with no behavior change
- `test:` - tests only
- `ci:` - CI / GitHub Actions
- `build:` - build system, packaging, nix, goreleaser
- `perf:` - performance improvement

### Rules

- Prefer one type that best matches the primary intent of the commit
- Description is lowercase, imperative, concise (no trailing period)
- Do not write unprefixed messages like `Initial implementation` or `Fix stuff`
- If amending or rewriting history, keep the same prefix style

### Examples

```
feat: add ftp sync command
bugfix: handle empty discovery responses
chore: tidy go.mod dependencies
ci: lint and cross-build on every push
docs: document nix develop workflow
```

## Comments and prose

Prefer no comments; when needed, keep them short and only explain non-obvious intent. Never comment that something is missing, removed, unfinished, or changed (no "not implemented", "formerly X", "removed Y", TODOs about absent behavior, etc.).

## ASCII only

Use basic ASCII only (bytes 0x00-0x7F) in code, comments, docs, commit messages, and user-facing strings. Do not use arrows, em/en dashes, ellipses, smart quotes, or other Unicode punctuation; write `->`, `-`, `...`, and straight quotes instead. Exception: existing TUI box-drawing characters in `internal/output` table rendering only; do not add new non-ASCII elsewhere.

## Public-repo confidentiality

This repository is public. Do not reference anything internal to the Vector Charts or Zydro platforms, their customers, users, or competitors, unless a developer explicitly confirms it is allowed. The only exception is essential public API surface the CLI must call or document (public endpoints, request/response fields the CLI uses, install URLs, and user-facing product names). Forbidden without explicit confirmation: internal service names, env names, hostnames, infra, architecture details, roadmaps, non-public APIs, customer or user identities, competitor names, private emails, and any non-essential company/org internals.

## Command documentation sync

When adding, removing, renaming, or changing CLI commands or flags, update the `## Command Reference` section in `README.md` in the same change (and Examples when representative usage should change). Keep that section complete and accurate for every public command and flag.

## README table of contents

Whenever you edit `README.md`, review the top link list (table of contents) and update it in the same change if sections were added, removed, renamed, or reordered. TOC links use the `https://github.com/vector-charts/vector-charts-cli#...` prefix and must match current top-level `##` section anchors.
