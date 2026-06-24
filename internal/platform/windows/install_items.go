package windows

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/run"
	"github.com/gin31259461/homebase/internal/ui"
)

type itemStatus struct {
	label     string
	kind      string
	installed bool
	known     bool
}

type installItemInfo struct {
	state     ui.SelectState
	summary   string
	detail    string
	installed []string
	missing   []string
	unknown   []string
}

type installScan struct {
	wingetKnown  bool
	wingetIDs    map[string]bool
	featureKnown bool
	features     map[string]bool
}

func packageItems(r run.Runner, groups []config.PackageGroup) []ui.SelectItem {
	scan := scanWindowsInstallState(r, groups)
	items := make([]ui.SelectItem, 0, len(groups))
	for _, group := range groups {
		info := windowsInstallInfo(group, scan)
		items = append(items, ui.SelectItem{
			Key:             group.Key,
			Label:           group.Label,
			DetailValue:     info.summary,
			Detail:          info.detail,
			Inspect:         installInspect(group, info),
			State:           info.state,
			DefaultSelected: group.Default,
		})
	}
	return items
}

func scanWindowsInstallState(r run.Runner, groups []config.PackageGroup) installScan {
	scan := installScan{
		wingetIDs: map[string]bool{},
		features:  map[string]bool{},
	}
	if needsWingetScan(groups) && commandExists("winget") {
		out, err := r.Capture("winget", "list", "--disable-interactivity")
		if err == nil && strings.TrimSpace(out) != "" {
			scan.wingetKnown = true
			scan.wingetIDs = parseWingetListIDs(out)
		}
	}
	if needsFeatureScan(groups) && commandExists(powerShellExe()) {
		if features, ok := scanInstallFeatures(r); ok {
			scan.featureKnown = true
			scan.features = features
		}
	}
	return scan
}

func needsWingetScan(groups []config.PackageGroup) bool {
	for _, group := range groups {
		if len(group.Winget) > 0 {
			return true
		}
		for _, feature := range group.Features {
			if feature == "powershell" {
				return true
			}
		}
	}
	return false
}

func needsFeatureScan(groups []config.PackageGroup) bool {
	for _, group := range groups {
		for _, feature := range group.Features {
			if isScannableInstallFeature(feature) {
				return true
			}
		}
	}
	return false
}

func isScannableInstallFeature(feature string) bool {
	return feature == "wezterm-context-menu" || feature == "win10-classic-menu"
}

func scanInstallFeatures(r run.Runner) (map[string]bool, bool) {
	out, err := r.Capture(powerShellExe(), "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", installFeatureScanCommand())
	if err != nil {
		return nil, false
	}
	features := map[string]bool{
		"wezterm-context-menu": false,
		"win10-classic-menu":   false,
	}
	for _, line := range strings.Split(out, "\n") {
		key := strings.TrimSpace(line)
		if _, ok := features[key]; ok {
			features[key] = true
		}
	}
	return features, true
}

func installFeatureScanCommand() string {
	return "$found = @(); " +
		"if (Test-Path 'HKCU:\\Software\\Classes\\Directory\\background\\shell\\wezterm-nightly') { $found += 'wezterm-context-menu' }; " +
		"if (Test-Path 'HKCU:\\Software\\Classes\\CLSID\\{86ca1aa0-34aa-4e8b-a509-50c905bae2a2}\\InprocServer32') { $found += 'win10-classic-menu' }; " +
		"$found -join \"`n\""
}

func parseWingetListIDs(out string) map[string]bool {
	ids := map[string]bool{}
	for _, field := range strings.Fields(out) {
		if strings.Contains(field, ".") {
			ids[strings.ToLower(field)] = true
		}
	}
	return ids
}

func windowsInstallInfo(group config.PackageGroup, scan installScan) installItemInfo {
	statuses := installStatuses(group, scan)
	total := len(statuses)
	installed := 0
	unknown := 0
	info := installItemInfo{detail: installDetail(group)}
	for _, status := range statuses {
		label := status.kind + ": " + status.label
		switch {
		case !status.known:
			unknown++
			info.unknown = append(info.unknown, label)
		case status.installed:
			installed++
			info.installed = append(info.installed, label)
		default:
			info.missing = append(info.missing, label)
		}
	}
	info.summary = installSummary(installed, unknown, total)
	info.state = windowsInstallState(installed, unknown, total)
	return info
}

func installStatuses(group config.PackageGroup, scan installScan) []itemStatus {
	var statuses []itemStatus
	for _, value := range group.Features {
		statuses = append(statuses, featureStatus(value, scan))
	}
	for _, value := range group.Winget {
		statuses = append(statuses, wingetPackageStatus(value, scan))
	}
	for _, value := range group.ScoopBuckets {
		statuses = append(statuses, scoopBucketStatus(value))
	}
	for _, value := range group.Scoop {
		statuses = append(statuses, scoopPackageStatus(value))
	}
	for _, value := range group.PSModules {
		statuses = append(statuses, psModuleStatus(value))
	}
	return statuses
}

