package install

import (
	"strings"

	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/ui"
)

func GroupKeys(groups []config.PackageGroup) []string {
	keys := make([]string, 0, len(groups))
	for _, group := range groups {
		keys = append(keys, group.Key)
	}
	return keys
}

func PackageGroupSet(groups []config.PackageGroup) map[string]bool {
	set := map[string]bool{}
	for _, group := range groups {
		set[group.Key] = true
	}
	return set
}

func UniqueKnown(values []string, known map[string]bool) []string {
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

func InstallState(installed, total int) ui.SelectState {
	if total == 0 {
		return ui.SelectStateGood
	}
	switch {
	case installed == total:
		return ui.SelectStateGood
	case installed == 0:
		return ui.SelectStateBad
	default:
		return ui.SelectStatePartial
	}
}
