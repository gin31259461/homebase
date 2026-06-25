# Homebase

Homebase is a Go CLI for bootstrapping and maintaining dotfiles environments.
It builds one binary, `hb`, detects the active platform, seeds TOML defaults,
and dispatches bootstrap, install, cleanup, and sync commands to platform
specific code.

Homebase is installed as source at `~/.local/lib/homebase` and built to
`~/.local/bin/hb`. On Windows the binary is `~/.local/bin/hb.exe`.

## Features

- One `hb` CLI with platform-specific behavior behind automatic detection
- Bare Git dotfiles deployment into `$HOME`
- TOML defaults copied to `~/.config/homebase`
- Interactive package and cleanup selectors with automation flags
- Package groups for Arch Linux and Windows
- Windows WinGet, Scoop, Scoop bucket, PowerShell module, and setup support
- Arch Linux pacman and AUR support
- Dotfiles staging, commit, and push through the configured bare repo

## Platforms

Homebase currently supports:

- `archlinux`: Arch Linux and compatible Arch-family systems such as Manjaro
- `windows`: Windows PowerShell 5.1+ bootstrap with WinGet, Scoop, and Go

Platform detection runs automatically. To override it, edit:

```text
~/.config/homebase/homebase.toml
```

```toml
active_platform = "windows"
```

Use `active_platform = "auto"` for normal detection.

## Install

### Windows

Run the remote bootstrap from PowerShell:

```powershell
$repoUrl = "https://raw.githubusercontent.com/gin31259461/homebase"
irm "$repoUrl/main/bootstrap/windows.ps1" | iex
```

The Windows bootstrap:

1. Adds `~/.local/bin` to the user `Path`
2. Installs Git and Go through WinGet when missing
3. Clones or updates Homebase at `~/.local/lib/homebase`
4. Builds `~/.local/bin/hb.exe`
5. Runs `hb bootstrap`

Useful Windows bootstrap options:

```powershell
.\bootstrap\windows.ps1 -Yes
.\bootstrap\windows.ps1 -Install
.\bootstrap\windows.ps1 -DotfilesRepo "git@github.com:you/dotfiles-win.git"
```

### Arch Linux

Run the remote bootstrap from a shell:

```bash
repo_url="https://raw.githubusercontent.com/gin31259461/homebase"
bash <(curl -fsSL "$repo_url/main/bootstrap/archlinux.sh")
```

The Arch bootstrap installs the minimum pacman dependencies, clones or updates
Homebase at `~/.local/lib/homebase`, builds `~/.local/bin/hb`, and runs
`hb bootstrap`.

Useful Arch bootstrap options:

```bash
bash bootstrap/archlinux.sh --yes
bash bootstrap/archlinux.sh --install
bash bootstrap/archlinux.sh --repo git@github.com:you/dotfiles-arch.git
```

## Build From Source

```bash
git clone https://github.com/gin31259461/homebase.git ~/.local/lib/homebase
cd ~/.local/lib/homebase
go build -o ~/.local/bin/hb ./cmd/hb
```

On Windows:

```powershell
git clone https://github.com/gin31259461/homebase.git ~/.local/lib/homebase
Set-Location ~/.local/lib/homebase
go build -o ~/.local/bin/hb.exe ./cmd/hb
```

Verify the binary:

```bash
hb help
```

## Commands

```text
hb bootstrap [--yes] [--repo <repo>] [--install]
hb install   [--group <key>] [--all] [--yes] [--no-setup]
hb cleanup   [--task <key>] [--all] [--yes]
hb sync      [-m <message>] [--no-push]
hb config init [-f|--force]
```

Interactive commands use Bubble Tea by default. For unattended runs, pass
explicit selections with `--yes`.

## Bootstrap

`hb bootstrap` configures the current machine after the shell bootstrap has
built the binary.

```bash
hb bootstrap
hb bootstrap --yes
hb bootstrap --repo git@github.com:you/dotfiles.git
hb bootstrap --repo you/dotfiles --install
```

