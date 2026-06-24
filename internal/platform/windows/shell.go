package windows

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/run"
)

func ensureUserPath(r run.Runner, path string) error {
	addProcessPath(path)
	cmd := "$path = " + quotePS(path) + "; " +
		"$current = [Environment]::GetEnvironmentVariable('Path','User'); " +
		"$parts = @($current -split ';' | Where-Object { $_ }); " +
		"if ($parts -notcontains $path) { [Environment]::SetEnvironmentVariable('Path', (($parts + $path) -join ';'), 'User') }"
	return r.Run(powerShellExe(), "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command", cmd)
}

func addProcessPath(path string) {
	if strings.TrimSpace(path) == "" {
		return
	}
	parts := filepath.SplitList(os.Getenv("PATH"))
	for _, part := range parts {
		if strings.EqualFold(filepath.Clean(part), filepath.Clean(path)) {
			return
		}
	}
	_ = os.Setenv("PATH", os.Getenv("PATH")+string(os.PathListSeparator)+path)
}

func hbBinDir() string {
	return filepath.Join(config.Expand("~"), ".local", "bin")
}

func scoopAvailable() bool {
	if commandExists("scoop") {
		return true
	}
	if _, err := os.Stat(filepath.Join(config.Expand("~"), "scoop")); err == nil {
		addProcessPath(filepath.Join(config.Expand("~"), "scoop", "shims"))
		return true
	}
	return false
}

func powerShellExe() string {
	if commandExists("powershell.exe") {
		return "powershell.exe"
	}
	return "powershell"
}

func quotePS(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}
