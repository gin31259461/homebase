package windows

import (
	"flag"
	"fmt"

	"github.com/gin31259461/homebase/internal/config"
	sharedinstall "github.com/gin31259461/homebase/internal/install"
	"github.com/gin31259461/homebase/internal/platform"
	"github.com/gin31259461/homebase/internal/run"
	"github.com/gin31259461/homebase/internal/ui"
)

func runSetup(args []string, r run.Runner) error {
	fs := flag.NewFlagSet("setup", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "skip confirmation")
	fs.BoolVar(yes, "y", false, "skip confirmation")
	all := fs.Bool("all", false, "select all setup hooks")
	var selectedFlags stringList
	fs.Var(&selectedFlags, "hook", "setup hook key, repeatable")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 0 {
		return fmt.Errorf("usage: hb setup [--hook <key>] [--all] [--yes]")
	}
	if err := config.EnsureForPlatform(ID, false); err != nil {
		return err
	}
	groups, err := config.LoadPackageGroupsForPlatform(ID)
	if err != nil {
		return err
	}
	hooks := configuredSetupHooks(groups)

	selected := append([]string(nil), selectedFlags...)
	if *all {
		for _, hook := range hooks {
			selected = append(selected, hook.Key)
		}
	}
	if len(selected) == 0 {
		selected, err = ui.SelectKeys("Setup Hooks", windowsSetupItems(hooks))
		if err != nil {
			return err
		}
	}
	selected = sharedinstall.UniqueKnown(selected, windowsSetupHookSet(hooks))
	if len(selected) == 0 {
		ui.Warn("No setup hooks selected")
		return nil
	}

	hookByKey := map[string]platform.SetupHook{}
	ui.Section("Setup plan")
	for _, hook := range hooks {
		hookByKey[hook.Key] = hook
	}
	for _, key := range selected {
		fmt.Printf("  %s %s\n", ui.OKStyle.Render("+"), hookByKey[key].Label)
	}
	if !*yes && !ui.Confirm("Run selected setup hooks?", false) {
		ui.Warn("Aborted")
		return nil
	}
	for _, key := range selected {
		if err := installFeature(r, key); err != nil {
			return err
		}
	}
	ui.OK("Setup complete")
	return nil
}

func setupHooks() []platform.SetupHook {
	return []platform.SetupHook{
		{Key: "powershell-profile", Label: "PowerShell profile links"},
		{Key: "wezterm-context-menu", Label: "WezTerm context menu"},
		{Key: "win10-classic-menu", Label: "Windows 10 classic context menu"},
	}
}

func configuredSetupHooks(groups []config.PackageGroup) []platform.SetupHook {
	configured := map[string]bool{}
	for _, group := range groups {
		for _, feature := range group.Features {
			if isSetupFeature(feature) {
				configured[feature] = true
			}
		}
	}
	var hooks []platform.SetupHook
	for _, hook := range setupHooks() {
		if configured[hook.Key] {
			hooks = append(hooks, hook)
		}
	}
	return hooks
}

func windowsSetupItems(hooks []platform.SetupHook) []ui.SelectItem {
	items := make([]ui.SelectItem, 0, len(hooks))
	for _, hook := range hooks {
		items = append(items, ui.SelectItem{
			Key:         hook.Key,
			Label:       hook.Label,
			DetailValue: "ready",
			Inspect:     "Hook: " + hook.Key,
			State:       ui.SelectStateGood,
		})
	}
	return items
}

func windowsSetupHookSet(hooks []platform.SetupHook) map[string]bool {
	set := make(map[string]bool, len(hooks))
	for _, hook := range hooks {
		set[hook.Key] = true
	}
	return set
}
