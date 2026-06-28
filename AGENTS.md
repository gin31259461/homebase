# AGENTS Instructions

Homebase is a Go CLI for bootstrapping and maintaining dotfiles environments.
It builds one binary, `hb`, from `cmd/hb`, detects the active platform, seeds
TOML defaults, and dispatches commands to platform-specific code.

## AI Guidance

- Read this file before making changes in the repository
- Keep `cmd/hb/main.go` small; command behavior belongs in `internal/*`
- Keep one binary only: `hb`
- Keep command names and flags stable unless the user explicitly asks for a
  breaking CLI change
- Use `internal/run.Runner` for code that shells out
- Use `internal/testutil` for shared fakes
- Do not run package install, cleanup, sudo, or bootstrap side effects in tests
- Config seeding must copy global config plus the active platform only
- Update README, defaults, and affected skill docs together when changing config
  schema/defaults, CLI flags, bootstrap/install/cleanup/sync behavior, or the
  verification workflow
- Ask focused questions before complex ambiguous changes

## Required Skills

- For install/cleanup UI or platform parity work, read
  `.agents/skills/homebase-platform-ui/SKILL.md`
- For `bootstrap/windows.ps1`, remote install snippets, or PowerShell 5.1
  `irm/iwr | iex` compatibility work, read
  `.agents/skills/powershell-remote-bootstrap/SKILL.md`

## Boundaries

- `cmd/hb/`: thin CLI router only
- `bootstrap/`: platform shell/bootstrap entrypoints
- `internal/bootstrap/`: shared bootstrap flow and helpers
- `internal/install/`: shared install selector/planning helpers
- `internal/cleanup/`: shared cleanup helpers
- `internal/sync/`: `hb sync`
- `internal/config/`: TOML loading and config seeding
- `internal/platform/<id>/`: platform-owned install, cleanup, bootstrap, setup,
  shell commands, scanners, defaults, and OS-specific paths
- `internal/ui/`: Bubble Tea selector, prompts, spinner, styles
- `internal/run/`: command runner abstraction
- `internal/system/`: small system probes such as command detection
- `internal/gitutil/`: bare Git and repo memory helpers
- `internal/testutil/`: shared test fakes
- `config/`: default runtime config copied to `~/.config/homebase`

Keep ownership concrete:

- Platform defaults, side effects, commands, file layouts, and policy stay with
  the package/platform that owns them
- Keep platform bootstrap divergences in `internal/platform/<id>/bootstrap.go`
  and platform setup hooks under `internal/platform/<id>`
- Shared packages should provide policy-free helpers or contracts needed by at
  least two real callers
- Interactive install and cleanup state belongs near the platform command flow;
  shared selector behavior belongs in `internal/ui`

## Safety Boundaries

- Do not run `bootstrap/archlinux.sh`, `bootstrap/windows.ps1`, or
  `hb bootstrap` unless the user explicitly asks for live bootstrap side effects
- Do not run `hb install`, package managers, setup hooks, or cleanup tasks
  without explicit user approval
- Do not run commands that mutate `$HOME`, shell profiles, system services,
  package databases, registry entries, or dotfiles remotes unless required by
  the requested task
- Prefer tests, fake runners, static inspection, and non-side-effecting command
  checks for verification
- For unattended examples, use explicit `--group`/`--task` selections or
  `--all` with `--yes`

## Style

- Prefer small package-level helpers over broad shared abstractions
- Extract shared code only when it is policy-free or when at least two real
  callers need the same explicit contract
- Split files when they begin mixing orchestration, item/status construction,
  scanning, task execution, shell helpers, and tests
- Put unit tests next to the package they test
- Use same-package tests for internal helpers when practical
- Keep owner-specific behavior in owner package tests
- Shared package tests should cover only shared contracts and reusable helpers
- Preserve existing TOML shapes unless the task requires a schema change
- Keep markdown and CLI examples factual and verified against source

## Build/Test

Before finishing code changes, run:

```bash
make check
make build
```

`make check` runs:

```bash
gofmt -w cmd internal
go test ./...
go vet ./...
```

For README or Markdown changes, also run:

```bash
make lint
```

`make lint` checks:

```bash
markdownlint-cli2 README.md AGENTS.md \
  .agents/skills/homebase-platform-ui/SKILL.md \
  .agents/skills/powershell-remote-bootstrap/SKILL.md
```

Smoke-check command routing changes:

```bash
make smoke
```

`make smoke` builds `hb`, runs `hb help`, and runs:

```bash
hb install --group __none__ --yes --no-setup
```
