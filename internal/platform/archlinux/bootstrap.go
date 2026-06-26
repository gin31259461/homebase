package archlinux

import (
	"github.com/gin31259461/homebase/internal/run"
	"github.com/gin31259461/homebase/internal/ui"
)

func installBasics(r run.Runner, pkgs []string) error {
	if len(pkgs) == 0 {
		return nil
	}
	var missing []string
	for _, pkg := range pkgs {
		if !pacmanInstalled(r, pkg) {
			missing = append(missing, pkg)
		}
	}
	if len(missing) == 0 {
		ui.OK("Bootstrap packages already installed")
		return nil
	}
	ui.Section("Installing bootstrap packages")
	args := append([]string{"pacman", "-S", "--needed", "--noconfirm"}, missing...)
	return r.Run("sudo", args...)
}
