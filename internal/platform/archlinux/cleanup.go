package archlinux

import (
	"flag"
	"fmt"

	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/install"
	"github.com/gin31259461/homebase/internal/run"
	"github.com/gin31259461/homebase/internal/system"
	"github.com/gin31259461/homebase/internal/ui"
)

func runCleanup(args []string, r run.Runner) error {
	fs := flag.NewFlagSet("cleanup", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "skip confirmation")
	fs.BoolVar(yes, "y", false, "skip confirmation")
	all := fs.Bool("all", false, "select all cleanup tasks")
	var selectedFlags stringList
	fs.Var(&selectedFlags, "task", "cleanup task key, repeatable")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := config.EnsureForPlatform(ID, false); err != nil {
		return err
	}
	tasks, err := config.LoadCleanupTasksForPlatform(ID, system.CommandExists)
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
		selected, err = ui.SelectKeys("Cleanup Tasks", cleanupItems(r, tasks))
		if err != nil {
			return err
		}
	}
	selected = install.UniqueKnown(selected, cleanupTaskSet(tasks))
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
		if err := runCleanupTask(r, key); err != nil {
			return err
		}
	}
	ui.OK("System cleanup complete")
	return nil
}

func cleanupTaskSet(tasks []config.CleanupTask) map[string]bool {
	set := map[string]bool{}
	for _, task := range tasks {
		set[task.Key] = true
	}
	return set
}
