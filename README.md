# Homebase

Homebase is a Go CLI for bootstrapping and maintaining the maintainer's
dotfiles-based workstation setup. It builds one binary, `hb`, and keeps
platform-specific behavior behind the active platform implementation.

## Supported Platforms

- `archlinux`: Arch Linux and Manjaro hosts
- `windows`: Windows hosts

The checked-in defaults point at the maintainer's dotfiles repositories. For a
different machine, pass your own dotfiles repository during bootstrap or edit
the runtime config under `~/.config/homebase`.

## Requirements

- Go 1.24.2, matching `go.mod`
- `make` for development shortcuts
- `markdownlint-cli2` for Markdown linting
- Arch Linux or Manjaro: `bash`, `curl`, `sudo`, `pacman`
- Windows: PowerShell 5.1 or newer, `winget`

Bootstrap and install commands perform real machine setup. Read the command
sections before running them on a daily-use system.

## Installation

### Arch Linux

Run the bootstrap script from GitHub:

```bash
repo_url="https://raw.githubusercontent.com/gin31259461/homebase"
bash <(curl -fsSL "$repo_url/main/bootstrap/archlinux.sh") \
  --repo git@github.com:you/dotfiles-arch.git
```

Useful local options:

```bash
bash bootstrap/archlinux.sh --yes
bash bootstrap/archlinux.sh --install
bash bootstrap/archlinux.sh --repo git@github.com:you/dotfiles-arch.git
bash bootstrap/archlinux.sh --homebase-dir "$HOME/src/homebase"
bash bootstrap/archlinux.sh --homebase-repo https://github.com/you/homebase.git
```

The script installs `git`, `base-devel`, and `go` with `pacman`,
then clones or updates Homebase at `~/.local/lib/homebase`,
builds `hb` into `~/.local/bin`, and runs `hb bootstrap`.

### Windows

Run the bootstrap script from PowerShell:

```powershell
$repoUrl = "https://raw.githubusercontent.com/gin31259461/homebase"
$script = irm "$repoUrl/main/bootstrap/windows.ps1"
& ([scriptblock]::Create($script)) -DotfilesRepo "git@github.com:you/dotfiles-win.git"
```

Useful local options:

```powershell
.\bootstrap\windows.ps1 -Yes
.\bootstrap\windows.ps1 -Install
.\bootstrap\windows.ps1 -DotfilesRepo "git@github.com:you/dotfiles-win.git"
.\bootstrap\windows.ps1 `
  -HomebaseRepo "https://github.com/you/homebase.git" `
  -Branch main
```

The script adds `~/.local/bin` to the user `Path`,
installs Git and Go through WinGet when missing,
clones or updates Homebase, builds `hb.exe`,
and runs `hb bootstrap`.

### Build from Source

```bash
git clone https://github.com/gin31259461/homebase.git ~/.local/lib/homebase
cd ~/.local/lib/homebase
go build -o ~/.local/bin/hb ./cmd/hb
hb help
```

On Windows:

```powershell
git clone https://github.com/gin31259461/homebase.git ~/.local/lib/homebase
Set-Location ~/.local/lib/homebase
go build -o ~/.local/bin/hb.exe ./cmd/hb
hb help
```

`HOMEBASE_DIR` can point `hb` at a different source checkout for config
seeding. Without it, Homebase reads defaults from `~/.local/lib/homebase`.

## Commands

```text
hb bootstrap [--yes] [--repo <repo>] [--install]
hb install   [--group <key>] [--all] [--yes] [--no-setup]
hb cleanup   [--task <key>] [--all] [--yes]
hb sync      [-m <message>] [--no-push]
hb config init [-f|--force]
```

Interactive install and cleanup flows use a Bubble Tea selector. For
unattended runs, pass `--yes` with explicit `--group` or `--task` values, or
use `--all`.

Unknown explicit group and task keys are skipped with a warning.

### Bootstrap

`hb bootstrap` prepares the dotfiles environment after the shell bootstrap has
built the binary.

```bash
hb bootstrap --repo git@github.com:you/dotfiles.git
hb bootstrap --repo you/dotfiles --install
hb bootstrap --yes
```

Common behavior:

- Seeds runtime config for the detected platform.
- Installs configured bootstrap basics.
- Deploys the dotfiles repo as a bare Git repo at `~/.dotfiles`.
- Copies tracked files into `$HOME`.
- Stores the resolved repo in `~/.dotfiles-repo`.
- Runs `git submodule update --init --recursive`.
- Runs `hb install --all --yes` when `--install` and `--yes` are both used.

