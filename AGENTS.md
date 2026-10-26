# AGENTS.md — LabTether CLI

Go and Cobra command-line client for the Hub API.

## Read first

Codex (GPT-6 Astra) and Claude Code (Opus 5.5) both load this file. Edit
`AGENTS.md` only; `CLAUDE.md` is a relative symlink to it.

This file covers routine work in this repo. The workspace guide,
`../AGENTS.md` (from a worktree:
`$HOME/Development/LabTether/AGENTS.md`), has the full rules, lab
machines and release stages. Read it before release, signing, QA, lab or
cross-repo work. If the two disagree, the workspace guide wins; fix the stale
line here.

## How to work

- Carry the task to a checked result. Make routine, reversible choices
  yourself. Stop to ask only for a decision that is the owner's or a boundary
  below, and keep doing independent work while you wait.
- Check the branch and dirty files before editing; leave unrelated work alone.
  Reuse this checkout. Worktrees go under
  `$HOME/.codex/worktrees/LabTether/`; throwaway files go in the
  task's `work/` folder or `mktemp -d`, never directly in `$HOME` or
  `$HOME/Development`. Clean up only what you created.
- Run one broad build or suite at a time and reuse existing caches.
- Reply in short, plain words: what changed, whether it worked, what is next.
- User-facing text follows the workspace "No AI Slop" rules: keep normal
  homelab words (node, VM, pool, snapshot, uptime), cut hype, filler and
  internal engineering words, and never show raw IDs or bare status codes.

## Boundaries

- Run VMs, Docker stacks, simulator QA, installed-agent tests and long suites
  on the lab test machines named in the workspace guide, never on the
  development laptop. One test file, lint or one compile is fine locally.
- Machines the workspace guide marks as no-touch are out of scope: do not
  query, operate on or cite them.
- Signing material stays in its local-only folder outside every repository.
  Never list, copy, print, stage or upload any of it. Signing and notarization
  need the owner's go-ahead for each release.
- Removing repos, branches, worktrees, user files or dirty work needs an exact
  preview and the owner's approval.
- Prefer a scoped disposable token or an existing session to changing auth.
  Before any temporary auth change, set restore traps and confirm the original
  state afterwards. Never rotate the owner password when a safer route exists.
- In `zsh`, `status` is reserved; name exit codes `rc` or `exit_code`.
- Keep personal details out of files, test data and commit messages: real
  names, home folders, lab host names, home IP addresses, personal emails and
  tailnet names. Use neutral examples such as `/Users/demo`, `nas01` and
  `user@example.com`. `python3 scripts/ci/check-private-info.py` and CI block
  them; the workspace git hooks also block the exact private values.

## Checks

- Hard limit: 500 code lines per handwritten file (source, tests, scripts,
  executable CI/build/config), counted by pinned `cloc 2.10`. Run
  `python3 scripts/ci/check-line-limit.py` here. Split by responsibility; do
  not minify, number chunks or move code into data. Existing violations stay
  open until the checker passes.
- Add or keep a test only when it catches a real fault or guards a user
  promise, and name that risk. Skip getter-only, duplicate and
  implementation-mirror tests. Docs-only edits need path, link, diff and
  line-limit checks, not builds.
- Before starting or rerunning remote CI, check the remaining minutes and
  spend cap and name what the run proves. Do not rerun for another green.
- Report what actually passed. A build does not prove runtime behavior, a mock
  does not prove a live result, and a backup is not a restore.

## This repo

- Entry point: `main.go`. Commands and output: `cmd/`. HTTP client:
  `internal/client/`.
- Keep table and JSON output stable, exit codes meaningful, and Hub auth and
  error handling unchanged.
- Credentials never appear in arguments or logs. `--api-key` is rejected on
  purpose; use a protected file or a local environment variable.
- Run `go test ./cmd ./internal/client`, narrowing with `-run` for small
  changes. `go.mod` sets the Go version.
- The CLI joins the workspace prerequisite release stage; tag it only through
  the staged release script.
