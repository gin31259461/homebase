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
	commitMessage, ok := resolveCommitMessage(*msg, func() string {
		return ui.PromptText("Commit message", "")
	})
	if !ok {
		ui.OK("No commit message provided; nothing changed")
		return nil
	}

	ui.Section("Sync")
	workTree := config.Expand("~")
	addArgs := append(gitutil.DotArgs(cfg, "add", "--all", "--"), paths...)
	if err := r.RunIn(workTree, addArgs[0], addArgs[1:]...); err != nil {
		return err
	}
	diffArgs := gitutil.DotArgs(cfg, "diff", "--cached", "--quiet")
	if err := r.QuietIn(workTree, diffArgs[0], diffArgs[1:]...); err == nil {
		ui.OK("No staged changes")
		return nil
	}
	commitArgs := gitutil.DotArgs(cfg, "commit", "-m", commitMessage)
	if err := r.RunIn(workTree, commitArgs[0], commitArgs[1:]...); err != nil {
		return err
	}
	ui.OK("Committed: " + commitMessage)
	if !*noPush {
		pushArgs := gitutil.DotArgs(cfg, "push", "origin", cfg.Dotfiles.Branch)
		if err := r.RunIn(workTree, pushArgs[0], pushArgs[1:]...); err != nil {
			return err
		}
		ui.OK("Pushed to origin " + cfg.Dotfiles.Branch)
	}
	return nil
}

func resolveCommitMessage(flagValue string, prompt func() string) (string, bool) {
	message := strings.TrimSpace(flagValue)
	if message == "" {
		message = strings.TrimSpace(prompt())
	}
	if message == "" {
		return "", false
	}
	return message, true
}