Arch bootstrap also configures the `dot` alias in `.zshrc` and can install
Oh My Zsh plus the configured plugins and theme. Windows bootstrap links
PowerShell profiles when `~/.pwsh/profile.ps1` exists.

Repository inputs accepted by `--repo`:

- `user/repo`
- `git@github.com:user/repo.git`
- `https://github.com/user/repo.git`
- Other `git@...` SSH URLs

For GitHub repositories, Homebase prefers SSH and falls back to HTTPS when SSH
access to `HEAD` is unavailable.

### Install Packages

```bash
hb install
hb install --group core --group cli-tools --yes
hb install --all --yes
hb install --group docker --yes --no-setup
```

Package groups are read from:

```text
~/.config/homebase/platforms/<platform>/packages.d/*.toml
```

Arch groups can include:

- `pacman`
- `aur`

Arch install scans installed packages with `pacman -Qq`, installs missing
official packages with `sudo pacman -S --needed --noconfirm`, installs missing
AUR packages with the configured helper, and runs post-install setup hooks
unless `--no-setup` is passed.

Windows groups can include:

- `features`
- `winget`
- `scoop_buckets`
- `scoop`
- `psmodules`

Windows install can install WinGet packages, Scoop buckets and packages,
PowerShell modules, and setup features such as PowerShell profile links,
WezTerm context menu entries, and the Windows 10 classic context menu key.
`--no-setup` filters out setup-only features while keeping core features such
as Scoop and Node/pnpm.

### Cleanup

```bash
hb cleanup
hb cleanup --task pacman-cache --task thumbnails --yes
hb cleanup --task temp-files --task recycle-bin --yes
hb cleanup --all --yes
```

Cleanup tasks are read from:

```text
~/.config/homebase/platforms/<platform>/cleanup.toml
```

Tasks with a `requires` command are hidden when that command is unavailable.
The selector shows cheap scanner results when implemented. Unknown scanner
state remains selectable.

Arch cleanup includes tasks for pacman cache, AUR cache, orphan packages,
systemd journal, npm cache, and thumbnails. Orphan package cleanup is
conservative: it prints a review list and uses pacman confirmation instead of
`--noconfirm`.

Windows cleanup includes tasks for Scoop cache, temp files, npm cache, WinGet
cache, Recycle Bin, and thumbnail cache.

### Sync Dotfiles

`hb sync` stages configured paths from `$HOME`, commits them in the bare
dotfiles repo, and pushes to the configured branch unless `--no-push` is set.

```bash
hb sync
hb sync -m "chore: sync dotfiles"
hb sync -m "chore: sync dotfiles" --no-push
```

Tracked paths are read from:

```text
~/.config/homebase/platforms/<platform>/sync.toml
```

Homebase runs dotfiles Git commands with this layout:

```bash
git --git-dir=$HOME/.dotfiles --work-tree=$HOME
```

If no commit message is passed, `hb sync` prompts for one. An empty message
exits without committing.

### Config

Default config lives in `config/`. Runtime config is seeded to:

```text
~/.config/homebase
```

Only `config/homebase.toml` and the detected platform subtree are copied.
Existing files are preserved unless `--force` is used.

```bash
hb config init
hb config init --force
```

Global platform selection:

```toml
active_platform = "auto"

[platform_aliases]
arch = "archlinux"
manjaro = "archlinux"
```

- Use `active_platform = "auto"` for normal detection,
  or set a platform id such as `"archlinux"` or `"windows"` to override detection
- `default = true` preselects an item in the interactive selector
- Explicit `--group`, `--task`, and `--all` choices ignore interactive defaults

## Project Layout

```text
cmd/hb/              CLI router
bootstrap/           platform bootstrap entrypoints
config/              defaults copied to ~/.config/homebase
internal/bootstrap/  shared bootstrap flow and helpers
internal/install/    shared install selector and planning helpers
internal/cleanup/    shared cleanup helpers
internal/sync/       hb sync implementation
internal/config/     TOML loading and config seeding
internal/platform/   platform detection and implementations
internal/ui/         Bubble Tea selectors, prompts, spinner, styles
internal/run/        command runner abstraction
internal/system/     small system probes
internal/gitutil/    bare Git and repo memory helpers
internal/testutil/   shared test fakes
```

Platform-owned behavior lives under `internal/platform/archlinux/` and
`internal/platform/windows/`.

## Development

Run the standard checks before finishing code changes:

```bash
make check
make build
```

Do not use `hb install`, `hb cleanup`, `hb bootstrap`, package managers, sudo,
or bootstrap scripts as verification unless you explicitly want live machine
side effects.
