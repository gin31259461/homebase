package archlinux

import (
	"fmt"
	"strings"

	"github.com/gin31259461/homebase/internal/config"
	sharedinstall "github.com/gin31259461/homebase/internal/install"
	"github.com/gin31259461/homebase/internal/run"
	"github.com/gin31259461/homebase/internal/ui"
)

func packageItems(groups []config.PackageGroup, installed map[string]bool) []ui.SelectItem {
	var items []ui.SelectItem
	for _, group := range groups {
		all := append(append([]string{}, group.Pacman...), group.AUR...)
		installedPkgs, missingPkgs := splitInstalled(all, installed)
		state := sharedinstall.InstallState(len(installedPkgs), len(all))
		items = append(items, ui.SelectItem{
			Key:             group.Key,
			Label:           group.Label,
			DetailValue:     fmt.Sprintf("%s installed, %d pacman, %d AUR", installRatio(all, installed), len(group.Pacman), len(group.AUR)),
			Inspect:         installInspect(group, installedPkgs, missingPkgs),
			State:           state,
			DefaultSelected: group.Default,
		})
	}
	return items
}

func splitInstalled(pkgs []string, installed map[string]bool) ([]string, []string) {
	var installedPkgs, missingPkgs []string
	for _, pkg := range pkgs {
		if installed[pkg] {
			installedPkgs = append(installedPkgs, pkg)
		} else {
			missingPkgs = append(missingPkgs, pkg)
		}
	}
	return installedPkgs, missingPkgs
}

func installInspect(group config.PackageGroup, installedPkgs, missingPkgs []string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Label: %s\n", group.Label)
	fmt.Fprintf(&b, "Pacman: %s\n", strings.Join(group.Pacman, ", "))
	fmt.Fprintf(&b, "AUR: %s\n", strings.Join(group.AUR, ", "))
	fmt.Fprintf(&b, "Installed: %s\n", strings.Join(installedPkgs, ", "))
	fmt.Fprintf(&b, "Missing: %s", strings.Join(missingPkgs, ", "))
	return b.String()
}

func installRatio(pkgs []string, installed map[string]bool) string {
	count := 0
	for _, pkg := range pkgs {
		if installed[pkg] {
			count++
		}
	}
	return fmt.Sprintf("%d/%d", count, len(pkgs))
}

func pacmanInstalled(r run.Runner, pkg string) bool {
	return r.Quiet("pacman", "-Qi", pkg) == nil
}

func installedPackageSet(r run.Runner) (map[string]bool, error) {
	output, err := r.Capture("pacman", "-Qq")
	if err != nil {
		return nil, err
	}
	return parseInstalledPackages(output), nil
}

func parseInstalledPackages(output string) map[string]bool {
	installed := map[string]bool{}
	for _, pkg := range strings.Fields(output) {
		installed[pkg] = true
	}
	return installed
}
