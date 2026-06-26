package windows

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	sharedcleanup "github.com/gin31259461/homebase/internal/cleanup"
	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/run"
	"github.com/gin31259461/homebase/internal/ui"
)

type cleanupItemInfo struct {
	state   ui.SelectState
	summary string
	inspect []string
}

func cleanupItems(r run.Runner, tasks []config.CleanupTask) []ui.SelectItem {
	var items []ui.SelectItem
	if len(tasks) > 0 {
		_ = ui.WithSpinner("Scanning Windows cleanup state", func() error {
			items = buildCleanupItems(r, tasks)
			return nil
		})
	}
	return items
}

func buildCleanupItems(r run.Runner, tasks []config.CleanupTask) []ui.SelectItem {
	items := make([]ui.SelectItem, 0, len(tasks))
	for _, task := range tasks {
		info := windowsCleanupInfo(r, task)
		items = append(items, ui.SelectItem{
			Key:             task.Key,
			Label:           task.Label,
			DetailValue:     info.summary,
			Detail:          task.Detail,
			Inspect:         cleanupInspect(task, info),
			State:           info.state,
			DefaultSelected: task.Default,
		})
	}
	return items
}

func windowsCleanupInfo(r run.Runner, task config.CleanupTask) cleanupItemInfo {
	switch task.Key {
	case "scoop-cache":
		return dirCleanupInfo(filepath.Join(config.Expand("~"), "scoop", "cache"), "Scoop package cache")
	case "temp-files":
		return tempFilesCleanupInfo(r)
	case "npm-cache":
		return npmCacheCleanupInfo(r)
	case "winget-cache":
		return multiDirCleanupInfo("WinGet download cache", wingetCachePaths()...)
	case "recycle-bin":
		return recycleBinCleanupInfo(r)
	case "thumbnail-cache":
		return thumbnailCleanupInfo()
	default:
		return cleanupItemInfo{
			state:   ui.SelectStateUnknown,
			summary: "size unknown",
			inspect: []string{"No scanner is implemented for this task yet."},
		}
	}
}

func tempFilesCleanupInfo(r run.Runner) cleanupItemInfo {
	path := windowsTempDir()
	if commandExists(powerShellExe()) {
		out, err := r.Capture(powerShellExe(), "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", tempFilesSizeCommand())
		if err == nil {
			if bytes, ok := parseInt64Line(out); ok {
				return cleanupItemInfo{
					state:   sharedcleanup.CleanupSizeState(bytes),
					summary: sharedcleanup.FormatBytes(bytes),
					inspect: []string{"Windows Temp folder: " + sharedcleanup.FormatBytes(bytes), "Path: " + path},
				}
			}
		}
	}
	return dirCleanupInfo(path, "Windows Temp folder")
}

func tempFilesSizeCommand() string {
	return "$path = [IO.Path]::GetTempPath(); " +
		"$total = [int64]0; " +
		"if (Test-Path -LiteralPath $path) { " +
		"Get-ChildItem -LiteralPath $path -Force -Recurse -File -ErrorAction SilentlyContinue | ForEach-Object { $total += $_.Length } " +
		"}; " +
		"Write-Output $total"
}

func recycleBinCleanupInfo(r run.Runner) cleanupItemInfo {
	if !commandExists(powerShellExe()) {
		return cleanupItemInfo{
			state:   ui.SelectStateUnknown,
			summary: "size unknown",
			inspect: []string{"PowerShell is required to scan Recycle Bin size."},
		}
	}
	out, err := r.Capture(powerShellExe(), "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", recycleBinSizeCommand())
	if err != nil {
		return cleanupItemInfo{
			state:   ui.SelectStateUnknown,
			summary: "size unknown",
			inspect: []string{"Recycle Bin size scan failed; the cleanup task can still run."},
		}
	}
	bytes, ok := parseInt64Line(out)
	if !ok {
		return cleanupItemInfo{
			state:   ui.SelectStateUnknown,
			summary: "size unknown",
			inspect: []string{"Recycle Bin size output could not be parsed.", strings.TrimSpace(out)},
		}
	}
	return cleanupItemInfo{
		state:   sharedcleanup.CleanupSizeState(bytes),
		summary: sharedcleanup.FormatBytes(bytes),
		inspect: []string{"Recycle Bin: " + sharedcleanup.FormatBytes(bytes)},
	}
}

func recycleBinSizeCommand() string {
	return "$total = [int64]0; " +
		"Get-PSDrive -PSProvider FileSystem | ForEach-Object { " +
		"$path = Join-Path $_.Root '$Recycle.Bin'; " +
		"if (Test-Path -LiteralPath $path) { " +
		"Get-ChildItem -LiteralPath $path -Force -Recurse -File -ErrorAction SilentlyContinue | ForEach-Object { $total += $_.Length } " +
		"} }; " +
		"Write-Output $total"
}

