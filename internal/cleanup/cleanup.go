package cleanup

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/run"
	"github.com/gin31259461/homebase/internal/system"
	"github.com/gin31259461/homebase/internal/ui"
)

type stringList []string

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func Run(args []string) error {
	return RunWithPlatform(args, run.New(), "archlinux")
}

func RunWithPlatform(args []string, r run.Runner, platformID string) error {
	fs := flag.NewFlagSet("cleanup", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "skip confirmation")
	fs.BoolVar(yes, "y", false, "skip confirmation")
	all := fs.Bool("all", false, "select all cleanup tasks")
	var selectedFlags stringList
	fs.Var(&selectedFlags, "task", "cleanup task key, repeatable")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := config.EnsureForPlatform(platformID, false); err != nil {
		return err
	}
	tasks, err := config.LoadCleanupTasksForPlatform(platformID, system.CommandExists)
	if err != nil {
		return err
	}

	selected := append([]string(nil), selectedFlags...)
	if *all {
		for _, task := range tasks {
			selected = append(selected, task.Key)
		}
	}
	if len(selected) == 0 {
		selected, err = ui.SelectKeys("Cleanup Tasks", CleanupItems(tasks))
		if err != nil {
			return err
		}
	}
	selected = UniqueKnown(selected, CleanupTaskSet(tasks))
	if len(selected) == 0 {
		ui.Warn("No cleanup tasks selected")
		return nil
	}

	ui.Section("Cleanup plan")
	taskByKey := map[string]config.CleanupTask{}
	for _, task := range tasks {
		taskByKey[task.Key] = task
	}
	for _, key := range selected {
		task := taskByKey[key]
		fmt.Printf("  %s %s\n", ui.OKStyle.Render("+"), task.Label)
		fmt.Printf("    %s\n", ui.DimStyle.Render(task.Detail))
	}
	if !*yes && !ui.Confirm("Proceed with cleanup?", false) {
		ui.Warn("Aborted")
		return nil
	}
	for _, key := range selected {
		if taskByKey[key].Sudo {
			if err := r.Run("sudo", "-v"); err != nil {
				return err
			}
			break
		}
	}
	for _, key := range selected {
		if err := RunTask(r, key); err != nil {
			return err
		}
	}
	ui.OK("System cleanup complete")
	return nil
}

func CleanupItems(tasks []config.CleanupTask) []ui.SelectItem {
	var items []ui.SelectItem
	for _, task := range tasks {
		items = append(items, ui.SelectItem{Key: task.Key, Label: task.Label, Detail: task.Detail})
	}
	return items
}

func CleanupTaskSet(tasks []config.CleanupTask) map[string]bool {
	set := map[string]bool{}
	for _, task := range tasks {
		set[task.Key] = true
	}
	return set
}

func UniqueKnown(values []string, known map[string]bool) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || seen[value] {
			continue
		}
		if !known[value] {
			ui.Warn("Skipping unknown key: " + value)
			continue
		}
		out = append(out, value)
		seen[value] = true
	}
	return out
}

func RunTask(r run.Runner, key string) error {
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
		out, _ := r.Capture("pacman", "-Qdtq")
		pkgs := strings.Fields(out)
		if len(pkgs) == 0 {
			ui.OK("No orphaned packages found")
			return nil
		}
		args := append([]string{"pacman", "-Rns", "--noconfirm"}, pkgs...)
		return r.Run("sudo", args...)
	case "journal":
		return r.Run("sudo", "journalctl", "--vacuum-time=2weeks")
	case "npm-cache":
		return r.Run("npm", "cache", "clean", "--force")
	case "thumbnails":
		return os.RemoveAll(config.Expand("~/.cache/thumbnails"))
	default:
		return fmt.Errorf("no runner for cleanup task: %s", key)
	}
}
