# Homebase

![Go](https://img.shields.io/badge/Go-1.24.2-00ADD8?style=flat-square&logo=go&logoColor=white)
![CLI](https://img.shields.io/badge/CLI-hb-222?style=flat-square)
![Platforms](https://img.shields.io/badge/platforms-Arch%20Linux%20%7C%20Windows-blue?style=flat-square)

Homebase is a Go CLI for bootstrapping and maintaining personal dotfiles
environments. It builds one binary, `hb`, detects the active platform, seeds
platform-specific TOML defaults, and delegates setup work to the matching
platform package.

Sections: [Features](#features), [Install](#install), [Usage](#usage),
[Configuration](#configuration), [Development](#development).

## Features

- Bootstrap a bare Git dotfiles repository into `$HOME`.
- Install grouped packages from TOML defaults.
- Run platform-specific setup hooks after package installation.
- Inspect and run cleanup tasks for package caches, temp files, journals, and
  other common local clutter.
- Sync configured dotfile paths by staging, committing, and optionally pushing
  through the bare Git repository.
- Support interactive Bubble Tea selectors and unattended flags for automation.

## Platforms

Arch Linux and Manjaro:

- Detection: `/etc/arch-release` or `/etc/manjaro-release`
- Package sources: `pacman` and an AUR helper, default `yay`
- Cleanup: pacman cache, AUR cache, orphans, journal, npm cache, thumbnails

Windows:

- Detection: `runtime.GOOS == "windows"`
- Package sources: WinGet, Scoop, PowerShell modules, setup features
- Cleanup: temp files, Recycle Bin, WinGet cache, Scoop cache, npm cache,
  thumbnails

> [!IMPORTANT]
> Homebase is designed for personal workstation setup. Bootstrap, install, and
> cleanup commands can modify dotfiles, package databases, shell profiles,
> services, and user registry entries. Review the TOML defaults before running
> unattended commands.

## Install

### Arch Linux

```bash
url=https://raw.githubusercontent.com/gin31259461/homebase/main/bootstrap
curl -fsSL "$url/archlinux.sh" | bash
```

The script installs Git, `base-devel`, and Go with `pacman`, clones Homebase to
`~/.local/lib/homebase`, builds `hb` into `~/.local/bin/hb`, then runs
`hb bootstrap`.

Useful options:

```bash
url=https://raw.githubusercontent.com/gin31259461/homebase/main/bootstrap
curl -fsSL "$url/archlinux.sh" | \
  bash -s -- --repo gin31259461/dotfiles-arch --yes --install
```

### Windows

Run from PowerShell:

```powershell
$url = "https://raw.githubusercontent.com/gin31259461/homebase/main/bootstrap"
irm "$url/windows.ps1" | iex
```

The script ensures Git and Go are available through WinGet, clones Homebase to
`$HOME\.local\lib\homebase`, builds `hb.exe` into `$HOME\.local\bin`, then runs
`hb bootstrap`.

With explicit options:

```powershell
$url = "https://raw.githubusercontent.com/gin31259461/homebase/main/bootstrap"
$script = irm "$url/windows.ps1"
& ([scriptblock]::Create($script)) `
  -DotfilesRepo gin31259461/dotfiles-win `
  -Yes `
  -Install
```

### Build from source

```bash
git clone https://github.com/gin31259461/homebase.git ~/.local/lib/homebase
cd ~/.local/lib/homebase
make build
```

`make build` writes the binary to `~/.local/bin/hb` on Unix-like systems and
`%USERPROFILE%/.local/bin/hb.exe` on Windows unless `HB_BIN` is overridden.

## Usage

```text
hb bootstrap [--yes] [--repo <repo>] [--install]
hb install   [--group <key>] [--all] [--yes] [--no-setup]
hb cleanup   [--task <key>] [--all] [--yes]
hb sync      [-m <message>] [--no-push]
hb config init [-f|--force]
```

### Bootstrap dotfiles

```bash
hb bootstrap --repo gin31259461/dotfiles-arch --yes
```

Bootstrap resolves the configured dotfiles repository, clones it as a bare Git
repo at `~/.dotfiles`, copies the worktree into `$HOME`, configures
`status.showUntrackedFiles = no`, initializes submodules, and records the chosen
repo in `~/.dotfiles-repo`.

On Arch Linux, bootstrap also installs configured basic packages and can install
Oh My Zsh plus common plugins. On Windows, bootstrap ensures `~/.local/bin` is
on the user PATH and links PowerShell profiles from `.pwsh/profile.ps1` when
present.

### Install package groups

```bash
hb install
hb install --group cli-tools --group terminal --yes --no-setup
hb install --all --yes
```

Without flags, Homebase opens an interactive selector. With flags, it builds an
install plan from the selected TOML groups and skips packages already detected
as installed where the platform scanner supports it.

### Clean local caches

```bash
hb cleanup
hb cleanup --task npm-cache --yes
hb cleanup --all --yes
```

Cleanup uses platform TOML task definitions and scanner output to show
reclaimable size where possible before asking for confirmation.

### Sync dotfiles

```bash
hb sync -m "Update shell config"
hb sync -m "Update editor config" --no-push
```

`hb sync` loads the active platform's `sync.toml`, stages those paths through
the bare dotfiles repository, commits with the provided message, and pushes to
the configured branch unless `--no-push` is set.

## Configuration

Homebase seeds defaults from the repository into `~/.config/homebase`.

```text
~/.config/homebase/
|-- homebase.toml
`-- platforms/
    |-- archlinux/
    |   |-- config.toml
    |   |-- cleanup.toml
    |   |-- sync.toml
    |   `-- packages.d/*.toml
    `-- windows/
        |-- config.toml
        |-- cleanup.toml
        |-- sync.toml
        `-- packages.d/*.toml
```

Run this to seed or refresh the active platform defaults:

```bash
hb config init
hb config init --force
```

`homebase.toml` controls platform detection. By default:

```toml
active_platform = "auto"

[platform_aliases]
arch = "archlinux"
manjaro = "archlinux"
```

Set `HOMEBASE_DIR` when running from a non-default source checkout. If unset,
Homebase reads default config from `~/.local/lib/homebase`.

## Development

Prerequisites:

- Go `1.24.2`
- `make`
- `markdownlint-cli2` for Markdown linting

Common commands:

```bash
make check   # gofmt, go test ./..., go vet ./...
make build   # build hb into ~/.local/bin
make lint    # markdownlint README.md
make smoke   # build, hb help, non-mutating empty install selection
```

The CLI entry point is intentionally thin: `cmd/hb/main.go` routes commands,
while command behavior lives under `internal/*`. Platform-owned behavior lives
under `internal/platform/archlinux` and `internal/platform/windows`.

> [!TIP]
> For unattended install and cleanup examples, pass explicit `--group` or
> `--task` values, or use `--all --yes`. Interactive defaults are controlled by
> `default = true` in TOML package groups and cleanup tasks.

## Troubleshooting

- `no supported platform detected`: set `active_platform` in
  `~/.config/homebase/homebase.toml` or run on a supported platform.
- `default config ... not found`: confirm `HOMEBASE_DIR` points at the Homebase
  source checkout, or use the default `~/.local/lib/homebase` location.
- GitHub SSH clone fails during bootstrap: Homebase falls back to HTTPS for
  GitHub repos when SSH `HEAD` is unavailable, then resets the bare repo origin
  to SSH when an SSH repo was configured.
