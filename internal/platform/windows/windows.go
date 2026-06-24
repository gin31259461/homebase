package windows

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/gitutil"
	"github.com/gin31259461/homebase/internal/run"
	synccmd "github.com/gin31259461/homebase/internal/sync"
	"github.com/gin31259461/homebase/internal/system"
	"github.com/gin31259461/homebase/internal/ui"
)

const ID = "windows"

type Platform struct {
	runner run.Runner
}

type stringList []string

type installPlan struct {
	Features     []string
	Winget       []string
	ScoopBuckets []string
	Scoop        []string
	PSModules    []string
}

func New() Platform {
	return Platform{runner: run.New()}
}

func (p Platform) ID() string {
	return ID
}

func (p Platform) Family() string {
	return ID
}

func (p Platform) Matches() bool {
	return runtime.GOOS == "windows"
}

func (p Platform) Bootstrap(args []string) error {
	return runBootstrap(args, p.runner)
}

func (p Platform) Install(args []string) error {
	return runInstall(args, p.runner)
}

func (p Platform) Cleanup(args []string) error {
	return runCleanup(args, p.runner)
}

func (p Platform) Sync(args []string) error {
	return synccmd.RunWithPlatform(args, p.runner, ID)
}

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}

func runBootstrap(args []string, r run.Runner) error {
	fs := flag.NewFlagSet("bootstrap", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "accept defaults and skip prompts")
	fs.BoolVar(yes, "y", false, "accept defaults and skip prompts")
	repo := fs.String("repo", "", "dotfiles repository URL")
	installAfterBootstrap := fs.Bool("install", false, "run package installer after bootstrap")
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

	ui.Section("Bootstrap")
	if err := ensureUserPath(r, hbBinDir()); err != nil {
		return err
	}
	if err := installBootstrapBasics(r, cfg.Bootstrap.Basics); err != nil {
		return err
	}
	sshRepo, httpsRepo, err := resolveDotfilesRepo(cfg, *repo, !*yes)
	if err != nil {
		return err
	}
	if err := deployDotfiles(r, cfg, sshRepo, httpsRepo); err != nil {
		return err
	}
	if err := configureDotfiles(r, cfg); err != nil {
		return err
	}
	if err := linkPowerShellProfiles(r); err != nil {
		return err
	}
	if *installAfterBootstrap {
		argv := []string{}
		if *yes {
			argv = append(argv, "--all", "--yes")
		}
		if err := runInstall(argv, r); err != nil {
			return err
		}
	} else {
		ui.Note("Package installation skipped. Run hb install when ready.")
	}

	ui.Section("Done")
	ui.OK("Bootstrap complete")
	ui.Note("Open a new terminal to refresh PATH and profile links")
	ui.Note("Use hb sync to stage, commit, and push configured dotfile paths")
	return nil
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
		selected, err = ui.SelectKeys("Package Groups", packageItems(groups))
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

func runCleanup(args []string, r run.Runner) error {
	fs := flag.NewFlagSet("cleanup", flag.ContinueOnError)
	yes := fs.Bool("yes", false, "skip confirmation")
	fs.BoolVar(yes, "y", false, "skip confirmation")
	all := fs.Bool("all", false, "select all cleanup tasks")
	var selectedFlags stringList
	fs.Var(&selectedFlags, "task", "cleanup task key, repeatable")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if err := config.EnsureForPlatform(ID, false); err != nil {
		return err
	}
	tasks, err := config.LoadCleanupTasksForPlatform(ID, system.CommandExists)
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
		selected, err = ui.SelectKeys("Cleanup Tasks", cleanupItems(tasks))
		if err != nil {
			return err
		}
	}
	selected = uniqueKnown(selected, cleanupTaskSet(tasks))
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
		if err := runCleanupTask(r, key); err != nil {
			return err
		}
	}
	ui.OK("System cleanup complete")
	return nil
}

func installBootstrapBasics(r run.Runner, basics []string) error {
	for _, basic := range basics {
		switch strings.ToLower(strings.TrimSpace(basic)) {
		case "", "git":
			if strings.TrimSpace(basic) == "" || system.CommandExists("git") {
				continue
			}
			if err := installWingetPackage(r, "Git.Git"); err != nil {
				return err
			}
		case "go", "golang":
			if system.CommandExists("go") {
				continue
			}
			if err := installWingetPackage(r, "GoLang.Go"); err != nil {
				return err
			}
		default:
			ui.Warn("Unknown Windows bootstrap basic: " + basic)
		}
	}
	return nil
}

