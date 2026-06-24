package install

import (
	"reflect"
	"testing"

	"github.com/gin31259461/homebase/internal/config"
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
