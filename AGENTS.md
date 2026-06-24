# AGENTS instructions

Homebase is a Go CLI for bootstrapping and maintaining an Arch Linux dotfiles
environment. It builds one binary, `hb`, from `cmd/hb`.

## Project layout

- `cmd/hb/`: thin CLI router only
- `internal/bootstrap/`: `hb bootstrap`
- `internal/install/`: `hb install` and package planning
- `internal/cleanup/`: `hb cleanup`
- `internal/sync/`: `hb sync`
- `internal/config/`: TOML loading and config seeding
- `internal/ui/`: Bubble Tea selector, prompts, spinner, styles
- `internal/run/`: command runner abstraction
- `internal/system/`: Arch, pacman, systemd, and user helpers
- `internal/gitutil/`: bare git and repo memory helpers
- `internal/setup/`: post-install setup routines
- `internal/testutil/`: test fakes
- `config/`: default runtime config copied to `~/.config/homebase`

## Development rules

- Keep `cmd/hb/main.go` small; command behavior belongs in `internal/*`
- Keep one binary only: `hb`
- Preserve current CLI flags unless the user asks for a breaking change
- Prefer small package-level helpers over large cross-package abstractions
- Keep config schema changes deliberate and update defaults plus README together
- Use the runner abstraction for code that shells out
- Do not run package install, cleanup, sudo, or bootstrap side effects in tests

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
