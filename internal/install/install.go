package install

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/run"
	"github.com/gin31259461/homebase/internal/setup"
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
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "skip confirmation")
	fs.BoolVar(yes, "y", false, "skip confirmation")
	all := fs.Bool("all", false, "select all package groups")
	noSetup := fs.Bool("no-setup", false, "skip post-install setup")
	var selectedFlags stringList
	fs.Var(&selectedFlags, "group", "package group key, repeatable")
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
	groups, err := config.LoadPackageGroupsForPlatform(platformID)
	if err != nil {
		return err
	}
	if err := ensureAURHelper(r, cfg.PackageManager.AUR); err != nil {
		return err
	}
	var installed map[string]bool
	if err := ui.WithSpinner("Scanning installed package database", func() error {
		var err error
		installed, err = system.InstalledPackageSet(r)
		return err
	}); err != nil {
		return err
	}

	selected := append([]string(nil), selectedFlags...)
	if *all {
		selected = groupKeys(groups)
	}
	if len(selected) == 0 {
		items := PackageItems(groups, installed)
		selected, err = ui.SelectKeys("Package Groups", items)
		if err != nil {
			return err
		}
	}
	selected = UniqueKnown(selected, PackageGroupSet(groups))
	if len(selected) == 0 {
		ui.Warn("No package groups selected")
		return nil
	}

	official, aur := InstallPlan(groups, selected, installed)
	if len(official) == 0 && len(aur) == 0 {
		ui.OK("All selected packages are already installed")
	} else {
		PrintInstallPlan(official, aur, cfg.PackageManager)
		if !*yes && !ui.Confirm(fmt.Sprintf("Install %d package(s)?", len(official)+len(aur)), false) {
			ui.Warn("Aborted")
			return nil
		}
		if err := InstallPackages(r, official, aur, cfg.PackageManager); err != nil {
			return err
		}
	}

	if !*noSetup {
		if err := RunSetup(r, groups, selected, installed, *yes); err != nil {
			return err
		}
	}
	return nil
}

func ensureAURHelper(r run.Runner, helper string) error {
	if helper == "" || system.CommandExists(helper) {
		return nil
	}
	if helper != "yay" {
		return fmt.Errorf("AUR helper %q is not installed", helper)
	}
	ui.Section("AUR helper")
	tmp, err := os.MkdirTemp("", "homebase-yay-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if err := r.Run("git", "clone", "--depth=1", "https://aur.archlinux.org/yay.git", filepath.Join(tmp, "yay")); err != nil {
		return err
	}
	return r.RunIn(filepath.Join(tmp, "yay"), "makepkg", "-si", "--noconfirm")
}

func InstallPlan(groups []config.PackageGroup, selected []string, installed map[string]bool) ([]string, []string) {
	groupByKey := map[string]config.PackageGroup{}
	for _, group := range groups {
		groupByKey[group.Key] = group
	}
	offSeen := map[string]bool{}
	aurSeen := map[string]bool{}
	var official, aur []string
	for _, key := range selected {
		group := groupByKey[key]
		for _, pkg := range group.Pacman {
			if !installed[pkg] && !offSeen[pkg] {
				official = append(official, pkg)
				offSeen[pkg] = true
			}
		}
		for _, pkg := range group.AUR {
			if !installed[pkg] && !aurSeen[pkg] {
				aur = append(aur, pkg)
				aurSeen[pkg] = true
			}
		}
	}
	sort.Strings(official)
	sort.Strings(aur)
	return official, aur
}

func PrintInstallPlan(official, aur []string, pm config.PackageManager) {
	ui.Section("Install plan")
	if len(official) > 0 {
		fmt.Printf("%s (%s)\n", ui.TitleStyle.Render("Official packages"), pm.Official)
		for _, pkg := range official {
			fmt.Printf("  %s %s\n", ui.OKStyle.Render("+"), pkg)
		}
	}
	if len(aur) > 0 {
		fmt.Printf("%s (%s)\n", ui.TitleStyle.Render("AUR packages"), pm.AUR)
		for _, pkg := range aur {
			fmt.Printf("  %s %s\n", ui.WarnStyle.Render("+"), pkg)
		}
	}
}

func InstallPackages(r run.Runner, official, aur []string, pm config.PackageManager) error {
	if len(official) > 0 {
		ui.Section("Installing official packages")
		switch pm.Official {
		case "pacman", "":
			args := append([]string{"pacman", "-S", "--needed", "--noconfirm"}, official...)
			if err := r.Run("sudo", args...); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported official package manager: %s", pm.Official)
		}
	}
	if len(aur) > 0 {
		ui.Section("Installing AUR packages")
		args := append([]string{"-S", "--needed", "--noconfirm"}, aur...)
		if err := r.Run(pm.AUR, args...); err != nil {
			return err
		}
	}
	return nil
}

func RunSetup(r run.Runner, groups []config.PackageGroup, selected []string, installed map[string]bool, yes bool) error {
	ui.Section("Extra configuration")
	groupByKey := map[string]config.PackageGroup{}
	for _, group := range groups {
		groupByKey[group.Key] = group
	}
	seen := map[string]bool{}
	var keys []string
	for _, key := range selected {
		group := groupByKey[key]
		if setup.Exists(key) && groupHasInstalledPackage(group, installed) {
			keys = append(keys, key)
			seen[key] = true
		}
		for _, pkg := range append(group.Pacman, group.AUR...) {
			if !seen[pkg] && setup.Exists(pkg) && installed[pkg] {
				keys = append(keys, pkg)
				seen[pkg] = true
			}
		}
	}
	for _, key := range keys {
		if err := setup.RunKey(r, key); err != nil {
			return err
		}
	}
	if setup.Exists("autologin") {
		if _, err := os.Stat("/etc/systemd/system/getty@tty1.service.d/override.conf"); err == nil {
			ui.OK("Autologin already configured")
		} else if yes || ui.Confirm("Configure tty1 autologin for "+os.Getenv("USER")+"?", false) {
			if err := setup.RunKey(r, "autologin"); err != nil {
				return err
			}
		}
	}
	if len(keys) == 0 {
		ui.Note("No package setup hooks matched the selected groups")
	}
	return nil
}

func PackageItems(groups []config.PackageGroup, installed map[string]bool) []ui.SelectItem {
	var items []ui.SelectItem
	for _, group := range groups {
		all := append(append([]string{}, group.Pacman...), group.AUR...)
		items = append(items, ui.SelectItem{
			Key:    group.Key,
			Label:  group.Label,
			Detail: fmt.Sprintf("%s installed, %d total", installRatio(all, installed), len(all)),
		})
	}
	return items
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

func PackageGroupSet(groups []config.PackageGroup) map[string]bool {
	set := map[string]bool{}
	for _, group := range groups {
		set[group.Key] = true
	}
	return set
}

func groupKeys(groups []config.PackageGroup) []string {
	var keys []string
	for _, group := range groups {
		keys = append(keys, group.Key)
	}
	return keys
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

func groupHasInstalledPackage(group config.PackageGroup, installed map[string]bool) bool {
	all := append(append([]string{}, group.Pacman...), group.AUR...)
	if len(all) == 0 {
		return true
	}
	for _, pkg := range all {
		if installed[pkg] {
			return true
		}
	}
	return false
}
