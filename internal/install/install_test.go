package install

import (
	"reflect"
	"testing"

	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/ui"
)

func TestUniqueKnown(t *testing.T) {
	got := UniqueKnown([]string{"core", "missing", "core", "", "dev"}, map[string]bool{"core": true, "dev": true})
	if want := []string{"core", "dev"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("UniqueKnown = %#v; want %#v", got, want)
	}
}

func TestPackageGroupHelpers(t *testing.T) {
	groups := []config.PackageGroup{{Key: "core"}, {Key: "dev"}}
	if got, want := GroupKeys(groups), []string{"core", "dev"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("GroupKeys = %#v; want %#v", got, want)
	}
	set := PackageGroupSet(groups)
	if !set["core"] || !set["dev"] || set["missing"] {
		t.Fatalf("PackageGroupSet = %#v; want core/dev only", set)
	}
}

func TestInstallState(t *testing.T) {
	tests := []struct {
		installed int
		total     int
		want      ui.SelectState
	}{
		{installed: 0, total: 0, want: ui.SelectStateGood},
		{installed: 2, total: 2, want: ui.SelectStateGood},
		{installed: 0, total: 2, want: ui.SelectStateBad},
		{installed: 1, total: 2, want: ui.SelectStatePartial},
	}
	for _, tt := range tests {
		if got := InstallState(tt.installed, tt.total); got != tt.want {
			t.Fatalf("InstallState(%d, %d) = %s; want %s", tt.installed, tt.total, got, tt.want)
		}
	}
}
