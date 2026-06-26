package archlinux

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gin31259461/homebase/internal/config"
	sharedinstall "github.com/gin31259461/homebase/internal/install"
	"github.com/gin31259461/homebase/internal/run"
	"github.com/gin31259461/homebase/internal/system"
	"github.com/gin31259461/homebase/internal/ui"
)

func runInstall(args []string, r run.Runner) error {
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

	if err := config.EnsureForPlatform(ID, false); err != nil {
		return err
	}
	cfg, err := config.LoadForPlatform(ID)
	if err != nil {
		return err
	}
	pm := archPackageManager(cfg.PackageManager)
	groups, err := config.LoadPackageGroupsForPlatform(ID)
	if err != nil {
		return err
	}
	var installed map[string]bool
	if err := ui.WithSpinner("Scanning installed package database", func() error {
		var err error
		installed, err = installedPackageSet(r)
		return err
	}); err != nil {
		return err
	}

	selected := append([]string(nil), selectedFlags...)
	if *all {
		selected = sharedinstall.GroupKeys(groups)
	}
	if len(selected) == 0 {
		items := packageItems(groups, installed)
		selected, err = ui.SelectKeys("Package Groups", items)
		if err != nil {
			return err
		}
	}
	selected = sharedinstall.UniqueKnown(selected, sharedinstall.PackageGroupSet(groups))
	if len(selected) == 0 {
		ui.Warn("No package groups selected")
		return nil
	}

	official, aur := installPlan(groups, selected, installed)
	if len(official) == 0 && len(aur) == 0 {
		ui.OK("All selected packages are already installed")
	} else {
		printInstallPlan(official, aur, pm)
		if !*yes && !ui.Confirm(fmt.Sprintf("Install %d package(s)?", len(official)+len(aur)), false) {
			ui.Warn("Aborted")
			return nil
		}
		if len(aur) > 0 {
			if err := ensureAURHelper(r, pm.AUR); err != nil {
				return err
			}
		}
		if err := installPackages(r, official, aur, pm); err != nil {
			return err
		}
		markInstalled(installed, official, aur)
	}

	if !*noSetup {
		if err := runSetup(r, groups, selected, installed, *yes); err != nil {
			return err
		}
	}
	return nil
}

func archPackageManager(pm config.PackageManager) config.PackageManager {
	if strings.TrimSpace(pm.Official) == "" {
		pm.Official = "pacman"
	}
	if strings.TrimSpace(pm.AUR) == "" {
		pm.AUR = "yay"
	}
	return pm
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

func installPlan(groups []config.PackageGroup, selected []string, installed map[string]bool) ([]string, []string) {
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

func printInstallPlan(official, aur []string, pm config.PackageManager) {
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

func installPackages(r run.Runner, official, aur []string, pm config.PackageManager) error {
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

func markInstalled(installed map[string]bool, packageLists ...[]string) {
	if installed == nil {
		return
	}
	for _, packages := range packageLists {
		for _, pkg := range packages {
			installed[pkg] = true
		}
	}
}
