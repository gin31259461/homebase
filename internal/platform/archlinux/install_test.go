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
			"lsmod":         "Module Size Used by\nexample 123 1\n",
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

func TestSystemSetupConfiguresMkinitcpioForAMDGPU(t *testing.T) {
	r := &installFileRunner{Runner: testutil.Runner{
		Outputs: map[string]string{
			"lsmod":                    "Module Size Used by\namdgpu 123456 42\n",
			"cat /etc/mkinitcpio.conf": "# mkinitcpio configuration\nMODULES=()\nHOOKS=(base udev autodetect)\n",
		},
	}}

	if err := systemSetup(r); err != nil {
		t.Fatal(err)
	}
	want := "# mkinitcpio configuration\nMODULES=(usbhid xhci_pci amdgpu)\nHOOKS=(base udev autodetect)\n"
	if r.installedContent != want {
		t.Fatalf("installed mkinitcpio.conf = %q; want %q", r.installedContent, want)
	}
	if !hasCall(r.Calls, "sudo mkinitcpio -P") {
		t.Fatalf("mkinitcpio image was not rebuilt, calls = %#v", r.Calls)
	}
}

func TestSystemSetupLeavesMkinitcpioUnchangedForDifferentGPU(t *testing.T) {
	r := &testutil.Runner{Outputs: map[string]string{
		"lsmod": "Module Size Used by\nnvidia 123456 42\n",
	}}

	output := captureStdout(t, func() {
		if err := systemSetup(r); err != nil {
			t.Fatal(err)
		}
	})
	for _, call := range r.Calls {
		if strings.Contains(call, mkinitcpioConfigPath) || strings.Contains(call, "mkinitcpio -P") {
			t.Fatalf("non-amdgpu setup must not change mkinitcpio, calls = %#v", r.Calls)
		}
	}
	if !strings.Contains(output, mkinitcpioConfigPath) || !strings.Contains(output, "blank screen") {
		t.Fatalf("warning = %q; want manual configuration path and blank-screen risk", output)
	}
}

func TestSystemSetupLeavesMkinitcpioUnchangedWhenGPUScanFails(t *testing.T) {
	r := &testutil.Runner{Errors: map[string]error{"lsmod": testutil.Err()}}

	output := captureStdout(t, func() {
		if err := systemSetup(r); err != nil {
			t.Fatal(err)
		}
	})
	if len(r.Calls) != 1 || r.Calls[0] != "lsmod" {
		t.Fatalf("failed scan must not change mkinitcpio, calls = %#v", r.Calls)
	}
	if !strings.Contains(output, "Unable to detect") || !strings.Contains(output, "blank screen") {
		t.Fatalf("warning = %q; want detection failure and blank-screen risk", output)
	}
}

func TestSetMkinitcpioModulesAppendsMissingSetting(t *testing.T) {
	got := setMkinitcpioModules("HOOKS=(base udev)\n")
	want := "HOOKS=(base udev)\nMODULES=(usbhid xhci_pci amdgpu)\n"
	if got != want {
		t.Fatalf("setMkinitcpioModules = %q; want %q", got, want)
	}
}

func TestRunSetupCommandReinstallsZshCompletion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeInstallDefaults(t, home, ``, `[shell]
label = "Shell & Prompt"
pacman = ["zsh"]
`)
	r := &completionInstallRunner{Runner: testutil.Runner{Outputs: map[string]string{
		"pacman -Qq": "zsh\n",
	}}}

	if err := runSetupCommand([]string{"--hook", "shell", "--yes"}, r); err != nil {
		t.Fatal(err)
	}
	completion := r.installed[zshCompletionPath]
	if !strings.Contains(completion, "#compdef hb") || !strings.Contains(completion, "'shell:Shell & Prompt'") {
		t.Fatalf("installed completion = %q; want hb definition with configured shell group", completion)
	}
}

func TestInstallBasicsInstallsZshCompletion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeInstallDefaults(t, home, ``, `[shell]
label = "Shell & Prompt"
pacman = ["zsh"]
`)
	if err := config.EnsureForPlatform(ID, false); err != nil {
		t.Fatal(err)
	}
	r := &completionInstallRunner{Runner: testutil.Runner{}}

	if err := installBasics(r, []string{"zsh"}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(r.installed[zshCompletionPath], "#compdef hb") {
		t.Fatalf("bootstrap did not install hb completion, calls = %#v", r.Calls)
	}
	if !hasCall(r.Calls, "pacman -Qi zsh") {
		t.Fatalf("bootstrap did not check zsh package, calls = %#v", r.Calls)
	}
}

func TestSetupHookItemsShowMissingAndUnknownPrerequisites(t *testing.T) {
	hooks := setupHooks()
	missing := setupHookItems(hooks, map[string]bool{})
	unknown := setupHookItems(hooks, nil)

	var missingShell, unknownShell ui.SelectItem
	for i := range hooks {
		if hooks[i].Key == "shell" {
			missingShell = missing[i]
			unknownShell = unknown[i]
			break
		}
	}
	if missingShell.State != ui.SelectStateBad || missingShell.DetailValue != "missing zsh" {
		t.Fatalf("missing shell item = %#v", missingShell)
	}
	if unknownShell.State != ui.SelectStateUnknown || unknownShell.DetailValue != "prerequisite unknown" {
		t.Fatalf("unknown shell item = %#v", unknownShell)
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

type installFileRunner struct {
	testutil.Runner
	installedContent string
}

type completionInstallRunner struct {
	testutil.Runner
	installed map[string]string
}

func (r *completionInstallRunner) Run(name string, args ...string) error {
	if name == "sudo" && len(args) == 4 && args[0] == "install" && args[1] == "-Dm644" {
		content, err := os.ReadFile(args[2])
		if err != nil {
			return err
		}
		if r.installed == nil {
			r.installed = map[string]string{}
		}
		r.installed[args[3]] = string(content)
	}
	return r.Runner.Run(name, args...)
}

func (r *installFileRunner) Run(name string, args ...string) error {
	if name == "sudo" && len(args) == 4 && args[0] == "install" && args[1] == "-Dm644" && args[3] == mkinitcpioConfigPath {
		content, err := os.ReadFile(args[2])
		if err != nil {
			return err
		}
		r.installedContent = string(content)
	}
	return r.Runner.Run(name, args...)
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "stdout-*")
	if err != nil {
		t.Fatal(err)
	}
	original := os.Stdout
	os.Stdout = f
	t.Cleanup(func() { os.Stdout = original })
	fn()
	os.Stdout = original
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	output, err := os.ReadFile(f.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(output)
}
