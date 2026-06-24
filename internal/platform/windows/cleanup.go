package windows

import (
	"flag"
	"fmt"

	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/run"
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
	tasks, err := config.LoadCleanupTasksForPlatform(ID, commandExists)
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
	selected = uniqueKnown(selected, cleanupTaskSet(tasks))
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

func runCleanupTask(r run.Runner, key string) error {
	ui.Section(key)
	switch key {
	case "scoop-cache":
		if !scoopAvailable() {
			ui.Warn("scoop not found; skipping")
			return nil
		}
		return runCleanupCommand("Clearing Scoop package cache", func() error {
			return r.Run("scoop", "cache", "rm", "*")
		})
	case "temp-files":
		return runCleanupCommand("Removing Windows Temp files", func() error {
			return r.Run(powerShellExe(), "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", "Get-ChildItem $env:TEMP -ErrorAction SilentlyContinue | Remove-Item -Recurse -Force -ErrorAction SilentlyContinue")
		})
	case "npm-cache":
		if !commandExists("npm") {
			ui.Warn("npm not found; skipping")
			return nil
		}
		return runCleanupCommand("Cleaning npm cache", func() error {
			return r.Run("npm", "cache", "clean", "--force")
		})
	case "winget-cache":
		cmd := "$paths = @((Join-Path $env:LOCALAPPDATA 'Packages\\Microsoft.DesktopAppInstaller_8wekyb3d8bbwe\\LocalCache'), (Join-Path $env:TEMP 'WinGet')); foreach ($path in $paths) { if (Test-Path $path) { Remove-Item $path -Recurse -Force -ErrorAction SilentlyContinue } }"
		return runCleanupCommand("Clearing WinGet download cache", func() error {
			return r.Run(powerShellExe(), "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", cmd)
		})
	case "recycle-bin":
		return runCleanupCommand("Clearing Recycle Bin", func() error {
			return r.Run(powerShellExe(), "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", "Clear-RecycleBin -Force -ErrorAction SilentlyContinue")
		})
	case "thumbnail-cache":
		cmd := "$explorer = Get-Process -Name explorer -ErrorAction SilentlyContinue; if ($explorer) { Stop-Process -Name explorer -Force -ErrorAction SilentlyContinue; Start-Sleep -Seconds 1 }; $dir = Join-Path $env:LOCALAPPDATA 'Microsoft\\Windows\\Explorer'; if (Test-Path $dir) { Get-ChildItem $dir -Filter 'thumbcache_*.db' -ErrorAction SilentlyContinue | Remove-Item -Force -ErrorAction SilentlyContinue }; if ($explorer) { Start-Process explorer.exe }"
		return runCleanupCommand("Clearing thumbnail cache", func() error {
			return r.Run(powerShellExe(), "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", cmd)
		})
	default:
		return fmt.Errorf("no runner for cleanup task: %s", key)
	}
}

func runCleanupCommand(message string, fn func() error) error {
	return ui.WithSpinner(message, fn)
}
