# AGENTS instructions

Homebase is a Go CLI for bootstrapping and maintaining dotfiles environments.
It builds one binary, `hb`, from `cmd/hb`, detects the active platform, and
dispatches commands to platform-specific code.

## Project layout

- `cmd/hb/`: thin CLI router only
- `bootstrap/`: platform-specific shell/bootstrap entrypoints
- `internal/bootstrap/`: `hb bootstrap`
- `internal/install/`: `hb install` and package planning
- `internal/cleanup/`: `hb cleanup`
- `internal/sync/`: `hb sync`
- `internal/config/`: TOML loading and config seeding
- `internal/platform/`: platform registry, detection, and implementations
- `internal/ui/`: Bubble Tea selector, prompts, spinner, styles
- `internal/run/`: command runner abstraction
- `internal/system/`: OS, package, service, and user helpers
- `internal/gitutil/`: bare git and repo memory helpers
- `internal/setup/`: post-install setup routines
- `internal/testutil/`: test fakes
- `config/`: default runtime config copied to `~/.config/homebase`
- `config/platforms/<id>/`: platform-specific runtime config defaults

## Development rules

- Keep `cmd/hb/main.go` small; command behavior belongs in `internal/*`
- Keep one binary only: `hb`
- Keep command names stable; platform differences belong behind detection
- Preserve current CLI flags unless the user asks for a breaking change
- Prefer small package-level helpers over large cross-package abstractions
- Keep config schema changes deliberate and update defaults plus README together
- Config seeding should copy global config plus the active platform only
- Keep platform-specific behavior under `internal/platform/<id>` when possible
- Use the runner abstraction for code that shells out
- Do not run package install, cleanup, sudo, or bootstrap side effects in tests
- For install/cleanup UI or platform parity work, read:
  `.agents/skills/homebase-platform-ui/SKILL.md`
- Do not let any file have a large amount of code, consider separating of concern
- Complex tasks should ask the user questions to confirm the detailed requirements

## Project subagents

Project-local subagents live in `.codex/agents/`. Use them deliberately for
focused work:

- `golang-pro`: Go implementation in `cmd/hb` and `internal/*`
- `cli-developer`: command flags, CLI UX, automation paths, and shell-facing
  workflows
- `powershell-5.1-expert`: Windows bootstrap compatibility and remote
  `irm/iwr | iex` behavior
- `powershell-7-expert`: PowerShell 7 post-bootstrap automation and future
  Windows platform scripting
- `test-automator`: targeted regression tests and fake-runner coverage
- `reviewer`: PR-style correctness, regression, security, and test review
- `code-mapper`: execution-flow and ownership mapping before broad changes
- `documentation-engineer`: README, config, and operator workflow docs tied to
  repository reality

## Testing

Put unit tests next to the package they test:

```text
internal/install/plan_test.go
internal/config/config_test.go
```

Prefer same-package tests for internal helpers:

```go
package install
```

Use `internal/testutil` for shared fakes.

## Verification

Before finishing code changes, run:

```bash
gofmt -w cmd internal
go test ./...
go vet ./...
go build -o ~/.local/bin/hb ./cmd/hb
```

For README or Markdown changes, also run:

```bash
markdownlint-cli2 README.md
```

Smoke-check the CLI when command routing changes:

```bash
hb help
hb install --group __none__ --yes --no-setup
```