func featureStatus(feature string, scan installScan) itemStatus {
	status := itemStatus{label: feature, kind: "Feature", known: true}
	switch feature {
	case "scoop":
		status.installed = scoopAvailable()
	case "powershell":
		if commandExists("pwsh") {
			status.installed = true
			return status
		}
		return wingetPackageStatus("Microsoft.PowerShell", scan).withLabel(feature, "Feature")
	case "psreadline":
		return psModuleStatus("PSReadLine").withLabel(feature, "Feature")
	case "node-pnpm":
		status.installed = commandExists("pnpm")
	case "powershell-profile":
		status.installed = powerShellProfileLinked()
	case "wezterm-context-menu":
		status.known = scan.featureKnown
		status.installed = scan.features[feature]
	case "win10-classic-menu":
		status.known = scan.featureKnown
		status.installed = scan.features[feature]
	default:
		status.known = false
	}
	return status
}

func (s itemStatus) withLabel(label, kind string) itemStatus {
	s.label = label
	s.kind = kind
	return s
}

func wingetPackageStatus(id string, scan installScan) itemStatus {
	status := itemStatus{label: id, kind: "WinGet", known: scan.wingetKnown}
	if !scan.wingetKnown {
		return status
	}
	status.installed = scan.wingetIDs[strings.ToLower(id)]
	return status
}

func scoopBucketStatus(bucket string) itemStatus {
	status := itemStatus{label: bucket, kind: "Scoop bucket", known: true}
	status.installed = childDirExists(filepath.Join(config.Expand("~"), "scoop", "buckets"), bucket)
	return status
}

func scoopPackageStatus(pkg string) itemStatus {
	status := itemStatus{label: pkg, kind: "Scoop", known: true}
	status.installed = childDirExists(filepath.Join(config.Expand("~"), "scoop", "apps"), pkg)
	return status
}

func psModuleStatus(module string) itemStatus {
	status := itemStatus{label: module, kind: "PowerShell module", known: true}
	status.installed = powerShellModuleExists(module)
	return status
}

func powerShellProfileLinked() bool {
	paths := []string{
		filepath.Join(config.Expand("~"), "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1"),
		filepath.Join(config.Expand("~"), "Documents", "PowerShell", "Microsoft.PowerShell_profile.ps1"),
	}
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}

func powerShellModuleExists(module string) bool {
	roots := []string{
		filepath.Join(config.Expand("~"), "Documents", "PowerShell", "Modules"),
		filepath.Join(config.Expand("~"), "Documents", "WindowsPowerShell", "Modules"),
	}
	for _, root := range filepath.SplitList(os.Getenv("PSModulePath")) {
		if strings.TrimSpace(root) != "" {
			roots = append(roots, root)
		}
	}
	for _, root := range roots {
		if childDirExists(root, module) {
			return true
		}
	}
	return false
}

func childDirExists(parent, name string) bool {
	entries, err := os.ReadDir(parent)
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if entry.IsDir() && strings.EqualFold(entry.Name(), name) {
			return true
		}
	}
	return false
}

func installSummary(installed, unknown, total int) string {
	if total == 0 {
		return "nothing configured"
	}
	summary := fmt.Sprintf("%d/%d installed", installed, total)
	if unknown > 0 {
		summary += fmt.Sprintf(", %d unknown", unknown)
	}
	return summary
}

func installDetail(group config.PackageGroup) string {
	parts := []string{
		fmt.Sprintf("%d WinGet", len(group.Winget)),
		fmt.Sprintf("%d Scoop", len(group.Scoop)),
		fmt.Sprintf("%d module(s)", len(group.PSModules)),
		fmt.Sprintf("%d feature(s)", len(group.Features)),
	}
	if len(group.ScoopBuckets) > 0 {
		parts = append(parts, fmt.Sprintf("%d bucket(s)", len(group.ScoopBuckets)))
	}
	return strings.Join(parts, ", ")
}

func windowsInstallState(installed, unknown, total int) ui.SelectState {
	switch {
	case total == 0:
		return ui.SelectStateGood
	case unknown == total:
		return ui.SelectStateUnknown
	case installed == total:
		return ui.SelectStateGood
	case installed == 0 && unknown == 0:
		return ui.SelectStateBad
	default:
		return ui.SelectStatePartial
	}
}

func installInspect(group config.PackageGroup, info installItemInfo) string {
	lines := []string{
		"Label: " + group.Label,
		"Configured: " + info.detail,
	}
	lines = appendStatusLines(lines, "Installed", info.installed)
	lines = appendStatusLines(lines, "Missing", info.missing)
	lines = appendStatusLines(lines, "Unknown", info.unknown)
	return strings.Join(lines, "\n")
}

func appendStatusLines(lines []string, label string, values []string) []string {
	if len(values) == 0 {
		return lines
	}
	lines = append(lines, label+":")
	lines = append(lines, values...)
	return lines
}
