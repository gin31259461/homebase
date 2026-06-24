package install

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
	official, aur := InstallPlan(groups, []string{"core", "dev"}, installed)

	if want := []string{"go", "make"}; !reflect.DeepEqual(official, want) {
		t.Fatalf("official = %#v; want %#v", official, want)
	}
	if want := []string{"tool-bin", "yay-bin"}; !reflect.DeepEqual(aur, want) {
		t.Fatalf("aur = %#v; want %#v", aur, want)
	}
}

func TestUniqueKnown(t *testing.T) {
	got := UniqueKnown([]string{"core", "missing", "core", "", "dev"}, map[string]bool{"core": true, "dev": true})
	if want := []string{"core", "dev"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("UniqueKnown = %#v; want %#v", got, want)
	}
}

func TestPackageItemsExposeStateAndDefaults(t *testing.T) {
	groups := []config.PackageGroup{
		{Key: "core", Label: "Core", Default: true, Pacman: []string{"git", "go"}},
		{Key: "dev", Label: "Dev", Pacman: []string{"make"}},
	}
	items := PackageItems(groups, map[string]bool{"git": true})
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

func TestRunWithPlatformDoesNotRequireAURHelperWithoutAURPlan(t *testing.T) {
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
	if err := RunWithPlatform([]string{"--group", "core", "--yes", "--no-setup"}, r, "archlinux"); err != nil {
		t.Fatal(err)
	}
	for _, call := range r.Calls {
		if strings.Contains(call, "homebase-missing-aur-helper") || strings.Contains(call, "aur.archlinux.org/yay.git") {
			t.Fatalf("AUR helper should not be used without AUR packages, calls = %#v", r.Calls)
		}
	}
}

func TestRunWithPlatformRunsSetupAfterInstallingMissingPackage(t *testing.T) {
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
	if err := RunWithPlatform([]string{"--group", "docker", "--yes"}, r, "archlinux"); err != nil {
		t.Fatal(err)
	}
	if !hasCall(r.Calls, "sudo systemctl enable --now docker.service") {
		t.Fatalf("docker setup did not run after package install, calls = %#v", r.Calls)
	}
}

func writeInstallDefaults(t *testing.T, home, configTOML, packagesTOML string) {
	t.Helper()
	base := filepath.Join(home, ".local", "lib", "homebase", "config")
	writeTestFile(t, filepath.Join(base, "homebase.toml"), `active_platform = "auto"`)
	writeTestFile(t, filepath.Join(base, "platforms", "archlinux", "config.toml"), configTOML)
	writeTestFile(t, filepath.Join(base, "platforms", "archlinux", "cleanup.toml"), ``)
	writeTestFile(t, filepath.Join(base, "platforms", "archlinux", "sync.toml"), ``)
	writeTestFile(t, filepath.Join(base, "platforms", "archlinux", "packages.d", "10-test.toml"), packagesTOML)
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
