package install

import (
	"reflect"
	"testing"

	"github.com/gin31259461/homebase/internal/config"
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
