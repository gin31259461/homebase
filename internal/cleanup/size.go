package cleanup

import (
	"fmt"

	"github.com/gin31259461/homebase/internal/ui"
)

const SmallCleanupBytes int64 = 10 * 1024 * 1024

// const MediumCleanupBytes int64 = 100 * 1024 * 1024

func CleanupSizeState(bytes int64) ui.SelectState {
	switch {
	case bytes == 0:
		return ui.SelectStateGood
	case bytes < SmallCleanupBytes:
		return ui.SelectStatePartial
	default:
		return ui.SelectStateBad
	}
}

func FormatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}
	units := []string{"B", "KiB", "MiB", "GiB", "TiB"}
	value := float64(bytes)
	unit := 0
	for value >= 1024 && unit < len(units)-1 {
		value /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%d %s", bytes, units[unit])
	}
	return fmt.Sprintf("%.1f %s", value, units[unit])
}
