package archlinux

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
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

const journalVacuumTargetBytes int64 = 100 * 1024 * 1024

func cleanupItems(r run.Runner, tasks []config.CleanupTask) []ui.SelectItem {
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
	state := sharedcleanup.CleanupSizeState(reclaimable)
	inspect := []string{"Reclaimable by paccache -r: " + sharedcleanup.FormatBytes(reclaimable)}
	if totalOK {
		inspect = append(inspect, "Total cache: "+sharedcleanup.FormatBytes(total))
	}
	inspect = append(inspect, "Path: "+path)
	return cleanupItemInfo{
		state:   state,
		summary: sharedcleanup.FormatBytes(reclaimable) + " reclaimable",
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
	state := sharedcleanup.CleanupSizeState(bytes)
	return cleanupItemInfo{
		state:   state,
		summary: sharedcleanup.FormatBytes(bytes),
		inspect: []string{label + ": " + sharedcleanup.FormatBytes(bytes), "Path: " + path},
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
		summary += ", " + sharedcleanup.FormatBytes(size)
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
		state := sharedcleanup.CleanupSizeState(reclaimable)
		return cleanupItemInfo{
			state:   state,
			summary: sharedcleanup.FormatBytes(reclaimable) + " over " + sharedcleanup.FormatBytes(journalVacuumTargetBytes) + " target",
			inspect: []string{
				"Total journal usage: " + sharedcleanup.FormatBytes(bytes),
				"Vacuum target: " + sharedcleanup.FormatBytes(journalVacuumTargetBytes),
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
	root := sharedcleanup.NPMCacheRoot(r, "~/.npm")
	path := sharedcleanup.NPMCachePayloadPath(root)
	info := dirCleanupInfo(r, path, "npm content-addressable cache")
	info.inspect = append(info.inspect, "npm cache root: "+root)
	return info
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
		lines = append(lines, "Installed size: "+sharedcleanup.FormatBytes(size))
	}
	lines = append(lines, "Packages:")
	lines = append(lines, pkgs...)
	return lines
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
