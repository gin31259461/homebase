package archlinux

import (
	"fmt"
	"os"
	"strings"

	sharedcleanup "github.com/gin31259461/homebase/internal/cleanup"
	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/run"
	"github.com/gin31259461/homebase/internal/system"
	"github.com/gin31259461/homebase/internal/ui"
)

func runCleanupTask(r run.Runner, key string) error {
	ui.Section(key)
	switch key {
	case "pacman-cache":
		if !system.CommandExists("paccache") {
			return fmt.Errorf("paccache not found; install pacman-contrib")
		}
		return r.Run("sudo", "paccache", "-r")
	case "yay-cache":
		return os.RemoveAll(config.Expand("~/.cache/yay"))
	case "orphans":
		pkgs, err := orphanPackages(r)
		if err != nil {
			return err
		}
		if len(pkgs) == 0 {
			ui.OK("No orphaned packages found")
			return nil
		}
		printOrphanRemovalReview(pkgs)
		if !ui.Confirm("Remove these orphan packages with pacman -Rns?", false) {
			ui.Warn("Skipped orphan package removal")
			return nil
		}
		return r.Run("sudo", orphanRemovalArgs(pkgs)...)
	case "journal":
		return r.Run("sudo", "journalctl", "--vacuum-size=100M")
	case "npm-cache":
		return sharedcleanup.RunNPMCacheClean(r)
	case "thumbnails":
		return os.RemoveAll(config.Expand("~/.cache/thumbnails"))
	default:
		return fmt.Errorf("no runner for cleanup task: %s", key)
	}
}

func orphanPackages(r run.Runner) ([]string, error) {
	out, err := r.Capture("pacman", "-Qdtq")
	if err != nil {
		if strings.TrimSpace(out) == "" {
			return nil, nil
		}
		return nil, err
	}
	return strings.Fields(out), nil
}

func orphanRemovalArgs(pkgs []string) []string {
	return append([]string{"pacman", "-Rns"}, pkgs...)
}

func printOrphanRemovalReview(pkgs []string) {
	ui.Warn("Review orphan packages before removal")
	ui.Note("To keep one, abort and run: sudo pacman -D --asexplicit <package>")
	for _, pkg := range pkgs {
		fmt.Printf("  %s %s\n", ui.BadStyle.Render("-"), pkg)
	}
}
