# AGENTS instructions

Homebase is a Go CLI for bootstrapping and maintaining dotfiles environments.
It builds one binary, `hb`, from `cmd/hb`, detects the active platform, and
dispatches commands to platform-specific code.

## Layout

- `cmd/hb/`: thin CLI router only
- `bootstrap/`: platform shell/bootstrap entrypoints
- `internal/bootstrap/`: shared bootstrap flow and helpers
- `internal/install/`, `internal/cleanup/`: shared selector/planning helpers
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

## Rules

- Keep `cmd/hb/main.go` small; command behavior belongs in `internal/*`
- Keep one binary only: `hb`
- Keep command names and flags stable unless the user asks for a breaking change
- Use the runner abstraction for code that shells out
- Do not run package install, cleanup, sudo, or bootstrap side effects in tests
- Config seeding should copy global config plus the active platform only
- Update README, defaults, and affected skill docs together when changing config
  schema/defaults, CLI flags, bootstrap/install/cleanup/sync behavior, or
  verification workflow
- Prefer small package-level helpers over broad shared abstractions
- Keep ownership concrete: defaults, side effects, commands, file layouts, and
  platform policy stay with the package/platform that owns them
- Extract shared code only when it is policy-free or at least two real callers
  need the same explicit contract
- Keep platform bootstrap divergences in `internal/platform/<id>/bootstrap.go`
  and platform setup hooks under `internal/platform/<id>`
- Split files when they begin mixing orchestration, item/status construction,
  scanning, task execution, shell helpers, and tests
- Ask focused questions before complex ambiguous changes

## Required Skills

- For install/cleanup UI or platform parity work, read
  `.agents/skills/homebase-platform-ui/SKILL.md`
- For `bootstrap/windows.ps1`, remote install snippets, or PowerShell 5.1
  `irm/iwr | iex` compatibility work, read
  `.agents/skills/powershell-remote-bootstrap/SKILL.md`

## Subagents

Project-local subagents live in `.codex/agents/`:

- `golang-pro`: Go implementation in `cmd/hb` and `internal/*`
- `cli-developer`: command flags, CLI UX, automation, and shell workflows
- `powershell-5.1-expert`: Windows bootstrap and remote `irm/iwr | iex`
- `powershell-7-expert`: PowerShell 7 post-bootstrap automation
- `test-automator`: regression tests and fake-runner coverage
- `reviewer`: correctness, regression, security, and test review
- `code-mapper`: execution-flow and ownership mapping
- `documentation-engineer`: README, config, and operator workflow docs

## Testing

- Put unit tests next to the package they test, using same-package tests for
  internal helpers when practical
- Use `internal/testutil` for shared fakes
- Keep owner-specific behavior in owner package tests; shared package tests
  should cover only shared contracts and reusable helpers

## Verification

Before finishing code changes, run:

```bash
make check
make build
```

For README or Markdown changes, also run:

```bash
make lint
```

Smoke-check the CLI when command routing changes:

```bash
make smoke
```
