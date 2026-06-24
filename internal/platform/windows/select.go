package windows

import (
	"strings"

	"github.com/gin31259461/homebase/internal/ui"
)

func uniqueKnown(values []string, known map[string]bool) []string {
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
