---
name: homebase-platform-ui
description: Use when implementing or refactoring Homebase install and cleanup platform flows, especially interactive list item state, default selection, inspection details, cleanup sizing, selector keymaps, and cross-platform parity.
---

# Homebase Platform UI Workflow

Use this workflow for Homebase platform install and cleanup changes.

## Relevant Subagents

- Use `cli-developer` for selector UX, flag behavior, interactive versus
  unattended command flows, and shell-facing workflow changes.
- Use `golang-pro` for Go implementation details in shared packages and
  platform command code.
- Use `test-automator` for regression coverage around selectors, fake runners,
  config parsing, and scanner behavior.
- Use `reviewer` after non-trivial platform parity or cleanup/install changes.
- Use `code-mapper` before broad changes that cross `internal/ui`,
  `internal/install`, `internal/cleanup`, and `internal/platform/*`.

## Architecture

- Keep shared list behavior in `internal/ui`; platform code should only
  supply `ui.SelectItem` data.
- Use `ui.SelectItem.State`, `DetailValue`, `Detail`, `Inspect`, and
  `DefaultSelected` instead of rendering custom platform list rows.
- Put calculated stateful values in `DetailValue`; the selector highlights
  that value according to `State`. Keep stable descriptions and command
  summaries in `Detail`.
- Keep platform-specific state detection near the platform command flow. Arch
  install state belongs in `internal/install`; Arch cleanup scanners belong in
  `internal/cleanup`; Windows-specific detection belongs in
  `internal/platform/windows`.
- Treat scanner failures as non-blocking. Unknown install or cleanup state
  should stay selectable and use `ui.SelectStateUnknown`.
- Keep `default = true` as a per-item config flag on package groups and
  cleanup tasks. It only preselects interactive lists when no explicit
  `--group`, `--task`, or `--all` choice was given.
- Keep default config safe: omit `default = true` unless the item is
  intentionally preselected by the user or platform owner.

## Install Items

- Green means every configured package or action is already satisfied.
- Yellow means only part of the item is satisfied, or the platform cannot
  fully know.
- Red means none of the package payload is installed or the item clearly needs work.
- Put the concise calculated ratio or summary in `DetailValue`.
- Put package/action breakdowns in `Inspect`, including installed versus
  missing items when known.

## Cleanup Items

- Green means no cleanup work was detected.
- Yellow means cleanup size is unknown or small.
- Red means reclaimable work was detected.
- Prefer cheap pre-list scanners: directory sizes for caches,
  package-manager queries for orphan counts, and native disk-usage commands
  for logs.
- Do not require sudo for scanners unless the cleanup task itself already
  needs privileged confirmation.
- Show scanner output and the cleanup command detail in `Inspect`.
- For Arch orphan cleanup, keep removal conservative: show review guidance,
  never pass `--noconfirm` to `pacman -Rns`, and let pacman show its own
  transaction confirmation.

## Selector Behavior

- Preserve shared controls: `j/k`, arrows, `gg`, `G`, page keys, `space`,
  `a`, `i`, `enter`, `q`.
- `i` toggles inspect details for the current item. `space` toggles
  selection. `enter` confirms.
- `ctrl+d` and `ctrl+u` scroll the fixed-height inspect panel.
- Keep fallback numbered selection compatible with defaults.

## Validation Checklist

- Add or update config parsing tests for new TOML fields.
- Add or update item-building tests for state, calculated detail values,
  inspect text, and defaults.
- Add platform scanner tests with fake runners or temporary directories.
- Run `gofmt -w cmd internal`.
- Run `go test ./...`, `go vet ./...`, and
  `go build -o ~/.local/bin/hb ./cmd/hb`.
- If README or Markdown changes were made, run
  `markdownlint-cli2 README.md`
  when available.
