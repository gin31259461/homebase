package archlinux

import (
	"flag"
	"fmt"
	"strings"

	"github.com/gin31259461/homebase/internal/completion"
	"github.com/gin31259461/homebase/internal/config"
	sharedinstall "github.com/gin31259461/homebase/internal/install"
	"github.com/gin31259461/homebase/internal/platform"
	"github.com/gin31259461/homebase/internal/run"
	"github.com/gin31259461/homebase/internal/ui"
)

const zshCompletionPath = "/usr/share/zsh/site-functions/_hb"

func runSetupCommand(args []string, r run.Runner) error {
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

	var installed map[string]bool
	if err := ui.WithSpinner("Scanning installed package database", func() error {
		var err error
		installed, err = installedPackageSet(r)
		return err
	}); err != nil {
		ui.Warn("Unable to scan installed packages; setup prerequisites are unknown")
	}

	hooks := setupHooks()
	selected := append([]string(nil), selectedFlags...)
	if *all {
		for _, hook := range hooks {
			selected = append(selected, hook.Key)
		}
	}
	if len(selected) == 0 {
		var err error
		selected, err = ui.SelectKeys("Setup Hooks", setupHookItems(hooks, installed))
		if err != nil {
			return err
		}
	}
	selected = sharedinstall.UniqueKnown(selected, setupHookSet(hooks))
	if len(selected) == 0 {
		ui.Warn("No setup hooks selected")
		return nil
	}

	hookByKey := map[string]platform.SetupHook{}
	for _, hook := range hooks {
		hookByKey[hook.Key] = hook
	}
	var runnable []string
	ui.Section("Setup plan")
	for _, key := range selected {
		missing := missingSetupRequirements(setupRequirements(key), installed)
		if installed != nil && len(missing) > 0 {
			ui.Warn(hookByKey[key].Label + " skipped; missing packages: " + strings.Join(missing, ", "))
			continue
		}
		fmt.Printf("  %s %s\n", ui.OKStyle.Render("+"), hookByKey[key].Label)
		runnable = append(runnable, key)
	}
	if len(runnable) == 0 {
		ui.Warn("No setup hooks are ready")
		return nil
	}
	if !*yes && !ui.Confirm("Run selected setup hooks?", false) {
		ui.Warn("Aborted")
		return nil
	}
	for _, key := range runnable {
		if err := runStandaloneSetupHook(r, key); err != nil {
			return err
		}
	}
	ui.OK("Setup complete")
	return nil
}

func setupHooks() []platform.SetupHook {
	return []platform.SetupHook{
		{Key: "system", Label: "System and mkinitcpio"},
		{Key: "shell", Label: "Zsh command completion"},
		{Key: "network", Label: "NetworkManager"},
		{Key: "docker", Label: "Docker service and group"},
		{Key: "razer", Label: "OpenRazer"},
		{Key: "sunshine", Label: "Sunshine service and capabilities"},
		{Key: "sddm", Label: "SDDM display manager"},
		{Key: "autologin", Label: "TTY1 autologin"},
		{Key: "git-credentials", Label: "Git Credential Manager (GPG/pass)"},
	}
}

func setupRequirements(key string) []string {
	switch key {
	case "shell":
		return []string{"zsh"}
	case "network":
		return []string{"networkmanager"}
	case "docker":
		return []string{"docker"}
	case "razer":
		return []string{"openrazer-daemon"}
	case "sunshine":
		return []string{"sunshine"}
	case "sddm":
		return []string{"sddm"}
	case "git-credentials":
		return []string{"git", "gnupg", "pass", "git-credential-manager-bin"}
	default:
		return nil
	}
}

func setupHookItems(hooks []platform.SetupHook, installed map[string]bool) []ui.SelectItem {
	items := make([]ui.SelectItem, 0, len(hooks))
	for _, hook := range hooks {
		requirements := setupRequirements(hook.Key)
		missing := missingSetupRequirements(requirements, installed)
		state := ui.SelectStateGood
		detail := "ready"
		if len(requirements) > 0 && installed == nil {
			state = ui.SelectStateUnknown
			detail = "prerequisite unknown"
		} else if len(missing) == 1 {
			state = ui.SelectStateBad
			detail = "missing " + missing[0]
		} else if len(missing) > 1 {
			state = ui.SelectStateBad
			detail = fmt.Sprintf("missing %d packages", len(missing))
		}
		items = append(items, ui.SelectItem{
			Key:         hook.Key,
			Label:       hook.Label,
			DetailValue: detail,
			Inspect:     "Hook: " + hook.Key + "\nRequired packages: " + displayRequirements(requirements),
			State:       state,
		})
	}
	return items
}

func missingSetupRequirements(requirements []string, installed map[string]bool) []string {
	if installed == nil {
		return nil
	}
	var missing []string
	for _, requirement := range requirements {
		if !installed[requirement] {
			missing = append(missing, requirement)
		}
	}
	return missing
}

func displayRequirements(requirements []string) string {
	if len(requirements) == 0 {
		return "none"
	}
	return strings.Join(requirements, ", ")
}

func setupHookSet(hooks []platform.SetupHook) map[string]bool {
	set := make(map[string]bool, len(hooks))
	for _, hook := range hooks {
		set[hook.Key] = true
	}
	return set
}

func runStandaloneSetupHook(r run.Runner, key string) error {
	if key == "system" {
		ui.Section("Setup system")
		return systemSetup(r)
	}
	if key == "git-credentials" {
		ui.Section("Setup Git credentials")
		return gitCredentials(r)
	}
	return runSetupKey(r, key)
}

func zshCompletion(r run.Runner) error {
	script, err := completion.ZshForPlatform(ID, setupHooks())
	if err != nil {
		return err
	}
	if err := writeSudoFile(r, zshCompletionPath, script); err != nil {
		return err
	}
	ui.OK("Installed Zsh completion for hb")
	ui.Note("Restart Zsh or run: autoload -Uz compinit && compinit")
	return nil
}