Bootstrap behavior:

1. Ensures global config and detected platform config exist
2. Installs configured bootstrap basics
3. Clones the dotfiles repo as a bare repo at `~/.dotfiles`
4. Copies tracked files into `$HOME`
5. Stores the selected dotfiles remote in `~/.dotfiles-repo`
6. Sets `status.showUntrackedFiles = no`
7. Initializes Git submodules
8. Runs platform-specific shell/profile setup
9. Optionally runs `hb install`

## Install Packages

Run the interactive selector:

```bash
hb install
```

Interactive lists highlight calculated state values:

- green: all installed or no cleanup work detected
- yellow: partly installed, unknown, or small cleanup work
- red: not installed or reclaimable cleanup work detected

Item titles stay neutral. The highlighted value is the calculated package
ratio, cleanup size, or reclaimable summary shown below the title.

List controls:

```text
j/k, arrows  move
gg, G        jump to top or bottom
space        toggle the current item
a            toggle all items
i            inspect the current item
ctrl+d/u     scroll inspect details
enter        confirm
q, esc       exit
```

Install selected groups:

```bash
hb install --group core --group cli
hb install --group fonts --yes
hb install --all --yes
```

Skip setup features while still installing packages:

```bash
hb install --group core --yes --no-setup
```

Default Windows groups include:

- `core`: Scoop, PowerShell, PSReadLine, Node.js, and pnpm
- `cli`: Starship, Neovim, ripgrep, WezTerm, and Lua through WinGet
- `apps`: Notion and Obsidian through WinGet
- `setup`: PowerShell profile and WezTerm context menu setup
- `fonts`: `nerd-fonts` Scoop bucket and `FiraCode-NF`
- `classic-menu`: Windows 10 classic context menu registry setup

Arch groups are defined under `config/platforms/archlinux/packages.d/` and
cover Hyprland, shell tools, desktop components, apps, hardware, and dev tools.

Runtime package groups live in:

```text
~/.config/homebase/platforms/<platform>/packages.d/*.toml
```

## Cleanup

Run the interactive cleanup selector:

```bash
hb cleanup
```

Run selected tasks:

```bash
hb cleanup --task scoop-cache --task temp-files
hb cleanup --all --yes
```

Runtime cleanup tasks live in:

```text
~/.config/homebase/platforms/<platform>/cleanup.toml
```

Windows cleanup tasks include Scoop cache, temp files, npm cache, WinGet cache,
Recycle Bin, and thumbnail cache. Arch cleanup tasks include pacman cache,
yay cache, orphaned packages, journal cleanup, npm cache, and thumbnails.

Arch pacman cache cleanup shows the amount reclaimable by `paccache -r`, not
the full `/var/cache/pacman/pkg` size. Inspect details include the total cache
size for context. Arch journal cleanup shows the amount over the 100 MiB
vacuum target, and npm cache cleanup scans npm's `_cacache` payload rather than
logs or other metadata under the npm cache root.

Arch orphan cleanup is intentionally conservative. The selector inspect view
shows the package count, installed size when known, review guidance, and the
full package list. Runtime removal prints the package list again, explains how
to keep a package with `sudo pacman -D --asexplicit <package>`, asks an
orphan-specific confirmation, and then runs `sudo pacman -Rns <packages...>`
without `--noconfirm` so pacman can show its own transaction prompt.

## Sync Dotfiles

`hb sync` stages configured paths from `$HOME`, commits them in the bare
dotfiles repo, and pushes to the configured branch.

```bash
hb sync
hb sync -m "chore: sync dotfiles"
hb sync -m "chore: sync dotfiles" --no-push
```

When prompted for a commit message, an empty value exits without staging,
committing, or pushing.

Tracked paths live in:

```text
~/.config/homebase/platforms/<platform>/sync.toml
```

