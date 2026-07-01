# AGENTS.md

## Project Overview

Homebase is a Go CLI for bootstrapping and maintaining personal dotfiles
environments. It builds one binary, `hb`, from `cmd/hb`, detects the active
platform, seeds TOML defaults, and delegates platform behavior to
`internal/platform/<id>`.

Supported platforms:

- `archlinux`: Arch Linux and Manjaro, detected through `/etc/arch-release` or
  `/etc/manjaro-release`.
- `windows`: Windows, detected through `runtime.GOOS == "windows"`.

Key dependencies:

- Go `1.24.2`
- Bubble Tea, Bubbles, and Lip Gloss for interactive selectors and terminal UI
- `github.com/pelletier/go-toml/v2` for TOML config loading

## Repository Layout

- `cmd/hb/`: CLI command router only.
- `bootstrap/`: live platform bootstrap entrypoints.
- `config/`: default runtime config copied into `~/.config/homebase`.
- `config/homebase.toml`: global platform selection and aliases.
- `config/platforms/<id>/config.toml`: platform dotfiles and package-manager
  defaults.
- `config/platforms/<id>/packages.d/*.toml`: install groups.
- `config/platforms/<id>/cleanup.toml`: cleanup task metadata.
- `config/platforms/<id>/sync.toml`: dotfile paths staged by `hb sync`.
- `internal/bootstrap/`: shared bootstrap flow for Arch-like platforms.
- `internal/install/`: shared install selector and planning helpers.
- `internal/cleanup/`: shared cleanup helpers.
- `internal/sync/`: `hb sync` implementation.
- `internal/config/`: config seeding, loading, expansion, and TOML parsing.
- `internal/platform/archlinux/`: Arch package, setup, cleanup, scanner, and
  bootstrap behavior.
- `internal/platform/windows/`: Windows package, setup, cleanup, scanner, and
  bootstrap behavior.
- `internal/ui/`: Bubble Tea selectors, fallback numbered selection, prompts,
  spinner, styles, and text rendering.
- `internal/run/`: command runner abstraction and command display redaction.
- `internal/system/`: small system probes.
- `internal/gitutil/`: bare Git and dotfiles repo memory helpers.
- `internal/testutil/`: shared fake runner for tests.

Keep owner-specific behavior with its owner. Move code into shared packages
only when it is policy-free or when at least two real callers need the same
contract.

## Core Commands

```text
hb bootstrap [--yes] [--repo <repo>] [--install]
hb install   [--group <key>] [--all] [--yes] [--no-setup]
hb cleanup   [--task <key>] [--all] [--yes]
hb sync      [-m <message>] [--no-push]
hb config init [-f|--force]
```

`cmd/hb/main.go` should stay a thin router. Add command behavior in `internal/*`
or the owning platform package.

## Side-Effect Rules

- Do not run `bootstrap/archlinux.sh`, `bootstrap/windows.ps1`, `hb bootstrap`,
  `hb install`, package managers, setup hooks, or cleanup tasks unless the user
  explicitly asks for live workstation changes.
- Do not run commands that mutate `$HOME`, shell profiles, system services,
  package databases, registry entries, or dotfiles remotes unless required by
  the user request.
- Prefer tests, fake runners, static inspection, and non-side-effecting command
  checks.
- Use `internal/run.Runner` for code that shells out.
- Use `internal/testutil` for shared fakes.
- Tests must not install packages, clean real caches, call `sudo`, bootstrap
  live dotfiles, change system services, or mutate real user registry/profile
  state.
- For unattended examples, use explicit `--group` or `--task` selections, or
  `--all --yes`.

## Config Rules

- Config seeding must copy the global config plus the active platform only.
- Preserve existing TOML shapes unless the task explicitly requires a schema
  change.
- `config.EnsureGlobal` copies `config/homebase.toml`.
- `config.EnsureForPlatform(platformID, force)` copies global config and
  `config/platforms/<platformID>`.
- `config.SourceRoot()` reads defaults from `HOMEBASE_DIR` when set, otherwise
  `~/.local/lib/homebase`.
- `default = true` in install groups or cleanup tasks affects interactive
  selection only. It must not change explicit `--group`, `--task`, or `--all`
  semantics.
- Keep scanner failures non-blocking unless the command cannot continue.

## Platform Ownership

- Arch install owns `pacman`, AUR helper handling, install state scanning, and
  Arch setup hooks under `internal/platform/archlinux`.
- Arch cleanup owns pacman cache, AUR cache, orphan packages, journal cleanup,
  npm cache, and thumbnail cleanup behavior.
- Windows install owns WinGet, Scoop, PowerShell modules, registry/profile setup
  features, and Windows package state scanning under
  `internal/platform/windows`.
- Windows cleanup owns temp files, Recycle Bin, WinGet cache, Scoop cache, npm
  cache, and thumbnail cleanup behavior.
- Shared selector state, inspection rendering, scrolling, defaults, and
  fallback numbered selection belong in `internal/ui`.

## Development Workflow

Use the repo's `Makefile`:

```bash
make fmt    # gofmt -w cmd internal
make test   # go test ./...
make vet    # go vet ./...
make check  # fmt, test, vet
make build  # builds hb to ~/.local/bin/hb or %USERPROFILE%/.local/bin/hb.exe
make lint   # markdownlint README.md
make smoke  # build, hb help, empty non-mutating install selection
```

`make smoke` runs:

```bash
hb help
hb install --group __none__ --yes --no-setup
```

The install smoke command relies on unknown group filtering and should not
install packages.

## Testing Instructions

- Run `make check` before finishing code changes.
- Run `make build` before finishing code changes.
- For README or Markdown changes, also run `make lint`.
- For command routing changes, also run `make smoke`.
- Run focused tests with `go test ./internal/<package>` or
  `go test ./internal/platform/<platform>`.
- Add or update tests next to changed code.
- Prefer same-package tests for internal helpers when practical.
- Shared package tests should cover shared contracts and reusable helpers only.
- Owner-specific behavior belongs in owner package tests.

## Code Style

- Keep one binary only: `hb`.
- Keep command names and flags stable unless the user asks for a breaking CLI
  change.
- Prefer small package-level helpers over broad shared abstractions.
- Keep `cmd/hb/main.go` as routing and usage text only.
- Keep command behavior in `internal/*`.
- Keep Markdown and CLI examples factual and verified against source.
- Use `gofmt`; there is no separate Go formatter configuration.

## Security And Safety Notes

- `internal/run.ExecRunner` prints commands before execution and redacts common
  password-like arguments. Preserve or extend redaction when adding sensitive
  command arguments.
- Bootstrap may clone dotfiles over HTTPS when GitHub SSH `HEAD` is not
  available, then reset the origin to the configured SSH remote.
- Cleanup tasks can remove files and caches. Keep task metadata explicit in
  `cleanup.toml` and require confirmation unless `--yes` is supplied.
- Arch setup hooks can write sudo-owned files and enable services. Windows setup
  features can modify user PATH, profile links, and HKCU registry keys. Keep
  this behavior platform-owned and covered by fakes in tests.

## Documentation Updates

Update README, defaults, and affected skill docs together when changing:

- Config schema or default values
- CLI flags or command behavior
- Bootstrap, install, cleanup, or sync behavior
- Verification workflow

Do not add license, contributing, or changelog content to `README.md`; those
belong in dedicated files.