func resolveDotfilesRepo(cfg config.App, flagRepo string, interactive bool) (string, string, error) {
	sshRepo := cfg.Dotfiles.SSHRepo
	httpsRepo := cfg.Dotfiles.HTTPSRepo
	if mem, err := gitutil.ReadRepoMemory(cfg.Dotfiles.MemoryFile); err == nil && mem.Repo != "" {
		if ssh, https, err := gitutil.NormalizeRepo(mem.Repo); err == nil {
			sshRepo, httpsRepo = ssh, https
		}
	}
	if flagRepo != "" {
		return gitutil.NormalizeRepo(flagRepo)
	}
	if interactive && !ui.Confirm("Use dotfiles repo "+sshRepo+"?", true) {
		input := ui.PromptText("Dotfiles repo", sshRepo)
		return gitutil.NormalizeRepo(input)
	}
	return sshRepo, httpsRepo, nil
}

func deployDotfiles(r run.Runner, cfg config.App, sshRepo, httpsRepo string) error {
	if _, err := os.Stat(cfg.Dotfiles.Dir); err == nil {
		ui.OK("Bare dotfiles repo already present at " + cfg.Dotfiles.Dir)
		return rememberExistingRemote(r, cfg)
	}
	cloneURL := sshRepo
	if strings.HasPrefix(sshRepo, "git@github.com:") && !gitHubRepoAvailable(r, sshRepo) && httpsRepo != "" {
		ui.Warn("No GitHub SSH access detected; cloning over HTTPS")
		cloneURL = httpsRepo
	}
	if cloneURL == "" {
		return errors.New("no dotfiles repository URL resolved")
	}

	tmp, err := os.MkdirTemp("", "homebase-dotfiles-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)

	ui.Section("Dotfiles repository")
	worktree := filepath.Join(tmp, "dotfiles")
	if err := r.Run("git", "clone", "--separate-git-dir="+cfg.Dotfiles.Dir, cloneURL, worktree); err != nil {
		return err
	}
	if err := copyTree(worktree, config.Expand("~")); err != nil {
		return err
	}
	if sshRepo != "" {
		remoteArgs := gitutil.DotArgs(cfg, "remote", "set-url", "origin", sshRepo)
		_ = r.Run(remoteArgs[0], remoteArgs[1:]...)
		if err := gitutil.SaveRepoMemory(cfg.Dotfiles.MemoryFile, sshRepo, cfg.Dotfiles.Branch); err != nil {
			return err
		}
	}
	ui.OK("Dotfiles deployed")
	return config.EnsureForPlatform(ID, false)
}

func rememberExistingRemote(r run.Runner, cfg config.App) error {
	if mem, err := gitutil.ReadRepoMemory(cfg.Dotfiles.MemoryFile); err == nil && mem.Repo != "" {
		return nil
	}
	remoteArgs := gitutil.DotArgs(cfg, "remote", "get-url", "origin")
	remote, err := r.Capture(remoteArgs[0], remoteArgs[1:]...)
	if err != nil || strings.TrimSpace(remote) == "" {
		return nil
	}
	return gitutil.SaveRepoMemory(cfg.Dotfiles.MemoryFile, strings.TrimSpace(remote), cfg.Dotfiles.Branch)
}

func configureDotfiles(r run.Runner, cfg config.App) error {
	ui.Section("Configuration")
	configArgs := gitutil.DotArgs(cfg, "config", "--local", "status.showUntrackedFiles", "no")
	if err := r.Run(configArgs[0], configArgs[1:]...); err != nil {
		return err
	}
	submoduleArgs := gitutil.DotArgs(cfg, "submodule", "update", "--init", "--recursive")
	if err := r.Run(submoduleArgs[0], submoduleArgs[1:]...); err != nil {
		return err
	}
	ui.OK("Submodules ready")
	return nil
}

func copyTree(src, dst string) error {
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil || rel == "." {
			return err
		}
		if rel == ".git" || strings.HasPrefix(rel, ".git"+string(os.PathSeparator)) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
}

func gitHubRepoAvailable(r run.Runner, repo string) bool {
	return r.Quiet("git", "ls-remote", "--exit-code", repo, "HEAD") == nil
}

func packageItems(groups []config.PackageGroup) []ui.SelectItem {
	items := make([]ui.SelectItem, 0, len(groups))
	for _, group := range groups {
		items = append(items, ui.SelectItem{
			Key:    group.Key,
			Label:  group.Label,
			Detail: fmt.Sprintf("%d configured item(s)", countGroupItems(group)),
		})
	}
	return items
}

func countGroupItems(group config.PackageGroup) int {
	return len(group.Winget) + len(group.ScoopBuckets) + len(group.Scoop) + len(group.PSModules) + len(group.Features)
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

func cleanupItems(tasks []config.CleanupTask) []ui.SelectItem {
	items := make([]ui.SelectItem, 0, len(tasks))
	for _, task := range tasks {
		items = append(items, ui.SelectItem{Key: task.Key, Label: task.Label, Detail: task.Detail})
	}
	return items
}

func cleanupTaskSet(tasks []config.CleanupTask) map[string]bool {
	set := map[string]bool{}
	for _, task := range tasks {
		set[task.Key] = true
	}
	return set
}

func uniqueKnown(values []string, known map[string]bool) []string {
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
		if system.CommandExists("pnpm") {
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
	if system.CommandExists("winget") && r.Quiet("winget", "list", "--id", id, "--exact") == nil {
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

func runCleanupTask(r run.Runner, key string) error {
	ui.Section(key)
	switch key {
	case "scoop-cache":
		if !scoopAvailable() {
			ui.Warn("scoop not found; skipping")
			return nil
		}
		return r.Run("scoop", "cache", "rm", "*")
	case "temp-files":
		return r.Run(powerShellExe(), "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", "Get-ChildItem $env:TEMP -ErrorAction SilentlyContinue | Remove-Item -Recurse -Force -ErrorAction SilentlyContinue")
	case "npm-cache":
		if !system.CommandExists("npm") {
			ui.Warn("npm not found; skipping")
			return nil
		}
		return r.Run("npm", "cache", "clean", "--force")
	case "winget-cache":
		cmd := "$paths = @((Join-Path $env:LOCALAPPDATA 'Packages\\Microsoft.DesktopAppInstaller_8wekyb3d8bbwe\\LocalCache'), (Join-Path $env:TEMP 'WinGet')); foreach ($path in $paths) { if (Test-Path $path) { Remove-Item $path -Recurse -Force -ErrorAction SilentlyContinue } }"
		return r.Run(powerShellExe(), "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", cmd)
	case "recycle-bin":
		return r.Run(powerShellExe(), "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", "Clear-RecycleBin -Force -ErrorAction SilentlyContinue")
	case "thumbnail-cache":
		cmd := "$explorer = Get-Process -Name explorer -ErrorAction SilentlyContinue; if ($explorer) { Stop-Process -Name explorer -Force -ErrorAction SilentlyContinue; Start-Sleep -Seconds 1 }; $dir = Join-Path $env:LOCALAPPDATA 'Microsoft\\Windows\\Explorer'; if (Test-Path $dir) { Get-ChildItem $dir -Filter 'thumbcache_*.db' -ErrorAction SilentlyContinue | Remove-Item -Force -ErrorAction SilentlyContinue }; if ($explorer) { Start-Process explorer.exe }"
		return r.Run(powerShellExe(), "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", cmd)
	default:
		return fmt.Errorf("no runner for cleanup task: %s", key)
	}
}

func ensureUserPath(r run.Runner, path string) error {
	addProcessPath(path)
	cmd := "$path = " + quotePS(path) + "; " +
		"$current = [Environment]::GetEnvironmentVariable('Path','User'); " +
		"$parts = @($current -split ';' | Where-Object { $_ }); " +
		"if ($parts -notcontains $path) { [Environment]::SetEnvironmentVariable('Path', (($parts + $path) -join ';'), 'User') }"
	return r.Run(powerShellExe(), "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", cmd)
}

func addProcessPath(path string) {
	if strings.TrimSpace(path) == "" {
		return
	}
	parts := filepath.SplitList(os.Getenv("PATH"))
	for _, part := range parts {
		if strings.EqualFold(filepath.Clean(part), filepath.Clean(path)) {
			return
		}
	}
	_ = os.Setenv("PATH", os.Getenv("PATH")+string(os.PathListSeparator)+path)
}

func hbBinDir() string {
	return filepath.Join(config.Expand("~"), ".local", "bin")
}

func scoopAvailable() bool {
	if system.CommandExists("scoop") {
		return true
	}
	if _, err := os.Stat(filepath.Join(config.Expand("~"), "scoop")); err == nil {
		addProcessPath(filepath.Join(config.Expand("~"), "scoop", "shims"))
		return true
	}
	return false
}

func powerShellExe() string {
	if system.CommandExists("powershell.exe") {
		return "powershell.exe"
	}
	return "powershell"
}

func quotePS(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