Homebase uses the same bare Git layout as the `dot` alias:

```bash
git --git-dir=$HOME/.dotfiles/ --work-tree=$HOME
```

## Configuration

Default config in this repository:

```text
config/
|-- homebase.toml
`-- platforms/
    |-- archlinux/
    |   |-- config.toml
    |   |-- cleanup.toml
    |   |-- sync.toml
    |   `-- packages.d/
    `-- windows/
        |-- config.toml
        |-- cleanup.toml
        |-- sync.toml
        `-- packages.d/
```

At runtime, Homebase copies only the global config and the detected platform
defaults into `~/.config/homebase`.

Seed missing global config and active platform config without overwriting
existing files:

```bash
hb config init
```

Overwrite existing Homebase config from defaults:

```bash
hb config init --force
```

Global config:

```toml
active_platform = "auto"

[platform_aliases]
arch = "archlinux"
manjaro = "archlinux"
```

Platform config:

```toml
[dotfiles]
ssh_repo = "git@github.com:gin31259461/dotfiles-win.git"
https_repo = "https://github.com/gin31259461/dotfiles-win.git"
dir = "~/.dotfiles"
branch = "main"
memory_file = "~/.dotfiles-repo"

[package_manager]
official = "winget"

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

Use only the fields that apply to the target platform.
Set `default = true` on a package group only when it should be preselected by
the interactive install selector. Explicit `--group` and `--all` arguments
ignore interactive defaults.

Cleanup task fields:

```toml
[cache]
label = "Cache"
default = false
detail = "safe cleanup command summary"
requires = "optional-command"
sudo = false
```

Set `default = true` on a cleanup task only when it should be preselected by
the interactive cleanup selector. Explicit `--task` and `--all` arguments ignore
interactive defaults.

The dotfiles memory file supports TOML:

```toml
repo = "git@github.com:gin31259461/dotfiles-win.git"
branch = "main"
```

## Development

Run the standard checks:

```bash
gofmt -w cmd internal
go test ./...
go vet ./...
go build -o ~/.local/bin/hb ./cmd/hb
markdownlint-cli2 README.md
```

The Makefile wraps the same workflow:

```bash
make check
make build
make lint
```

Smoke-check the CLI:

```bash
hb help
hb install --group __none__ --yes --no-setup
```

## Project Layout

```text
.
|-- cmd/hb/              # thin CLI router
|-- bootstrap/           # platform bootstrap entrypoints
|-- config/              # default runtime config
|-- internal/bootstrap/  # shared bootstrap flow
|-- internal/cleanup/    # shared Arch cleanup flow
|-- internal/config/     # TOML loading and config seeding
|-- internal/gitutil/    # bare Git repo helpers
|-- internal/install/    # shared Arch install planning
|-- internal/platform/   # platform registry and implementations
|-- internal/run/        # command runner abstraction
|-- internal/setup/      # post-install setup routines
|-- internal/sync/       # dotfiles sync command
|-- internal/system/     # OS and command helpers
|-- internal/testutil/   # test fakes
`-- internal/ui/         # prompts, selector, and spinner
```

Platform implementations:

```text
internal/platform/archlinux/
internal/platform/windows/
```

## Troubleshooting

If `hb` is missing, rebuild it:

```bash
cd ~/.local/lib/homebase
go build -o ~/.local/bin/hb ./cmd/hb
```

On Windows:

```powershell
Set-Location ~/.local/lib/homebase
go build -o ~/.local/bin/hb.exe ./cmd/hb
```

If config is missing:

```bash
hb config init
```

If config should be reset from defaults:

```bash
hb config init --force
```

If Windows cannot find `hb`, open a new terminal or confirm that this directory
is in the user `Path`:

```text
%USERPROFILE%\.local\bin
```

If Arch AUR installs fail, verify that the bootstrap dependencies and
configured AUR helper are available:

```bash
pacman -Qi base-devel git
command -v yay
```
