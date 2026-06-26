package archlinux

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

func TestInstallPlanDedupesAndSkipsInstalled(t *testing.T) {
	groups := []config.PackageGroup{
		{Key: "core", Pacman: []string{"git", "go"}, AUR: []string{"yay-bin"}},
		{Key: "dev", Pacman: []string{"go", "make"}, AUR: []string{"yay-bin", "tool-bin"}},
	}
	installed := map[string]bool{"git": true}
	official, aur := installPlan(groups, []string{"core", "dev"}, installed)

	if want := []string{"go", "make"}; !reflect.DeepEqual(official, want) {
		t.Fatalf("official = %#v; want %#v", official, want)
	}
	if want := []string{"tool-bin", "yay-bin"}; !reflect.DeepEqual(aur, want) {
		t.Fatalf("aur = %#v; want %#v", aur, want)
	}
}

func TestPackageItemsExposeStateAndDefaults(t *testing.T) {
	groups := []config.PackageGroup{
		{Key: "core", Label: "Core", Default: true, Pacman: []string{"git", "go"}},
		{Key: "dev", Label: "Dev", Pacman: []string{"make"}},
	}
	items := packageItems(groups, map[string]bool{"git": true})
	if !items[0].DefaultSelected {
		t.Fatal("default package group was not preselected")
	}
	if items[0].State != ui.SelectStatePartial {
		t.Fatalf("core state = %s; want partial", items[0].State)
	}
	if items[0].DetailValue != "1/2 installed, 2 pacman, 0 AUR" {
		t.Fatalf("core detail value = %q; want install summary", items[0].DetailValue)
	}
	if items[1].State != ui.SelectStateBad {
		t.Fatalf("dev state = %s; want bad", items[1].State)
	}
}

func TestArchPackageManagerDefaults(t *testing.T) {
	pm := archPackageManager(config.PackageManager{})
	if pm.Official != "pacman" || pm.AUR != "yay" {
		t.Fatalf("archPackageManager = %#v; want pacman/yay", pm)
	}
}

func TestRunInstallDoesNotRequireAURHelperWithoutAURPlan(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeInstallDefaults(t, home, `[package_manager]
official = "pacman"
aur = "homebase-missing-aur-helper"
`, `[core]
label = "Core"
pacman = ["git"]
`)
	r := &testutil.Runner{
		Outputs: map[string]string{
			"pacman -Qq": "git\n",
		},
	}
	if err := runInstall([]string{"--group", "core", "--yes", "--no-setup"}, r); err != nil {
		t.Fatal(err)
	}
	for _, call := range r.Calls {
		if strings.Contains(call, "homebase-missing-aur-helper") || strings.Contains(call, "aur.archlinux.org/yay.git") {
			t.Fatalf("AUR helper should not be used without AUR packages, calls = %#v", r.Calls)
		}
	}
}

func TestRunInstallRunsSetupAfterInstallingMissingPackage(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USER", "tester")
	writeInstallDefaults(t, home, ``, `[docker]
label = "Docker"
pacman = ["docker"]
`)
	r := &testutil.Runner{
		Outputs: map[string]string{
			"pacman -Qq":    "",
			"groups tester": "tester",
		},
	}
	if err := runInstall([]string{"--group", "docker", "--yes"}, r); err != nil {
		t.Fatal(err)
	}
	if !hasCall(r.Calls, "sudo systemctl enable --now docker.service") {
		t.Fatalf("docker setup did not run after package install, calls = %#v", r.Calls)
	}
}

func TestParseInstalledPackages(t *testing.T) {
	got := parseInstalledPackages("git\nbase-devel\n  go\n")
	for _, pkg := range []string{"git", "base-devel", "go"} {
		if !got[pkg] {
			t.Fatalf("missing package %q in %#v", pkg, got)
		}
	}
}

func writeInstallDefaults(t *testing.T, home, configTOML, packagesTOML string) {
	t.Helper()
	base := filepath.Join(home, ".local", "lib", "homebase", "config")
	writeTestFile(t, filepath.Join(base, "homebase.toml"), `active_platform = "auto"`)
	writeTestFile(t, filepath.Join(base, "platforms", ID, "config.toml"), configTOML)
	writeTestFile(t, filepath.Join(base, "platforms", ID, "cleanup.toml"), ``)
	writeTestFile(t, filepath.Join(base, "platforms", ID, "sync.toml"), ``)
	writeTestFile(t, filepath.Join(base, "platforms", ID, "packages.d", "10-test.toml"), packagesTOML)
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasCall(calls []string, want string) bool {
	for _, call := range calls {
		if call == want {
			return true
		}
	}
	return false
}
