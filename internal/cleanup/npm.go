package cleanup

import (
	"path/filepath"
	"strings"

	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/run"
)

func NPMCacheRoot(r run.Runner, fallback string) string {
	out, err := r.Capture("npm", "config", "get", "cache")
	if err == nil {
		root := strings.TrimSpace(out)
		if root != "" && root != "undefined" && root != "null" {
			return config.Expand(root)
		}
	}
	if strings.TrimSpace(fallback) == "" {
		return ""
	}
	return config.Expand(fallback)
}

func NPMCachePayloadPath(root string) string {
	if strings.TrimSpace(root) == "" {
		return ""
	}
	return filepath.Join(root, "_cacache")
}

func RunNPMCacheClean(r run.Runner) error {
	return r.Run("npm", "cache", "clean", "--force")
}
