package cleanup

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
		selected, err = ui.SelectKeys("Cleanup Tasks", CleanupItems(r, tasks))
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

func CleanupItems(r run.Runner, tasks []config.CleanupTask) []ui.SelectItem {
	var items []ui.SelectItem
	for _, task := range tasks {
		info := cleanupInfo(r, task)
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

type cleanupItemInfo struct {
	state   ui.SelectState
	summary string
	inspect []string
}

const smallCleanupBytes int64 = 10 * 1024 * 1024
const journalVacuumTargetBytes int64 = 100 * 1024 * 1024

func cleanupInfo(r run.Runner, task config.CleanupTask) cleanupItemInfo {
	switch task.Key {
	case "pacman-cache":
		return pacmanCacheCleanupInfo(r, "/var/cache/pacman/pkg")
	case "yay-cache":
		return dirCleanupInfo(r, config.Expand("~/.cache/yay"), "AUR build cache")
	case "orphans":
		return orphanCleanupInfo(r)
	case "journal":
		return journalCleanupInfo(r)
	case "npm-cache":
		return npmCacheCleanupInfo(r)
	case "thumbnails":
		return dirCleanupInfo(r, config.Expand("~/.cache/thumbnails"), "Thumbnail cache")
	default:
		return cleanupItemInfo{
			state:   ui.SelectStateUnknown,
			summary: "size unknown",
			inspect: []string{"No scanner is implemented for this task yet."},
		}
	}
}

func pacmanCacheCleanupInfo(r run.Runner, path string) cleanupItemInfo {
	total, totalOK := dirSize(r, path)
	reclaimable, reclaimableOK := paccacheReclaimableSize(r, path)
	if !reclaimableOK {
		return dirCleanupInfo(r, path, "Pacman package cache")
	}
	state := cleanupSizeState(reclaimable)
	inspect := []string{"Reclaimable by paccache -r: " + formatBytes(reclaimable)}
	if totalOK {
		inspect = append(inspect, "Total cache: "+formatBytes(total))
	}
	inspect = append(inspect, "Path: "+path)
	return cleanupItemInfo{
		state:   state,
		summary: formatBytes(reclaimable) + " reclaimable",
		inspect: inspect,
	}
}

func dirCleanupInfo(r run.Runner, path, label string) cleanupItemInfo {
	bytes, ok := dirSize(r, path)
	if !ok {
		return cleanupItemInfo{
			state:   ui.SelectStateUnknown,
			summary: "size unknown",
			inspect: []string{label + ": size could not be read", "Path: " + path},
		}
	}
	state := cleanupSizeState(bytes)
	return cleanupItemInfo{
		state:   state,
		summary: formatBytes(bytes),
		inspect: []string{label + ": " + formatBytes(bytes), "Path: " + path},
	}
}

func paccacheReclaimableSize(r run.Runner, path string) (int64, bool) {
	out, err := r.Capture("paccache", "-dq", "-c", path)
	if err != nil {
		return 0, false
	}
	candidates := strings.Fields(out)
	if len(candidates) == 0 {
		return 0, true
	}
	var total int64
	for _, candidate := range candidates {
		info, err := os.Stat(candidate)
		if err != nil || info.IsDir() {
			continue
		}
		total += info.Size()
	}
	return total, true
}

func orphanCleanupInfo(r run.Runner) cleanupItemInfo {
	out, err := r.Capture("pacman", "-Qdtq")
	if err != nil {
		if strings.TrimSpace(out) == "" {
			return noOrphanCleanupInfo()
		}
		return cleanupItemInfo{
			state:   ui.SelectStateUnknown,
			summary: "orphan count unknown",
			inspect: []string{"pacman -Qdtq failed; the cleanup task can still run."},
		}
	}
	pkgs := strings.Fields(out)
	if len(pkgs) == 0 {
		return noOrphanCleanupInfo()
	}
	size, ok := packagesInstalledSize(r, pkgs)
	summary := fmt.Sprintf("%d orphaned package(s)", len(pkgs))
	if ok {
		summary += ", " + formatBytes(size)
	}
	return cleanupItemInfo{
		state:   ui.SelectStateBad,
		summary: summary,
		inspect: orphanInspect(pkgs, size, ok),
	}
}

func noOrphanCleanupInfo() cleanupItemInfo {
	return cleanupItemInfo{
		state:   ui.SelectStateGood,
		summary: "no orphaned packages",
		inspect: []string{"pacman -Qdtq returned no packages."},
	}
}

func journalCleanupInfo(r run.Runner) cleanupItemInfo {
	out, err := r.Capture("journalctl", "--disk-usage")
	text := strings.TrimSpace(out)
	if err != nil || text == "" {
		return cleanupItemInfo{
			state:   ui.SelectStateUnknown,
			summary: "journal size unknown",
			inspect: []string{"journalctl --disk-usage failed; the cleanup task can still run."},
		}
	}
	if bytes, ok := parseSize(text); ok {
		reclaimable := journalReclaimableBytes(bytes)
		state := cleanupSizeState(reclaimable)
		return cleanupItemInfo{
			state:   state,
			summary: formatBytes(reclaimable) + " over " + formatBytes(journalVacuumTargetBytes) + " target",
			inspect: []string{
				"Total journal usage: " + formatBytes(bytes),
				"Vacuum target: " + formatBytes(journalVacuumTargetBytes),
				text,
			},
		}
	}
	return cleanupItemInfo{
		state:   ui.SelectStateUnknown,
		summary: text,
		inspect: []string{text},
	}
}

func journalReclaimableBytes(bytes int64) int64 {
	reclaimable := bytes - journalVacuumTargetBytes
	if reclaimable < 0 {
		return 0
	}
	return reclaimable
}

func npmCacheCleanupInfo(r run.Runner) cleanupItemInfo {
	root := npmCacheRoot(r)
	path := filepath.Join(root, "_cacache")
	info := dirCleanupInfo(r, path, "npm content-addressable cache")
	info.inspect = append(info.inspect, "npm cache root: "+root)
	return info
}

func npmCacheRoot(r run.Runner) string {
	out, err := r.Capture("npm", "config", "get", "cache")
	if err == nil {
		root := strings.TrimSpace(out)
		if root != "" && root != "undefined" && root != "null" {
			return config.Expand(root)
		}
	}
	return config.Expand("~/.npm")
}

func cleanupSizeState(bytes int64) ui.SelectState {
	switch {
	case bytes == 0:
		return ui.SelectStateGood
	case bytes < smallCleanupBytes:
		return ui.SelectStatePartial
	default:
		return ui.SelectStateBad
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

func dirSize(r run.Runner, path string) (int64, bool) {
	if bytes, ok := duSize(r, path); ok {
		return bytes, true
	}
	var total int64
	err := filepath.WalkDir(path, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
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

func duSize(r run.Runner, path string) (int64, bool) {
	out, _ := r.Capture("du", "-sb", path)
	fields := strings.Fields(out)
	if len(fields) == 0 {
		return 0, false
	}
	bytes, err := strconv.ParseInt(fields[0], 10, 64)
	if err != nil {
		return 0, false
	}
	return bytes, true
}

func packagesInstalledSize(r run.Runner, pkgs []string) (int64, bool) {
	if len(pkgs) == 0 {
		return 0, true
	}
	args := append([]string{"-Qi"}, pkgs...)
	out, err := r.Capture("pacman", args...)
	if err != nil {
		return 0, false
	}
	var total int64
	found := false
	for _, line := range strings.Split(out, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "Installed Size") {
			continue
		}
		if bytes, ok := parseSize(line); ok {
			total += bytes
			found = true
		}
	}
	return total, found
}

func orphanInspect(pkgs []string, size int64, sizeKnown bool) []string {
	lines := []string{
		"Review before removal.",
		"Keep a package: sudo pacman -D --asexplicit <package>",
		"Removal uses pacman confirmation; --noconfirm is not used.",
		fmt.Sprintf("Package count: %d", len(pkgs)),
	}
	if sizeKnown {
		lines = append(lines, "Installed size: "+formatBytes(size))
	}
	lines = append(lines, "Packages:")
	lines = append(lines, pkgs...)
	return lines
}

func formatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	value := float64(bytes)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d %s", bytes, units[unit])
	}
	return fmt.Sprintf("%.1f %s", value, units[unit])
}

var sizePattern = regexp.MustCompile(`(?i)([0-9]+(?:\.[0-9]+)?)\s*([kmgt]i?b?|bytes?|b)`)

func parseSize(text string) (int64, bool) {
	match := sizePattern.FindStringSubmatch(text)
	if len(match) != 3 {
		return 0, false
	}
	value, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0, false
	}
	unit := strings.ToLower(match[2])
	multiplier := float64(1)
	switch unit {
	case "k", "kb", "kib":
		multiplier = 1024
	case "m", "mb", "mib":
		multiplier = 1024 * 1024
	case "g", "gb", "gib":
		multiplier = 1024 * 1024 * 1024
	case "t", "tb", "tib":
		multiplier = 1024 * 1024 * 1024 * 1024
	}
	return int64(value * multiplier), true
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
		return r.Run("npm", "cache", "clean", "--force")
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
