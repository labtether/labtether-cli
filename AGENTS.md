# AGENTS.md — LabTether CLI

## Shared rules

For Astra and Opus 5.5: edit `AGENTS.md`; keep `CLAUDE.md -> AGENTS.md`.
Read `../AGENTS.md` once if available, or from a worktree use
`/Users/michael/Development/LabTether/AGENTS.md`. Read only task-relevant docs.
Finish scoped work with focused checks; make routine reversible choices yourself.
Use current manifests, preserve unrelated dirty work, and reply in short plain words.

- Reuse this repo; do not clone or copy it. Worktrees belong under
  `/Users/michael/.codex/worktrees/LabTether/`; temp files in task `work/` or
  `mktemp -d`. Create no repos, worktrees, caches or temp folders directly in
  `/Users/michael` or `/Users/michael/Development`. Clean only your own temp files.
- One broad build/suite at a time; require 100 GB free for heavy builds. Reuse
  caches. Removing user files, dirty work, repos, branches or worktrees needs
  an exact preview and explicit approval. Preview generated-cache cleanup.
- VM 102 / `UntrustedVM` is excluded and untouched: no enumeration, queries,
  inspection, backup, operations or QA evidence.
- Signing material stays local outside repos: never expose, list, copy, stage or
  upload it. Release signing/notarization needs explicit authorization; preserve owner
  signing pauses. Read workspace release rules; publish only verified distributables.
- Prefer scoped disposable credentials or an existing session; never rotate the
  owner password if either is available. Before temporary auth changes, install
  restore/cleanup traps, save the exact state without logging secrets, then
  verify restoration of its hash/timestamp and session baseline.
- In `zsh`, use `rc` or `exit_code`, never reserved `status`.

## File size and checks

- Hard limit: 500 code lines per handwritten source/test/script or executable
  CI/build/config file, using pinned `cloc 2.10` (excludes blanks/comment-only
  lines). Only genuine generated/vendor code is exempt. No legacy exceptions,
  minifying, numbered chunks or moving code into data; split by responsibility.
- Run `python3 scripts/ci/check-line-limit.py` here. Existing violations remain
  open until it passes. Builds do not prove live behavior; backup or
  verification does not prove restore. Report what actually passed.
- Add or keep a test only for a named real fault or important user promise.
  Test the real owner with the smallest useful check. Skip getter-only,
  duplicate, implementation-mirror and test-count padding. Broaden only for
  changed shared behavior or a concrete unresolved risk. Docs-only work needs
  path, link, diff and line-limit checks, not code tests or builds unless a
  required branch gate needs them.
- Before starting or rerunning remote CI, name the changed behavior and needed
  proof. Keep one job per distinct risk; avoid duplicate runs for the same
  commit. Cancel superseded runs, use path filters and realistic timeouts. Use
  paid platform runners or broad matrices only for relevant behavior. Preserve
  required security, release, signing and live checks that protect user promises.

## Repo guide

Go/Cobra command-line client for the Hub API.

- Entry point: `main.go`; commands and output: `cmd/`; HTTP client:
  `internal/client/`.
- Keep table and JSON output stable. Preserve meaningful exit codes and Hub
  authentication/error behavior.
- Credentials must not appear in command arguments or logs. `--api-key` is
  deliberately rejected; use a protected file or a local environment variable.
- From this repo, run `go test ./cmd ./internal/client`, narrowing with `-run`
  for a small change. Use `go.mod` for the Go version.
- CLI is part of the workspace prerequisite release stage. Use the staged
  release script for coordinated tags.
