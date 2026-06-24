package windows

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/run"
	"github.com/gin31259461/homebase/internal/ui"
)

type installPlan struct {
	Features     []string
	Winget       []string
	ScoopBuckets []string
	Scoop        []string
	PSModules    []string
}

func runInstall(args []string, r run.Runner) error {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "skip confirmation")
	fs.BoolVar(yes, "y", false, "skip confirmation")
	all := fs.Bool("all", false, "select all package groups")
	noSetup := fs.Bool("no-setup", false, "skip setup features")
	var selectedFlags stringList
	fs.Var(&selectedFlags, "group", "package group key, repeatable")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if err := config.EnsureForPlatform(ID, false); err != nil {
		return err
	}
	groups, err := config.LoadPackageGroupsForPlatform(ID)
	if err != nil {
		return err
	}

	selected := append([]string(nil), selectedFlags...)
	if *all {
		selected = groupKeys(groups)
	}
	if len(selected) == 0 {
		selected, err = ui.SelectKeys("Package Groups", packageItems(r, groups))
		if err != nil {
			return err
		}
	}
	selected = uniqueKnown(selected, packageGroupSet(groups))
	if len(selected) == 0 {
		ui.Warn("No package groups selected")
		return nil
	}

	plan := buildInstallPlan(groups, selected)
	if *noSetup {
		plan.Features = filterCoreFeatures(plan.Features)
	}
	if plan.empty() {
		ui.OK("No install actions selected")
		return nil
	}
	printInstallPlan(plan)
	if !*yes && !ui.Confirm("Install selected Windows items?", false) {
		ui.Warn("Aborted")
		return nil
	}
	return installWindowsPlan(r, plan)
}

func groupKeys(groups []config.PackageGroup) []string {
	keys := make([]string, 0, len(groups))
	for _, group := range groups {
		keys = append(keys, group.Key)
	}
	return keys
}

func packageGroupSet(groups []config.PackageGroup) map[string]bool {
	set := map[string]bool{}
	for _, group := range groups {
		set[group.Key] = true
	}
	return set
}

func buildInstallPlan(groups []config.PackageGroup, selected []string) installPlan {
	groupByKey := map[string]config.PackageGroup{}
	for _, group := range groups {
		groupByKey[group.Key] = group
	}
	var plan installPlan
	seen := map[string]bool{}
	add := func(kind, value string, dst *[]string) {
		value = strings.TrimSpace(value)
		key := kind + ":" + strings.ToLower(value)
		if value == "" || seen[key] {
			return
		}
		*dst = append(*dst, value)
		seen[key] = true
	}
	for _, key := range selected {
		group := groupByKey[key]
		for _, value := range group.Features {
			add("feature", value, &plan.Features)
		}
		for _, value := range group.Winget {
			add("winget", value, &plan.Winget)
		}
		for _, value := range group.ScoopBuckets {
			add("scoop-bucket", value, &plan.ScoopBuckets)
		}
		for _, value := range group.Scoop {
			add("scoop", value, &plan.Scoop)
		}
		for _, value := range group.PSModules {
			add("psmodule", value, &plan.PSModules)
		}
	}
	return plan
}

func (p installPlan) empty() bool {
	return len(p.Features)+len(p.Winget)+len(p.ScoopBuckets)+len(p.Scoop)+len(p.PSModules) == 0
}

func printInstallPlan(plan installPlan) {
	ui.Section("Install plan")
	printItems := func(title string, values []string) {
		if len(values) == 0 {
			return
		}
		fmt.Println(ui.TitleStyle.Render(title))
		for _, value := range values {
			fmt.Printf("  %s %s\n", ui.OKStyle.Render("+"), value)
		}
	}
	printItems("Windows features", plan.Features)
	printItems("WinGet packages", plan.Winget)
	printItems("Scoop buckets", plan.ScoopBuckets)
	printItems("Scoop packages", plan.Scoop)
	printItems("PowerShell modules", plan.PSModules)
}

func filterCoreFeatures(features []string) []string {
	setup := map[string]bool{
		"powershell-profile":   true,
		"wezterm-context-menu": true,
		"win10-classic-menu":   true,
	}
	var out []string
	for _, feature := range features {
		if !setup[feature] {
			out = append(out, feature)
		}
	}
	return out
}

func installWindowsPlan(r run.Runner, plan installPlan) error {
	for _, feature := range plan.Features {
		if err := installFeature(r, feature); err != nil {
			return err
		}
	}
	if len(plan.Scoop) > 0 {
		if err := installFeature(r, "scoop"); err != nil {
			return err
		}
	}
	for _, pkg := range plan.Winget {
		if err := installWingetPackage(r, pkg); err != nil {
			return err
		}
	}
	for _, bucket := range plan.ScoopBuckets {
		if err := ensureScoopBucket(r, bucket); err != nil {
			return err
		}
	}
	for _, pkg := range plan.Scoop {
		if err := installScoopPackage(r, pkg); err != nil {
			return err
		}
	}
	for _, module := range plan.PSModules {
		if err := installPSModule(r, module); err != nil {
			return err
		}
	}
	ui.OK("Installation complete")
	return nil
}

func installFeature(r run.Runner, feature string) error {
	switch feature {
	case "scoop":
		return installScoop(r)
	case "powershell":
		return installWingetPackage(r, "Microsoft.PowerShell")
	case "psreadline":
		return installPSModule(r, "PSReadLine")
	case "node-pnpm":
		if err := installWingetPackage(r, "OpenJS.NodeJS"); err != nil {
			return err
		}
		if commandExists("pnpm") {
			ui.OK("pnpm already installed")
			return nil
		}
		return r.Run("npm", "install", "-g", "pnpm")
	case "powershell-profile":
		return linkPowerShellProfiles(r)
	case "wezterm-context-menu":
		return addWezTermContextMenu(r)
	case "win10-classic-menu":
		return restoreClassicContextMenu(r)
	default:
		return fmt.Errorf("unsupported Windows feature: %s", feature)
	}
}

