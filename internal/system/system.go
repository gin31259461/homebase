package system

import (
	"errors"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/gin31259461/homebase/internal/run"
)

func RequireArch() error {
	if _, err := os.Stat("/etc/arch-release"); err != nil {
		return errors.New("this command targets Arch Linux only")
	}
	return nil
}

func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func PacmanInstalled(r run.Runner, pkg string) bool {
	return r.Quiet("pacman", "-Qi", pkg) == nil
}

func InstalledPackageSet(r run.Runner) (map[string]bool, error) {
	output, err := r.Capture("pacman", "-Qq")
	if err != nil {
		return nil, err
	}
	return ParseInstalledPackages(output), nil
}

func ParseInstalledPackages(output string) map[string]bool {
	installed := map[string]bool{}
	for _, pkg := range strings.Fields(output) {
		installed[pkg] = true
	}
	return installed
}

func SystemActive(r run.Runner, unit string) bool {
	return r.Quiet("systemctl", "is-active", "--quiet", unit) == nil
}

func UserInGroup(r run.Runner, user, group string) bool {
	out, err := r.Capture("groups", user)
	return err == nil && ContainsWord(out, group)
}

func ContainsWord(text, word string) bool {
	for _, field := range strings.Fields(text) {
		if field == word {
			return true
		}
	}
	return false
}

func LatestOpenRazerModule(root string) (string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return "", err
	}
	var names []string
	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "openrazer-driver-") {
			names = append(names, entry.Name())
		}
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "", nil
	}
	version := strings.TrimPrefix(names[len(names)-1], "openrazer-driver-")
	return "openrazer-driver/" + version, nil
}
