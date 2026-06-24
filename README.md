# Homebase

Homebase is a Go CLI for bootstrapping and maintaining an Arch Linux dotfiles
environment. It builds one binary, `hb`, and uses TOML config plus an
interactive Bubble Tea UI for package installation, cleanup, bootstrap, and
dotfile sync workflows.

Homebase is separate from the dotfiles repository. The dotfiles repo keeps the
small remote `bootstrap.sh` entrypoint, while Homebase owns the real setup and
maintenance logic.

## Features

- Bootstrap an Arch Linux machine from a bare dotfiles repository
- Build and install one binary at `~/.local/bin/hb`
- Seed default config into `~/.config/homebase`
- Install configured pacman and AUR package groups
- Run post-install setup for SDDM, NetworkManager, dnsmasq, Docker, Razer,
  Sunshine, and tty1 autologin
- Clean common system caches and orphan packages
- Stage, commit, and push configured dotfile paths through the bare git repo
- Support interactive UI by default and automation flags for non-interactive runs

## Requirements

- Arch Linux
- Go
- Git
- rsync
- `base-devel`
- `pacman`

The default AUR helper is `yay`. If `yay` is missing, `hb install` builds it
from the AUR.

## Installation

The normal install path is through the dotfiles bootstrap script:

```bash
repo_url="https://raw.githubusercontent.com/gin31259461/dotfiles-arch"
bash <(curl -fsSL "$repo_url/main/.local/bin/bootstrap.sh")
```

The bootstrap script installs the minimum system dependencies, clones Homebase
to `~/.local/lib/homebase`, builds `hb`, and then runs:

```bash
hb bootstrap
```

For a non-interactive bootstrap:

```bash
repo_url="https://raw.githubusercontent.com/gin31259461/dotfiles-arch"
bash <(curl -fsSL "$repo_url/main/.local/bin/bootstrap.sh") --yes
```

To run package installation after bootstrap:

```bash
repo_url="https://raw.githubusercontent.com/gin31259461/dotfiles-arch"
bash <(curl -fsSL "$repo_url/main/.local/bin/bootstrap.sh") --yes --install
```

## Build From Source

```bash
git clone https://github.com/gin31259461/homebase.git ~/.local/lib/homebase
cd ~/.local/lib/homebase
go build -o ~/.local/bin/hb ./cmd/hb
```

Verify the binary:

```bash
hb help
```

## Usage

```bash
hb bootstrap [--yes] [--repo <repo>] [--install]
hb install   [--group <key>] [--all] [--yes] [--no-setup]
hb cleanup   [--task <key>] [--all] [--yes]
hb sync      [-m <message>] [--no-push]
hb config init
```

Interactive commands use Bubble Tea by default. For automation, pass explicit
selections and `--yes`.

## Bootstrap

`hb bootstrap` configures the machine after the shell bootstrap has built the
binary.

```bash
hb bootstrap
hb bootstrap --yes
hb bootstrap --repo git@github.com:youruser/dotfiles-arch.git
hb bootstrap --repo youruser/dotfiles-arch --install
```

Bootstrap behavior:

1. Checks that the host is Arch Linux
2. Ensures Homebase config exists
3. Installs bootstrap basics from `config.toml`
4. Clones the dotfiles repo as a bare repo in `~/.dotfiles`
5. Deploys files into `$HOME`
6. Stores the selected dotfiles remote in `~/.dotfiles-repo`
7. Sets `status.showUntrackedFiles = no`
8. Adds the `dot` alias to `.zshrc` when missing
9. Initializes git submodules
10. Offers Oh My Zsh setup
11. Optionally runs `hb install`

## Install Packages

Run the interactive selector:

```bash
hb install
```

Install specific groups:

```bash
hb install --group core --group shell
hb install --group docker --yes
hb install --all --yes
```

Skip post-install setup hooks:

```bash
hb install --group dev --yes --no-setup
```

Package groups live in:

```text
~/.config/homebase/packages.d/*.toml
```

Default package config is copied from:

```text
~/.local/lib/homebase/config/packages.d/
```

## Cleanup

Run the interactive cleanup selector:

```bash
hb cleanup
```

Run specific tasks:

```bash
hb cleanup --task pacman-cache --task journal
hb cleanup --all --yes
```

Cleanup tasks are configured in:

```text
~/.config/homebase/cleanup.toml
```

## Sync Dotfiles

`hb sync` stages configured paths from `$HOME`, commits them in the bare
dotfiles repo, and pushes to the configured branch.

```bash
hb sync
hb sync -m "chore: sync dotfiles"
hb sync -m "chore: sync dotfiles" --no-push
```

Tracked path groups are configured in:

```text
~/.config/homebase/sync.toml
```

Homebase uses the same bare git layout as the `dot` alias:

```bash
git --git-dir=$HOME/.dotfiles/ --work-tree=$HOME
```

## Configuration

Default config is stored in this repository:

```text
config/
|-- config.toml
|-- cleanup.toml
|-- sync.toml
`-- packages.d/
```

At runtime, Homebase copies defaults into `~/.config/homebase` when that
directory is missing or empty.

Force-copy the default config:

```bash
hb config init
```

Main settings live in `config.toml`:

```toml
[homebase]
repo = "https://github.com/gin31259461/homebase.git"
dir = "~/.local/lib/homebase"

[dotfiles]
ssh_repo = "git@github.com:gin31259461/dotfiles-arch.git"
https_repo = "https://github.com/gin31259461/dotfiles-arch.git"
dir = "~/.dotfiles"
branch = "main"
memory_file = "~/.dotfiles-repo"

[package_manager]
official = "pacman"
aur = "yay"
```

The dotfiles memory file supports TOML:

```toml
repo = "git@github.com:gin31259461/dotfiles-arch.git"
branch = "main"
```

## Development

Run formatting, tests, vet, and build:

```bash
gofmt -w cmd internal
go test ./...
go vet ./...
go build -o ~/.local/bin/hb ./cmd/hb
```

Smoke-check the binary:

```bash
hb help
hb install --group __none__ --yes --no-setup
```

## Architecture

```text
.
|-- cmd/hb/              # thin CLI router
|-- internal/bootstrap/  # hb bootstrap
|-- internal/cleanup/    # hb cleanup
|-- internal/config/     # TOML loading and config seeding
|-- internal/gitutil/    # bare git repo helpers and repo memory
|-- internal/install/    # hb install and install planning
|-- internal/run/        # command runner abstraction
|-- internal/setup/      # post-install setup routines
|-- internal/sync/       # hb sync
|-- internal/system/     # Arch, pacman, systemd, and user helpers
|-- internal/testutil/   # test fakes
|-- internal/ui/         # Bubble Tea selector, prompts, spinner, styles
|-- config/              # default runtime config
|-- go.mod
`-- go.sum
```

The project intentionally produces one binary. Internal packages keep concerns
separate without exposing a public Go API.

## Testing

Unit tests live next to the package they test:

```text
internal/config/config_test.go
internal/gitutil/gitutil_test.go
internal/install/plan_test.go
internal/system/system_test.go
internal/ui/ui_test.go
```

Run all tests:

```bash
go test ./...
```

## Troubleshooting

If `hb` is missing, rebuild it from the Homebase checkout:

```bash
cd ~/.local/lib/homebase
go build -o ~/.local/bin/hb ./cmd/hb
```

If config is missing:

```bash
hb config init
```

If AUR installs fail, verify that `base-devel`, `git`, and the configured AUR
helper are available:

```bash
pacman -Qi base-devel git
command -v yay
```
