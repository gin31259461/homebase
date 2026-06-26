package cleanup

import (
	"path/filepath"
	"testing"

	"github.com/gin31259461/homebase/internal/testutil"
	"github.com/gin31259461/homebase/internal/ui"
)

func TestCleanupSizeStateThreshold(t *testing.T) {
	tests := []struct {
		name  string
		bytes int64
		want  ui.SelectState
	}{
		{name: "empty", bytes: 0, want: ui.SelectStateGood},
		{name: "small", bytes: SmallCleanupBytes - 1, want: ui.SelectStatePartial},
		{name: "large", bytes: SmallCleanupBytes, want: ui.SelectStateBad},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CleanupSizeState(tt.bytes); got != tt.want {
				t.Fatalf("CleanupSizeState(%d) = %s; want %s", tt.bytes, got, tt.want)
			}
		})
	}
}

func TestNPMCachePayloadPath(t *testing.T) {
	if got, want := NPMCachePayloadPath("/tmp/npm"), filepath.Join("/tmp/npm", "_cacache"); got != want {
		t.Fatalf("NPMCachePayloadPath = %q; want %q", got, want)
	}
	if got := NPMCachePayloadPath(""); got != "" {
		t.Fatalf("NPMCachePayloadPath empty = %q; want empty", got)
	}
}

func TestNPMCacheRootUsesConfiguredCache(t *testing.T) {
	root := t.TempDir()
	r := &testutil.Runner{
		Outputs: map[string]string{
			"npm config get cache": root + "\n",
		},
	}
	if got := NPMCacheRoot(r, "~/.npm"); got != root {
		t.Fatalf("NPMCacheRoot = %q; want %q", got, root)
	}
}