func parseInt64Line(out string) (int64, bool) {
	lines := strings.Split(out, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		value, err := strconv.ParseInt(line, 10, 64)
		return value, err == nil
	}
	return 0, false
}

func npmCacheCleanupInfo(r run.Runner) cleanupItemInfo {
	root := sharedcleanup.NPMCacheRoot(r, windowsNPMCacheFallback())
	path := sharedcleanup.NPMCachePayloadPath(root)
	if path != "" {
		info := dirCleanupInfo(path, "npm content-addressable cache")
		info.inspect = append(info.inspect, "npm cache root: "+root)
		return info
	}
	return cleanupItemInfo{
		state:   ui.SelectStateUnknown,
		summary: "size unknown",
		inspect: []string{"npm cache path could not be resolved."},
	}
}

func windowsNPMCacheFallback() string {
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		return filepath.Join(local, "npm-cache")
	}
	return ""
}

func wingetCachePaths() []string {
	var paths []string
	if local := os.Getenv("LOCALAPPDATA"); local != "" {
		paths = append(paths, filepath.Join(local, "Packages", "Microsoft.DesktopAppInstaller_8wekyb3d8bbwe", "LocalCache"))
	}
	if temp := windowsTempDir(); temp != "" {
		paths = append(paths, filepath.Join(temp, "WinGet"))
	}
	return paths
}

func windowsTempDir() string {
	if temp := os.Getenv("TEMP"); temp != "" {
		return temp
	}
	if temp := os.Getenv("TMP"); temp != "" {
		return temp
	}
	return os.TempDir()
}

func thumbnailCleanupInfo() cleanupItemInfo {
	local := os.Getenv("LOCALAPPDATA")
	if local == "" {
		return cleanupItemInfo{
			state:   ui.SelectStateUnknown,
			summary: "size unknown",
			inspect: []string{"LOCALAPPDATA is not set."},
		}
	}
	dir := filepath.Join(local, "Microsoft", "Windows", "Explorer")
	bytes, ok := globSize(filepath.Join(dir, "thumbcache_*.db"))
	if !ok {
		return cleanupItemInfo{
			state:   ui.SelectStateUnknown,
			summary: "size unknown",
			inspect: []string{"Thumbnail cache size could not be read.", "Path: " + dir},
		}
	}
	return cleanupItemInfo{
		state:   sharedcleanup.CleanupSizeState(bytes),
		summary: sharedcleanup.FormatBytes(bytes),
		inspect: []string{"Thumbnail cache: " + sharedcleanup.FormatBytes(bytes), "Path: " + dir},
	}
}

func dirCleanupInfo(path, label string) cleanupItemInfo {
	bytes, ok := dirSize(path)
	if !ok {
		return cleanupItemInfo{
			state:   ui.SelectStateUnknown,
			summary: "size unknown",
			inspect: []string{label + ": size could not be read", "Path: " + path},
		}
	}
	return cleanupItemInfo{
		state:   sharedcleanup.CleanupSizeState(bytes),
		summary: sharedcleanup.FormatBytes(bytes),
		inspect: []string{label + ": " + sharedcleanup.FormatBytes(bytes), "Path: " + path},
	}
}

func multiDirCleanupInfo(label string, paths ...string) cleanupItemInfo {
	var total int64
	var inspect []string
	for _, path := range paths {
		bytes, ok := dirSize(path)
		if !ok {
			inspect = append(inspect, path+": size could not be read")
			continue
		}
		total += bytes
		inspect = append(inspect, path+": "+sharedcleanup.FormatBytes(bytes))
	}
	if len(inspect) == 0 {
		return cleanupItemInfo{
			state:   ui.SelectStateUnknown,
			summary: "size unknown",
			inspect: []string{label + " paths could not be resolved."},
		}
	}
	return cleanupItemInfo{
		state:   sharedcleanup.CleanupSizeState(total),
		summary: sharedcleanup.FormatBytes(total),
		inspect: append([]string{label + ": " + sharedcleanup.FormatBytes(total)}, inspect...),
	}
}

func cleanupInspect(task config.CleanupTask, info cleanupItemInfo) string {
	lines := []string{
		"Label: " + task.Label,
		"Command: " + task.Detail,
		fmt.Sprintf("Sudo: %t", task.Sudo),
	}
	lines = append(lines, info.inspect...)
	return strings.Join(lines, "\n")
}

func dirSize(path string) (int64, bool) {
	var total int64
	err := filepath.WalkDir(path, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		total += info.Size()
		return nil
	})
	if err == nil {
		return total, true
	}
	if os.IsNotExist(err) {
		return 0, true
	}
	return 0, false
}

func globSize(pattern string) (int64, bool) {
	files, err := filepath.Glob(pattern)
	if err != nil {
		return 0, false
	}
	var total int64
	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil || info.IsDir() {
			continue
		}
		total += info.Size()
	}
	return total, true
}
