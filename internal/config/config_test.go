package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadPackageGroupsFromDir(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "20-dev.toml"), `[dev]
label = "Dev"
pacman = ["go"]
`)
	writeFile(t, filepath.Join(dir, "10-core.toml"), `[core]
label = "Core"
aur = ["yay-bin"]
`)
	groups, err := LoadPackageGroupsFromDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	got := []string{groups[0].Key, groups[1].Key}
	if want := []string{"core", "dev"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("group keys = %#v; want %#v", got, want)
	}
}

func TestLoadPackageGroupsRejectsDuplicate(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "10-a.toml"), `[core]
label = "Core"
`)
	writeFile(t, filepath.Join(dir, "20-b.toml"), `[core]
label = "Core Again"
`)
	if _, err := LoadPackageGroupsFromDir(dir); err == nil {
		t.Fatal("expected duplicate key error")
	}
}

func TestLoadCleanupTasksFiltersRequires(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cleanup.toml")
	writeFile(t, path, `[keep]
label = "Keep"
detail = "available"
requires = "git"

[skip]
label = "Skip"
detail = "missing"
requires = "missing"
`)
	tasks, err := LoadCleanupTasksFromFile(path, func(name string) bool { return name == "git" })
	if err != nil {
		t.Fatal(err)
	}
	if len(tasks) != 1 || tasks[0].Key != "keep" {
		t.Fatalf("tasks = %#v", tasks)
	}
}

func TestLoadSyncPathsDedupesStableOrder(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sync.toml")
	writeFile(t, path, `[b]
paths = ["two", "one"]

[a]
paths = ["one", "zero"]
`)
	paths, err := LoadSyncPathsFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"one", "zero", "two"}; !reflect.DeepEqual(paths, want) {
		t.Fatalf("paths = %#v; want %#v", paths, want)
	}
}

func TestLoadGlobal(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".config", "homebase")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "homebase.toml"), `active_platform = "auto"

[platform_aliases]
arch = "archlinux"
`)
	global, err := LoadGlobal()
	if err != nil {
		t.Fatal(err)
	}
	if global.ActivePlatform != "auto" {
		t.Fatalf("active platform = %q", global.ActivePlatform)
	}
	if global.PlatformAliases["arch"] != "archlinux" {
		t.Fatalf("aliases = %#v", global.PlatformAliases)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
