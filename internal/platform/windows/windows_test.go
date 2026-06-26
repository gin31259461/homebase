package windows

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/testutil"
	"github.com/gin31259461/homebase/internal/ui"
)

func TestBuildInstallPlanDedupesInSelectionOrder(t *testing.T) {
	groups := []config.PackageGroup{
		{
			Key:       "core",
			Features:  []string{"scoop", "psreadline"},
			Winget:    []string{"Git.Git"},
			PSModules: []string{"PSReadLine"},
		},
		{
			Key:          "apps",
			Features:     []string{"scoop"},
			Winget:       []string{"Git.Git", "wez.wezterm"},
			ScoopBuckets: []string{"custom-bucket"},
			Scoop:        []string{"fzf"},
		},
	}
	plan := buildInstallPlan(groups, []string{"core", "apps"})

	if want := []string{"scoop", "psreadline"}; !reflect.DeepEqual(plan.Features, want) {
		t.Fatalf("features = %#v; want %#v", plan.Features, want)
	}
	if want := []string{"Git.Git", "wez.wezterm"}; !reflect.DeepEqual(plan.Winget, want) {
		t.Fatalf("winget = %#v; want %#v", plan.Winget, want)
	}
	if want := []string{"custom-bucket"}; !reflect.DeepEqual(plan.ScoopBuckets, want) {
		t.Fatalf("scoop buckets = %#v; want %#v", plan.ScoopBuckets, want)
	}
	if want := []string{"fzf"}; !reflect.DeepEqual(plan.Scoop, want) {
		t.Fatalf("scoop = %#v; want %#v", plan.Scoop, want)
	}
	if want := []string{"PSReadLine"}; !reflect.DeepEqual(plan.PSModules, want) {
		t.Fatalf("psmodules = %#v; want %#v", plan.PSModules, want)
	}
}

func TestFilterCoreFeaturesDropsSetupFeatures(t *testing.T) {
	got := filterCoreFeatures([]string{"scoop", "node-pnpm", "powershell-profile"})
	if want := []string{"scoop", "node-pnpm"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("filterCoreFeatures = %#v; want %#v", got, want)
	}
}

func TestPackageItemsUseSingleWingetScanForInstallState(t *testing.T) {
	withCommandExists(t, func(name string) bool {
		return name == "winget"
	})
	r := &testutil.Runner{Outputs: map[string]string{
		"winget list --disable-interactivity": "Name      Id          Version\nGit       Git.Git     2.50.0\n",
	}}
	groups := []config.PackageGroup{{
		Key:     "cli",
		Label:   "CLI tools",
		Default: true,
		Winget:  []string{"Git.Git", "wez.wezterm"},
	}}

	items := packageItems(r, groups)

	if len(items) != 1 {
		t.Fatalf("len(items) = %d; want 1", len(items))
	}
	item := items[0]
	if !item.DefaultSelected {
		t.Fatal("default package group was not preselected")
	}
	if item.State != ui.SelectStatePartial {
		t.Fatalf("state = %q; want partial", item.State)
	}
	if item.DetailValue != "1/2 installed" {
		t.Fatalf("detail value = %q; want install summary", item.DetailValue)
	}
	if item.Detail != "2 WinGet, 0 Scoop, 0 module(s), 0 feature(s)" {
		t.Fatalf("detail = %q; want configured kind summary", item.Detail)
	}
	if want := []string{"winget list --disable-interactivity"}; !reflect.DeepEqual(r.Calls, want) {
		t.Fatalf("runner calls = %#v; want %#v", r.Calls, want)
	}
	for _, want := range []string{"Installed:\nWinGet: Git.Git", "Missing:\nWinGet: wez.wezterm"} {
		if !strings.Contains(item.Inspect, want) {
			t.Fatalf("inspect = %q; want %q", item.Inspect, want)
		}
	}
}

func TestPackageItemsWingetUnknownWhenScanUnavailable(t *testing.T) {
	withCommandExists(t, func(string) bool { return false })
	items := packageItems(&testutil.Runner{}, []config.PackageGroup{{
		Key:    "apps",
		Label:  "Apps",
		Winget: []string{"Notion.Notion"},
	}})

	if items[0].State != ui.SelectStateUnknown {
		t.Fatalf("state = %q; want unknown", items[0].State)
	}
	if items[0].DetailValue != "0/1 installed, 1 unknown" {
		t.Fatalf("detail value = %q; want unknown summary", items[0].DetailValue)
	}
	if !strings.Contains(items[0].Inspect, "Unknown:\nWinGet: Notion.Notion") {
		t.Fatalf("inspect = %q; want unknown package detail", items[0].Inspect)
	}
}

