package windows

import (
	"reflect"
	"testing"

	"github.com/gin31259461/homebase/internal/config"
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
