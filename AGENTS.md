# AGENTS Instructions

Homebase is a Go CLI for bootstrapping and maintaining dotfiles environments.
It builds one binary, `hb`, from `cmd/hb`, detects the active platform, seeds
TOML defaults, and delegates platform behavior to `internal/platform/<id>`.

## Start Here

- Read this file before changing the repository.
- Keep `cmd/hb/main.go` as a thin command router.
- Keep command behavior in `internal/*`.
- Keep one binary only: `hb`.
- Keep command names and flags stable unless the user asks for a breaking CLI
  change.
- Use `internal/run.Runner` for code that shells out.
- Use `internal/testutil` for shared fakes.
- Do not run package install, cleanup, sudo, or bootstrap side effects in
  tests.
- Config seeding must copy global config plus the active platform only.
- Ask a focused question before complex ambiguous changes.

## Repository Layout

- `cmd/hb/`: CLI router only
- `bootstrap/`: platform shell/bootstrap entrypoints
- `config/`: default runtime config copied to `~/.config/homebase`
- `internal/bootstrap/`: shared bootstrap flow and helpers
- `internal/install/`: shared install selector and planning helpers
- `internal/cleanup/`: shared cleanup helpers
- `internal/sync/`: `hb sync`
- `internal/config/`: TOML loading and config seeding
- `internal/platform/<id>/`: platform side effects, setup hooks, scanners,
  shell commands, defaults, and OS-specific paths
- `internal/ui/`: Bubble Tea selectors, prompts, spinner, styles
- `internal/run/`: command runner abstraction
- `internal/system/`: small system probes
- `internal/gitutil/`: bare Git and repo memory helpers
- `internal/testutil/`: shared test fakes

Keep owner-specific behavior with its owner. Move code into shared packages
only when it is policy-free or when at least two real callers need the same
contract.

## Side-Effect Rules

- Do not run `bootstrap/archlinux.sh`, `bootstrap/windows.ps1`, or
  `hb bootstrap` unless the user explicitly asks for live bootstrap side
  effects.
- Do not run `hb install`, package managers, setup hooks, or cleanup tasks
  without explicit user approval.
- Do not run commands that mutate `$HOME`, shell profiles, system services,
  package databases, registry entries, or dotfiles remotes unless required by
  the requested task.
- Prefer tests, fake runners, static inspection, and non-side-effecting command
  checks.
- For unattended examples, use explicit `--group` or `--task` selections, or
  `--all` with `--yes`.

## Implementation Guidance

- Prefer small package-level helpers over broad shared abstractions.
- Preserve existing TOML shapes unless the task requires a schema change.
- Add or update tests next to the package being changed.
- Use same-package tests for internal helpers when practical.
- Keep owner-specific behavior in owner package tests.
- Shared package tests should cover shared contracts and reusable helpers only.
- Keep scanner failures non-blocking unless the command itself cannot continue.
- Keep install and cleanup default selection semantics consistent across
  platforms: `default = true` only affects interactive selection.
- Keep Markdown and CLI examples factual and verified against source.

## Platform Notes

- Arch install owns `pacman`, AUR, and Arch setup hooks under
  `internal/platform/archlinux`.
- Arch cleanup owns pacman cache, AUR cache, orphan package, journal, npm cache,
  and thumbnail cleanup behavior.
- Windows install owns WinGet, Scoop, PowerShell modules, registry/profile setup
  features, and Windows package state scanning under
  `internal/platform/windows`.
- Windows cleanup owns temp files, Recycle Bin, WinGet cache, Scoop cache, npm
  cache, and thumbnail cleanup behavior.
- Shared selector state, inspection text rendering, scrolling, defaults, and
  fallback numbered selection belong in `internal/ui`.

## Docs

Update README, defaults, and affected skill docs together when changing:

- Config schema or default values
- CLI flags or command behavior
- Bootstrap, install, cleanup, or sync behavior
- Verification workflow

## Build and Test

Before finishing code changes, run:

```bash
make check
make build
```

For README or Markdown changes, also run:

```bash
make lint
```

For command routing changes, run:

```bash
make smoke
```