func TestPackageItemsIncludeLocalInstallStateAndInspect(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	appsDir := filepath.Join(home, "scoop", "apps", "fzf")
	if err := os.MkdirAll(appsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	withCommandExists(t, func(string) bool { return false })
	items := packageItems(&testutil.Runner{}, []config.PackageGroup{{
		Key:   "cli",
		Label: "CLI tools",
		Scoop: []string{"fzf", "ripgrep"},
	}})

	if items[0].State != ui.SelectStatePartial {
		t.Fatalf("state = %q; want partial", items[0].State)
	}
	if items[0].DetailValue != "1/2 installed" {
		t.Fatalf("detail value = %q; want install summary", items[0].DetailValue)
	}
	for _, want := range []string{"Installed:\nScoop: fzf", "Missing:\nScoop: ripgrep"} {
		if !strings.Contains(items[0].Inspect, want) {
			t.Fatalf("inspect = %q; want %q", items[0].Inspect, want)
		}
	}
}

func TestPackageItemsIncludeCommandFeatureStateAndInspect(t *testing.T) {
	withCommandExists(t, func(name string) bool {
		return name == "powershell.exe"
	})
	key := "powershell.exe -NoProfile -ExecutionPolicy Bypass -Command " + installFeatureScanCommand()
	r := &testutil.Runner{Outputs: map[string]string{
		key: "wezterm-context-menu\n",
	}}

	items := packageItems(r, []config.PackageGroup{{
		Key:      "setup",
		Label:    "Windows setup",
		Features: []string{"wezterm-context-menu", "win10-classic-menu"},
	}})

	if items[0].State != ui.SelectStatePartial {
		t.Fatalf("state = %q; want partial", items[0].State)
	}
	if items[0].DetailValue != "1/2 installed" {
		t.Fatalf("detail value = %q; want command feature summary", items[0].DetailValue)
	}
	if want := []string{key}; !reflect.DeepEqual(r.Calls, want) {
		t.Fatalf("runner calls = %#v; want %#v", r.Calls, want)
	}
	for _, want := range []string{"Installed:\nFeature: wezterm-context-menu", "Missing:\nFeature: win10-classic-menu"} {
		if !strings.Contains(items[0].Inspect, want) {
			t.Fatalf("inspect = %q; want %q", items[0].Inspect, want)
		}
	}
}

func TestCleanupItemsIncludeSizeStateAndInspect(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMP", tmp)
	t.Setenv("TEMP", tmp)
	if err := os.WriteFile(filepath.Join(tmp, "cache.tmp"), []byte("abc"), 0o644); err != nil {
		t.Fatal(err)
	}
	tasks := []config.CleanupTask{{
		Key:     "temp-files",
		Label:   "Windows Temp folder",
		Detail:  "Remove files under %TEMP%",
		Default: true,
	}}

	items := cleanupItems(&testutil.Runner{}, tasks)

	if len(items) != 1 {
		t.Fatalf("len(items) = %d; want 1", len(items))
	}
	item := items[0]
	if !item.DefaultSelected {
		t.Fatal("default cleanup task was not preselected")
	}
	if item.State != ui.SelectStatePartial {
		t.Fatalf("state = %q; want partial", item.State)
	}
	if item.DetailValue != "3 B" {
		t.Fatalf("detail value = %q; want temp size", item.DetailValue)
	}
	if !strings.Contains(item.Inspect, "Windows Temp folder: 3 B") || !strings.Contains(item.Inspect, "Command: Remove files under %TEMP%") {
		t.Fatalf("inspect = %q; want scanner and command detail", item.Inspect)
	}
}

func TestCleanupItemsIncludeTempFilesCommandSize(t *testing.T) {
	withCommandExists(t, func(name string) bool {
		return name == "powershell.exe"
	})
	key := "powershell.exe -NoProfile -ExecutionPolicy Bypass -Command " + tempFilesSizeCommand()
	r := &testutil.Runner{Outputs: map[string]string{
		key: "2048\n",
	}}

	items := cleanupItems(r, []config.CleanupTask{{
		Key:    "temp-files",
		Label:  "Windows Temp folder",
		Detail: "Remove files under %TEMP%",
	}})

	if items[0].State != ui.SelectStatePartial {
		t.Fatalf("state = %q; want partial", items[0].State)
	}
	if items[0].DetailValue != "2.0 KiB" {
		t.Fatalf("detail value = %q; want temp size", items[0].DetailValue)
	}
	if want := []string{key}; !reflect.DeepEqual(r.Calls, want) {
		t.Fatalf("runner calls = %#v; want %#v", r.Calls, want)
	}
	if !strings.Contains(items[0].Inspect, "Windows Temp folder: 2.0 KiB") {
		t.Fatalf("inspect = %q; want temp scanner detail", items[0].Inspect)
	}
}

func TestCleanupItemsIncludeRecycleBinSize(t *testing.T) {
	withCommandExists(t, func(name string) bool {
		return name == "powershell.exe"
	})
	key := "powershell.exe -NoProfile -ExecutionPolicy Bypass -Command " + recycleBinSizeCommand()
	r := &testutil.Runner{Outputs: map[string]string{
		key: "1536\n",
	}}

	items := cleanupItems(r, []config.CleanupTask{{
		Key:    "recycle-bin",
		Label:  "Recycle Bin",
		Detail: "Clear-RecycleBin -Force",
	}})

	if items[0].State != ui.SelectStatePartial {
		t.Fatalf("state = %q; want partial", items[0].State)
	}
	if items[0].DetailValue != "1.5 KiB" {
		t.Fatalf("detail value = %q; want recycle bin size", items[0].DetailValue)
	}
	if want := []string{key}; !reflect.DeepEqual(r.Calls, want) {
		t.Fatalf("runner calls = %#v; want %#v", r.Calls, want)
	}
	if !strings.Contains(items[0].Inspect, "Recycle Bin: 1.5 KiB") {
		t.Fatalf("inspect = %q; want recycle bin scanner detail", items[0].Inspect)
	}
}

func TestCleanupItemsUseNPMContentCachePayload(t *testing.T) {
	withCommandExists(t, func(name string) bool {
		return name == "npm"
	})
	root := t.TempDir()
	cache := filepath.Join(root, "_cacache")
	if err := os.MkdirAll(cache, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cache, "payload"), make([]byte, 2048), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "_logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "_logs", "debug.log"), make([]byte, 4096), 0o644); err != nil {
		t.Fatal(err)
	}
	r := &testutil.Runner{
		Outputs: map[string]string{
			"npm config get cache": root + "\n",
		},
	}

	items := cleanupItems(r, []config.CleanupTask{{
		Key:    "npm-cache",
		Label:  "npm cache",
		Detail: "npm cache clean --force",
	}})

	if items[0].DetailValue != "2.0 KiB" {
		t.Fatalf("detail value = %q; want _cacache size only", items[0].DetailValue)
	}
	if !strings.Contains(items[0].Inspect, "npm content-addressable cache: 2.0 KiB") ||
		!strings.Contains(items[0].Inspect, "npm cache root: "+root) {
		t.Fatalf("inspect = %q; want _cacache scanner detail and cache root", items[0].Inspect)
	}
	if want := []string{"npm config get cache"}; !reflect.DeepEqual(r.Calls, want) {
		t.Fatalf("runner calls = %#v; want %#v", r.Calls, want)
	}
}

