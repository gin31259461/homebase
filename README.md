# Homebase

Homebase is a Go CLI for bootstrapping and maintaining dotfiles environments.
It builds one binary, `hb`, detects the active platform, seeds TOML defaults,
and routes setup work to platform-owned code.

Use Homebase when you want a repeatable way to bring a Windows or Arch-family
machine onto the same dotfiles, package groups, cleanup tasks, and sync workflow
without keeping that logic in one large shell script.

## Why Homebase

- Bootstrap a fresh machine from source into `~/.local/lib/homebase`
- Build a single `hb` binary into `~/.local/bin`
- Seed only the global config plus the detected platform config
- Deploy dotfiles as a bare Git repository at `~/.dotfiles`
- Install platform package groups through an interactive selector or explicit
  automation flags
- Run platform cleanup tasks with inspectable plans before destructive work
- Stage, commit, and push configured dotfile paths with `hb sync`

Homebase is opinionated about ownership. Shared packages handle routing,
configuration, selectors, and reusable helpers. Platform packages own install
policy, cleanup policy, bootstrap differences, OS paths, setup hooks, and
side effects.

## Supported Platforms

- `archlinux`: systems detected by `/etc/arch-release` or
  `/etc/manjaro-release`
- `windows`: Windows detected by Go at runtime

Platform detection runs automatically. To override it, edit:

```text
~/.config/homebase/homebase.toml
```

```toml
active_platform = "windows"
```

Use `active_platform = "auto"` for normal detection.

## Requirements

Runtime bootstrap requirements:

- Windows: PowerShell 5.1 or newer and `winget` in `PATH`
- Arch Linux or Manjaro: `bash`, `curl`, `sudo`, and `pacman`
- Build from source: Go 1.24.2, as declared in `go.mod`

Development requirements:

- `make`
- `markdownlint-cli2` for `make lint`

## Get Started

The default platform configs in this repository point at the maintainer's
dotfiles repositories:

```text
git@github.com:gin31259461/dotfiles-arch.git
git@github.com:gin31259461/dotfiles-win.git
```

For your own machine, pass your dotfiles repository during bootstrap or edit
the runtime config before running `hb bootstrap`.

### Arch Linux

Run the remote bootstrap and pass your dotfiles repo:

```bash
repo_url="https://raw.githubusercontent.com/gin31259461/homebase"
bash <(curl -fsSL "$repo_url/main/bootstrap/archlinux.sh") \
  --repo git@github.com:you/dotfiles-arch.git
```

Useful local options from a checked-out copy:

```bash
bash bootstrap/archlinux.sh --yes
bash bootstrap/archlinux.sh --install
bash bootstrap/archlinux.sh --repo git@github.com:you/dotfiles-arch.git
bash bootstrap/archlinux.sh --homebase-dir "$HOME/src/homebase"
```

The Arch bootstrap verifies an Arch-family host, installs minimum build
dependencies with `pacman`, clones or updates Homebase, builds `hb`, and runs
`hb bootstrap` with the remaining arguments.

### Windows

Run the remote bootstrap from PowerShell:

```powershell
$repoUrl = "https://raw.githubusercontent.com/gin31259461/homebase"
$script = irm "$repoUrl/main/bootstrap/windows.ps1"
& ([scriptblock]::Create($script)) -DotfilesRepo "git@github.com:you/dotfiles-win.git"
```

Useful local options from a checked-out copy:

```powershell
.\bootstrap\windows.ps1 -Yes
.\bootstrap\windows.ps1 -Install
.\bootstrap\windows.ps1 -DotfilesRepo "git@github.com:you/dotfiles-win.git"
```

The Windows bootstrap adds `~/.local/bin` to the user `Path`, installs Git and
Go through WinGet when missing, clones or updates Homebase, builds `hb.exe`,
and runs `hb bootstrap`.

### Build From Source

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

## Command Summary

```text
hb bootstrap [--yes] [--repo <repo>] [--install]
hb install   [--group <key>] [--all] [--yes] [--no-setup]
hb cleanup   [--task <key>] [--all] [--yes]
hb sync      [-m <message>] [--no-push]
hb config init [-f|--force]
```

Interactive commands use Bubble Tea by default. Automation should pass `--yes`
with explicit `--group` or `--task` selections, or `--all`.

## Core Workflows

### Bootstrap

`hb bootstrap` prepares the current machine after the shell bootstrap has built
the binary.

```bash
hb bootstrap --repo git@github.com:you/dotfiles.git
hb bootstrap --repo you/dotfiles --install
hb bootstrap --yes
```

Bootstrap ensures config exists, installs configured basics, clones the
dotfiles repo as a bare repo, copies tracked files into `$HOME`, records the
chosen dotfiles remote in `~/.dotfiles-repo`, initializes submodules, configures
dotfile Git status, and optionally runs `hb install`.

On Arch-family systems, the shared bootstrap flow can also install Oh My Zsh
and bundled plugins/theme when accepted or when running with `--yes`.

### Install

Run the interactive selector:

```bash
hb install
```

Install explicit groups:

```bash
hb install --group core --group cli --yes
hb install --all --yes
hb install --group core --yes --no-setup
```

Arch package groups use `pacman` and AUR packages through the configured helper,
defaulting to `yay`. Windows groups support WinGet packages, Scoop buckets,
Scoop packages, PowerShell modules, and setup features.

Runtime package groups live in:

