package sync

import (
	"errors"
	"flag"
	"strings"

	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/gitutil"
	"github.com/gin31259461/homebase/internal/run"
	"github.com/gin31259461/homebase/internal/ui"
)

func Run(args []string) error {
	return RunWithPlatform(args, run.New(), "archlinux")
}

func RunWithPlatform(args []string, r run.Runner, platformID string) error {
	fs := flag.NewFlagSet("sync", flag.ContinueOnError)
	msg := fs.String("m", "", "commit message")
	noPush := fs.Bool("no-push", false, "commit without pushing")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := config.EnsureForPlatform(platformID, false); err != nil {
		return err
	}
	cfg, err := config.LoadForPlatform(platformID)
	if err != nil {
		return err
	}
	paths, err := config.LoadSyncPathsForPlatform(platformID)
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return errors.New("sync config has no paths")
	}
	if strings.TrimSpace(*msg) == "" {
		*msg = ui.PromptText("Commit message", "sync dotfiles")
	}

	ui.Section("Sync")
	addArgs := append(gitutil.DotArgs(cfg, "add", "--all", "--"), paths...)
	if err := r.Run(addArgs[0], addArgs[1:]...); err != nil {
		return err
	}
	diffArgs := gitutil.DotArgs(cfg, "diff", "--cached", "--quiet")
	if err := r.Quiet(diffArgs[0], diffArgs[1:]...); err == nil {
		ui.OK("No staged changes")
		return nil
	}
	commitArgs := gitutil.DotArgs(cfg, "commit", "-m", *msg)
	if err := r.Run(commitArgs[0], commitArgs[1:]...); err != nil {
		return err
	}
	ui.OK("Committed: " + *msg)
	if !*noPush {
		pushArgs := gitutil.DotArgs(cfg, "push", "origin", cfg.Dotfiles.Branch)
		if err := r.Run(pushArgs[0], pushArgs[1:]...); err != nil {
			return err
		}
		ui.OK("Pushed to origin " + cfg.Dotfiles.Branch)
	}
	return nil
}