func TestRunCleanupTaskUsesBestEffortPowerShellScripts(t *testing.T) {
	withCommandExists(t, func(name string) bool {
		return name == "powershell.exe"
	})
	r := &testutil.Runner{}

	if err := runCleanupTask(r, "temp-files"); err != nil {
		t.Fatal(err)
	}
	if err := runCleanupTask(r, "recycle-bin"); err != nil {
		t.Fatal(err)
	}

	if len(r.Calls) != 2 {
		t.Fatalf("calls = %#v; want two cleanup commands", r.Calls)
	}
	for _, call := range r.Calls {
		if !strings.Contains(call, "exit 0") {
			t.Fatalf("call = %q; want cleanup script to exit 0", call)
		}
	}
}

func TestCleanupItemsUnknownWithoutScanner(t *testing.T) {
	items := cleanupItems(&testutil.Runner{}, []config.CleanupTask{{
		Key:    "custom-cleanup",
		Label:  "Custom cleanup",
		Detail: "custom command",
	}})

	if items[0].State != ui.SelectStateUnknown {
		t.Fatalf("state = %q; want unknown", items[0].State)
	}
	if items[0].DetailValue != "size unknown" {
		t.Fatalf("detail value = %q; want unknown size", items[0].DetailValue)
	}
	if !strings.Contains(items[0].Inspect, "No scanner is implemented") {
		t.Fatalf("inspect = %q; want unknown scanner detail", items[0].Inspect)
	}
}

func withCommandExists(t *testing.T, fn func(string) bool) {
	t.Helper()
	orig := commandExists
	commandExists = fn
	t.Cleanup(func() {
		commandExists = orig
	})
}