```text
~/.config/homebase/platforms/<platform>/packages.d/*.toml
```

### Cleanup

Run the interactive cleanup selector:

```bash
hb cleanup
```

Run explicit tasks:

```bash
hb cleanup --task scoop-cache --task temp-files --yes
hb cleanup --all --yes
```

Runtime cleanup tasks live in:

```text
~/.config/homebase/platforms/<platform>/cleanup.toml
```

Tasks with a missing `requires` command are hidden. Arch orphan cleanup is
intentionally conservative: it prints the package list and review guidance,
then runs `sudo pacman -Rns <packages...>` without `--noconfirm` so pacman can
show its own transaction prompt.

### Sync Dotfiles

`hb sync` stages configured paths from `$HOME`, commits them in the bare
dotfiles repo, and pushes to the configured branch unless `--no-push` is set.

```bash
hb sync
hb sync -m "chore: sync dotfiles"
hb sync -m "chore: sync dotfiles" --no-push
```

An empty prompted commit message exits without staging, committing, or pushing.
Tracked paths live in:

```text
~/.config/homebase/platforms/<platform>/sync.toml
```

Homebase uses this bare Git layout for dotfiles operations:

```bash
git --git-dir=$HOME/.dotfiles --work-tree=$HOME
```

### Config

Default config lives in this repository under `config/`. At runtime, Homebase
copies only the global config and the detected platform defaults into
`~/.config/homebase`.

```bash
hb config init
hb config init --force
```

`hb config init` seeds missing files without overwriting existing config.
`--force` refreshes runtime config from repository defaults.

Global config:

```toml
active_platform = "auto"

[platform_aliases]
arch = "archlinux"
manjaro = "archlinux"
```

Platform config includes dotfiles, package manager, and bootstrap basics:

```toml
[dotfiles]
ssh_repo = "git@github.com:you/dotfiles.git"
https_repo = "https://github.com/you/dotfiles.git"
dir = "~/.dotfiles"
branch = "main"
memory_file = "~/.dotfiles-repo"

[package_manager]
official = "pacman"
aur = "yay"

[bootstrap]
basics = [
  "git",
]
```

Package group fields:

```toml
[example]
label = "Example"
default = false
pacman = ["git"]
aur = ["yay-bin"]
winget = ["Git.Git"]
scoop_buckets = ["nerd-fonts"]
scoop = ["FiraCode-NF"]
psmodules = ["PSReadLine"]
features = ["powershell-profile"]
```

Cleanup task fields:

```toml
[cache]
label = "Cache"
default = false
detail = "safe cleanup command summary"
requires = "optional-command"
sudo = false
```

Set `default = true` only when an item should be preselected in the interactive
selector. Explicit `--group`, `--task`, and `--all` arguments ignore interactive
defaults.

## Project Layout

```text
cmd/hb/              thin CLI router
bootstrap/           platform bootstrap entrypoints
config/              default runtime config copied to ~/.config/homebase
internal/bootstrap/  shared bootstrap flow and helpers
internal/install/    shared install selector/planning helpers
internal/cleanup/    shared cleanup helpers
internal/sync/       hb sync implementation
internal/config/     TOML loading and config seeding
internal/platform/   platform registry and implementations
internal/ui/         Bubble Tea selector, prompts, spinner, styles
internal/run/        command runner abstraction
internal/system/     system probes
internal/gitutil/    bare Git and repo memory helpers
internal/testutil/   shared test fakes
```

Platform-owned code lives under:

```text
internal/platform/archlinux/
internal/platform/windows/
```

## FAQ

### How do I avoid cloning the maintainer's dotfiles?

Pass `--repo` to `hb bootstrap` or pass `-DotfilesRepo` to the Windows
bootstrap script. You can also edit `~/.config/homebase/platforms/<platform>/`
after `hb config init` and before running `hb bootstrap`.

### Where does Homebase install itself?

The default source checkout is `~/.local/lib/homebase`. The binary is
`~/.local/bin/hb` on Unix-like systems and `~/.local/bin/hb.exe` on Windows.

### Why did platform detection fail?

Homebase currently supports Windows and Arch-family Linux. On a supported host,
run `hb config init`, then set `active_platform` in
`~/.config/homebase/homebase.toml` if auto-detection is not enough.

### Why was a package group or cleanup task skipped?

Unknown `--group` and `--task` keys are skipped with a warning. Cleanup tasks
with a `requires` command are also hidden when that command is not available.

### What gets copied into runtime config?

Homebase copies `config/homebase.toml` plus only the active platform subtree
from `config/platforms/<platform>/` into `~/.config/homebase`.

### What if Windows cannot find `hb`?

Open a new terminal or confirm this directory is in the user `Path`:

```text
%USERPROFILE%\.local\bin
```

### What if Arch AUR installs fail?

Verify the bootstrap dependencies and configured AUR helper:

```bash
pacman -Qi base-devel git
command -v yay
```

## Contributing

Read `AGENTS.md` before changing code. It captures the repository boundaries,
side-effect rules, verification commands, and AI-specific workflow constraints.

Before finishing code changes, run:

```bash
make check
make build
```

For README or Markdown changes, also run:

```bash
make lint
```

Smoke-check command routing changes:

```bash
make smoke
```

Bug reports should include the platform, command, relevant flags, expected
behavior, actual output, and whether runtime config came from defaults or local
edits. Pull requests should keep command names and flags stable unless the
change intentionally proposes a breaking CLI change.
