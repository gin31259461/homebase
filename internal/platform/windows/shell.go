package windows

import (
	"fmt"
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

func ensureNodeNPMOnPath() error {
	if commandExists("npm") {
		return nil
	}
	for _, path := range nodeJSPathCandidates() {
		if !hasAnyFile(path, "npm.cmd", "npm.exe", "npm") {
			continue
		}
		addProcessPath(path)
		if commandExists("npm") {
			return nil
		}
	}
	return fmt.Errorf("npm not found in PATH after installing Node.js")
}

func nodeJSPathCandidates() []string {
	var paths []string
	add := func(base string, elems ...string) {
		if strings.TrimSpace(base) == "" {
			return
		}
		all := append([]string{base}, elems...)
		paths = append(paths, filepath.Join(all...))
	}
	add(os.Getenv("ProgramFiles"), "nodejs")
	add(os.Getenv("LocalAppData"), "Programs", "nodejs")
	add(os.Getenv("ProgramFiles(x86)"), "nodejs")
	return paths
}

func hasAnyFile(dir string, names ...string) bool {
	for _, name := range names {
		if _, err := os.Stat(filepath.Join(dir, name)); err == nil {
			return true
		}
	}
	return false
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
