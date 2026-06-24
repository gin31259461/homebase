package windows

import (
	"errors"
	"flag"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/gitutil"
	"github.com/gin31259461/homebase/internal/run"
	"github.com/gin31259461/homebase/internal/ui"
)

func runBootstrap(args []string, r run.Runner) error {
	fs := flag.NewFlagSet("bootstrap", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "accept defaults and skip prompts")
	fs.BoolVar(yes, "y", false, "accept defaults and skip prompts")
	repo := fs.String("repo", "", "dotfiles repository URL")
	installAfterBootstrap := fs.Bool("install", false, "run package installer after bootstrap")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if err := config.EnsureForPlatform(ID, false); err != nil {
		return err
	}
	cfg, err := config.LoadForPlatform(ID)
	if err != nil {
		return err
	}

	ui.Section("Bootstrap")
	if err := ensureUserPath(r, hbBinDir()); err != nil {
		return err
	}
	if err := installBootstrapBasics(r, cfg.Bootstrap.Basics); err != nil {
		return err
	}
	sshRepo, httpsRepo, err := resolveDotfilesRepo(cfg, *repo, !*yes)
	if err != nil {
		return err
	}
	if err := deployDotfiles(r, cfg, sshRepo, httpsRepo); err != nil {
		return err
	}
	if err := configureDotfiles(r, cfg); err != nil {
		return err
	}
	if err := linkPowerShellProfiles(r); err != nil {
		return err
	}
	if *installAfterBootstrap {
		argv := []string{}
		if *yes {
			argv = append(argv, "--all", "--yes")
		}
		if err := runInstall(argv, r); err != nil {
			return err
		}
	} else {
		ui.Note("Package installation skipped. Run hb install when ready.")
	}

	ui.Section("Done")
	ui.OK("Bootstrap complete")
	ui.Note("Open a new terminal to refresh PATH and profile links")
	ui.Note("Use hb sync to stage, commit, and push configured dotfile paths")
	return nil
}

func installBootstrapBasics(r run.Runner, basics []string) error {
	for _, basic := range basics {
		switch strings.ToLower(strings.TrimSpace(basic)) {
		case "", "git":
			if strings.TrimSpace(basic) == "" || commandExists("git") {
				continue
			}
			if err := installWingetPackage(r, "Git.Git"); err != nil {
				return err
			}
		case "go", "golang":
			if commandExists("go") {
				continue
			}
			if err := installWingetPackage(r, "GoLang.Go"); err != nil {
				return err
			}
		default:
			ui.Warn("Unknown Windows bootstrap basic: " + basic)
		}
	}
	return nil
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
		return rememberExistingRemote(r, cfg)
	}
	cloneURL := sshRepo
	if strings.HasPrefix(sshRepo, "git@github.com:") && !gitutil.RemoteHeadAvailable(r, sshRepo) && httpsRepo != "" {
		ui.Warn("No GitHub SSH access detected; cloning over HTTPS")
		cloneURL = httpsRepo
	}
	if cloneURL == "" {
		return errors.New("no dotfiles repository URL resolved")
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
	if err := copyTree(worktree, config.Expand("~")); err != nil {
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
	return config.EnsureForPlatform(ID, false)
}

func rememberExistingRemote(r run.Runner, cfg config.App) error {
	if mem, err := gitutil.ReadRepoMemory(cfg.Dotfiles.MemoryFile); err == nil && mem.Repo != "" {
		return nil
	}
	remoteArgs := gitutil.DotArgs(cfg, "remote", "get-url", "origin")
	remote, err := r.Capture(remoteArgs[0], remoteArgs[1:]...)
	if err != nil || strings.TrimSpace(remote) == "" {
		return nil
	}
	return gitutil.SaveRepoMemory(cfg.Dotfiles.MemoryFile, strings.TrimSpace(remote), cfg.Dotfiles.Branch)
}

func configureDotfiles(r run.Runner, cfg config.App) error {
	ui.Section("Configuration")
	configArgs := gitutil.DotArgs(cfg, "config", "--local", "status.showUntrackedFiles", "no")
	if err := r.Run(configArgs[0], configArgs[1:]...); err != nil {
		return err
	}
	submoduleArgs := gitutil.DotArgs(cfg, "submodule", "update", "--init", "--recursive")
	if err := r.Run(submoduleArgs[0], submoduleArgs[1:]...); err != nil {
		return err
	}
	ui.OK("Submodules ready")
	return nil
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil || rel == "." {
			return err
		}
		if rel == ".git" || strings.HasPrefix(rel, ".git"+string(os.PathSeparator)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}