func installWingetPackage(r run.Runner, id string) error {
	if commandExists("winget") && r.Quiet("winget", "list", "--id", id, "--exact") == nil {
		ui.OK(id + " already installed")
		return nil
	}
	ui.Section("WinGet " + id)
	return r.Run("winget", "install", "--id", id, "--source", "winget", "--accept-source-agreements", "--accept-package-agreements")
}

func installScoop(r run.Runner) error {
	if scoopAvailable() {
		ui.OK("Scoop already installed")
		addProcessPath(filepath.Join(config.Expand("~"), "scoop", "shims"))
		return nil
	}
	ui.Section("Scoop")
	cmd := "Set-ExecutionPolicy RemoteSigned -Scope CurrentUser -Force; iwr -useb https://get.scoop.sh | iex"
	if err := r.Run(powerShellExe(), "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", cmd); err != nil {
		return err
	}
	addProcessPath(filepath.Join(config.Expand("~"), "scoop", "shims"))
	bucketCmd := "foreach ($bucket in 'extras','versions') { if (-not (scoop bucket list | Select-String -SimpleMatch $bucket)) { scoop bucket add $bucket } }"
	return r.Run(powerShellExe(), "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", bucketCmd)
}

func ensureScoopBucket(r run.Runner, bucket string) error {
	if err := installScoop(r); err != nil {
		return err
	}
	cmd := "if (-not (scoop bucket list | Select-String -SimpleMatch " + quotePS(bucket) + ")) { scoop bucket add " + quotePS(bucket) + " }"
	return r.Run(powerShellExe(), "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", cmd)
}

func installScoopPackage(r run.Runner, pkg string) error {
	if r.Quiet("scoop", "prefix", pkg) == nil {
		ui.OK(pkg + " already installed")
		return nil
	}
	ui.Section("Scoop " + pkg)
	if err := r.Run("scoop", "install", pkg); err != nil {
		return err
	}
	return r.Run("scoop", "reset", pkg)
}

func installPSModule(r run.Runner, module string) error {
	check := "if (Get-Module -ListAvailable -Name " + quotePS(module) + ") { exit 0 } else { exit 1 }"
	if r.Quiet(powerShellExe(), "-NoProfile", "-Command", check) == nil {
		ui.OK(module + " already installed")
		return nil
	}
	ui.Section("PowerShell module " + module)
	cmd := "Install-Module " + quotePS(module) + " -Force -SkipPublisherCheck -Scope CurrentUser"
	return r.Run(powerShellExe(), "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", cmd)
}

func linkPowerShellProfiles(r run.Runner) error {
	source := filepath.Join(config.Expand("~"), ".pwsh", "profile.ps1")
	if _, err := os.Stat(source); err != nil {
		ui.Warn("Profile source not found: " + source)
		return nil
	}
	cmd := "$source = " + quotePS(source) + "; " +
		"$profiles = @(" +
		"(Join-Path $HOME 'Documents\\WindowsPowerShell\\Microsoft.PowerShell_profile.ps1')," +
		"(Join-Path $HOME 'Documents\\PowerShell\\Microsoft.PowerShell_profile.ps1')" +
		"); " +
		"foreach ($profilePath in $profiles) { " +
		"$dir = Split-Path $profilePath -Parent; New-Item -ItemType Directory -Path $dir -Force | Out-Null; " +
		"try { New-Item -ItemType SymbolicLink -Path $profilePath -Target $source -Force -ErrorAction Stop | Out-Null } " +
		"catch { Copy-Item -Path $source -Destination $profilePath -Force } }"
	ui.Section("PowerShell profile")
	return r.Run(powerShellExe(), "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", cmd)
}

func addWezTermContextMenu(r run.Runner) error {
	cmd := "$cmd = Get-Command wezterm-gui.exe -CommandType Application -ErrorAction SilentlyContinue; " +
		"if (-not $cmd) { $prefix = scoop prefix wezterm-nightly 2>$null; if ($prefix) { $path = Join-Path $prefix 'wezterm-gui.exe' } } else { $path = $cmd.Source }; " +
		"if (-not $path -or -not (Test-Path $path)) { throw 'wezterm-gui.exe not found' }; " +
		"$key = 'HKCU:\\Software\\Classes\\Directory\\background\\shell\\wezterm-nightly'; $sub = Join-Path $key 'command'; " +
		"New-Item -Path $key -Force | Out-Null; Set-ItemProperty -Path $key -Name '(default)' -Value 'Open with wezterm-nightly'; Set-ItemProperty -Path $key -Name 'icon' -Value $path; " +
		"New-Item -Path $sub -Force | Out-Null; Set-ItemProperty -Path $sub -Name '(default)' -Value (\"$path start --cwd .\")"
	ui.Section("WezTerm context menu")
	return r.Run(powerShellExe(), "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", cmd)
}

func restoreClassicContextMenu(r run.Runner) error {
	cmd := "$path = 'HKCU:\\Software\\Classes\\CLSID\\{86ca1aa0-34aa-4e8b-a509-50c905bae2a2}\\InprocServer32'; " +
		"New-Item -Path $path -Force | Out-Null; Set-ItemProperty -Path $path -Name '(default)' -Value '' -Force; " +
		"Stop-Process -Name explorer -Force -ErrorAction SilentlyContinue; Start-Process explorer.exe"
	ui.Section("Win10 classic context menu")
	return r.Run(powerShellExe(), "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", cmd)
}
