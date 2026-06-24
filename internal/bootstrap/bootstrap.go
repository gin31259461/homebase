package bootstrap

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/gitutil"
	"github.com/gin31259461/homebase/internal/install"
	"github.com/gin31259461/homebase/internal/run"
	"github.com/gin31259461/homebase/internal/system"
	"github.com/gin31259461/homebase/internal/ui"
)

func Run(args []string) error {
	return RunWith(args, run.New())
}

func RunWith(args []string, r run.Runner) error {
	fs := flag.NewFlagSet("bootstrap", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "accept defaults and skip prompts")
	fs.BoolVar(yes, "y", false, "accept defaults and skip prompts")
	repo := fs.String("repo", "", "dotfiles repository URL")
	runInstall := fs.Bool("install", false, "run package installer after bootstrap")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if err := system.RequireArch(); err != nil {
		return err
	}
	if err := config.Ensure(false); err != nil {
		return err
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	ui.Section("Bootstrap")
	if err := installBasics(r, cfg.Bootstrap.Basics); err != nil {
		return err
	}

	effectiveSSH, effectiveHTTPS, err := resolveDotfilesRepo(cfg, *repo, !*yes)
	if err != nil {
		return err
	}
	if err := deployDotfiles(r, cfg, effectiveSSH, effectiveHTTPS); err != nil {
		return err
	}
	if err := configureDotfiles(r, cfg); err != nil {
		return err
	}
	if err := installOhMyZsh(r, *yes); err != nil {
		return err
	}
	if *runInstall {
		argv := []string{}
		if *yes {
			argv = append(argv, "--all", "--yes")
		}
		if err := install.RunWith(argv, r); err != nil {
			return err
		}
	} else {
		ui.Note("Package installation skipped. Run hb install when ready.")
	}

	ui.Section("Done")
	ui.OK("Bootstrap complete")
	ui.Note("Restart your shell or run: exec zsh")
	ui.Note("Use hb sync to stage, commit, and push configured dotfile paths")
	return nil
}

func installBasics(r run.Runner, pkgs []string) error {
	if len(pkgs) == 0 {
		return nil
	}
	var missing []string
	for _, pkg := range pkgs {
		if !system.PacmanInstalled(r, pkg) {
			missing = append(missing, pkg)
		}
	}
	if len(missing) == 0 {
		ui.OK("Bootstrap packages already installed")
		return nil
	}
	ui.Section("Installing bootstrap packages")
	args := append([]string{"pacman", "-S", "--needed", "--noconfirm"}, missing...)
	return r.Run("sudo", args...)
}

func resolveDotfilesRepo(cfg config.App, flagRepo string, interactive bool) (string, string, error) {
	sshRepo := cfg.Dotfiles.SSHRepo
	httpsRepo := cfg.Dotfiles.HTTPSRepo
	if mem, err := gitutil.ReadRepoMemory(cfg.Dotfiles.MemoryFile); err == nil && mem.Repo != "" {
		if ssh, https, err := gitutil.NormalizeRepo(mem.Repo); err == nil {
			sshRepo, httpsRepo = ssh, https
		}
	}
	if flagRepo != "" {
		return gitutil.NormalizeRepo(flagRepo)
	}
	if interactive && !ui.Confirm("Use dotfiles repo "+sshRepo+"?", true) {
		input := ui.PromptText("Dotfiles repo", sshRepo)
		return gitutil.NormalizeRepo(input)
	}
	return sshRepo, httpsRepo, nil
}

func deployDotfiles(r run.Runner, cfg config.App, sshRepo, httpsRepo string) error {
	if _, err := os.Stat(cfg.Dotfiles.Dir); err == nil {
		ui.OK("Bare dotfiles repo already present at " + cfg.Dotfiles.Dir)
		if mem, err := gitutil.ReadRepoMemory(cfg.Dotfiles.MemoryFile); err != nil || mem.Repo == "" {
			remoteArgs := gitutil.DotArgs(cfg, "remote", "get-url", "origin")
			if remote, err := r.Capture(remoteArgs[0], remoteArgs[1:]...); err == nil && strings.TrimSpace(remote) != "" {
				_ = gitutil.SaveRepoMemory(cfg.Dotfiles.MemoryFile, strings.TrimSpace(remote), cfg.Dotfiles.Branch)
			}
		}
		return nil
	}

	cloneURL := sshRepo
	if strings.HasPrefix(sshRepo, "git@github.com:") && !gitutil.GitHubSSHAvailable(r) && httpsRepo != "" {
		ui.Warn("No GitHub SSH access detected; cloning over HTTPS")
		cloneURL = httpsRepo
	}
	if cloneURL == "" {
		return fmt.Errorf("no dotfiles repository URL resolved")
	}
	tmp, err := os.MkdirTemp("", "homebase-dotfiles-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	ui.Section("Dotfiles repository")
	worktree := filepath.Join(tmp, "dotfiles")
	if err := r.Run("git", "clone", "--separate-git-dir="+cfg.Dotfiles.Dir, cloneURL, worktree); err != nil {
		return err
	}
	if err := r.Run("rsync", "--recursive", "--exclude", ".git", worktree+"/", config.Expand("~")+"/"); err != nil {
		return err
	}
	if sshRepo != "" {
		remoteArgs := gitutil.DotArgs(cfg, "remote", "set-url", "origin", sshRepo)
		_ = r.Run(remoteArgs[0], remoteArgs[1:]...)
		if err := gitutil.SaveRepoMemory(cfg.Dotfiles.MemoryFile, sshRepo, cfg.Dotfiles.Branch); err != nil {
			return err
		}
	}
	ui.OK("Dotfiles deployed")
	return config.Ensure(false)
}

func configureDotfiles(r run.Runner, cfg config.App) error {
	ui.Section("Configuration")
	configArgs := gitutil.DotArgs(cfg, "config", "--local", "status.showUntrackedFiles", "no")
	if err := r.Run(configArgs[0], configArgs[1:]...); err != nil {
		return err
	}
	zshrc := config.Expand("~/.zshrc")
	aliasLine := "alias dot='git --git-dir=$HOME/.dotfiles/ --work-tree=$HOME'"
	if b, err := os.ReadFile(zshrc); err == nil && strings.Contains(string(b), "alias dot=") {
		ui.OK("dot alias already present")
	} else {
		f, err := os.OpenFile(zshrc, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		if _, err := fmt.Fprintf(f, "\n# dotfiles bare repo\n%s\n", aliasLine); err != nil {
			f.Close()
			return err
		}
		if err := f.Close(); err != nil {
			return err
		}
		ui.OK("dot alias added to .zshrc")
	}
	submoduleArgs := gitutil.DotArgs(cfg, "submodule", "update", "--init", "--recursive")
	if err := r.Run(submoduleArgs[0], submoduleArgs[1:]...); err != nil {
		return err
	}
	ui.OK("Submodules ready")
	return nil
}

func installOhMyZsh(r run.Runner, yes bool) error {
	ui.Section("Oh My Zsh")
	if _, err := os.Stat(config.Expand("~/.oh-my-zsh")); err == nil {
		ui.OK("Oh My Zsh already installed")
		return nil
	}
	if !yes && !ui.Confirm("Install Oh My Zsh?", true) {
		ui.Warn("Skipped Oh My Zsh")
		return nil
	}
	if err := r.Run("sh", "-c", `RUNZSH=no KEEP_ZSHRC=yes sh -c "$(curl -fsSL https://raw.githubusercontent.com/ohmyzsh/ohmyzsh/master/tools/install.sh)"`); err != nil {
		return err
	}
	custom := os.Getenv("ZSH_CUSTOM")
	if custom == "" {
		custom = config.Expand("~/.oh-my-zsh/custom")
	}
	repos := []struct {
		url string
		dir string
	}{
		{"https://github.com/zsh-users/zsh-autosuggestions", filepath.Join(custom, "plugins/zsh-autosuggestions")},
		{"https://github.com/zsh-users/zsh-syntax-highlighting", filepath.Join(custom, "plugins/zsh-syntax-highlighting")},
		{"https://github.com/romkatv/powerlevel10k.git", filepath.Join(custom, "themes/powerlevel10k")},
	}
	for _, repo := range repos {
		if _, err := os.Stat(repo.dir); err == nil {
			continue
		}
		if err := r.Run("git", "clone", "--depth=1", repo.url, repo.dir); err != nil {
			return err
		}
	}
	ui.OK("Oh My Zsh plugins and theme ready")
	return nil
}
