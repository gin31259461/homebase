package archlinux

import (
	"os"

	"github.com/gin31259461/homebase/internal/bootstrap"
	"github.com/gin31259461/homebase/internal/cleanup"
	"github.com/gin31259461/homebase/internal/install"
	"github.com/gin31259461/homebase/internal/run"
	synccmd "github.com/gin31259461/homebase/internal/sync"
)

const ID = "archlinux"

type Platform struct {
	runner run.Runner
}

func New() Platform {
	return Platform{runner: run.New()}
}

func (p Platform) ID() string {
	return ID
}

func (p Platform) Family() string {
	return "archlinux"
}

func (p Platform) Matches() bool {
	if _, err := os.Stat("/etc/arch-release"); err == nil {
		return true
	}
	if _, err := os.Stat("/etc/manjaro-release"); err == nil {
		return true
	}
	return false
}

func (p Platform) Bootstrap(args []string) error {
	return bootstrap.RunWithPlatform(args, p.runner, ID)
}

func (p Platform) Install(args []string) error {
	return install.RunWithPlatform(args, p.runner, ID)
}

func (p Platform) Cleanup(args []string) error {
	return cleanup.RunWithPlatform(args, p.runner, ID)
}

func (p Platform) Sync(args []string) error {
	return synccmd.RunWithPlatform(args, p.runner, ID)
}
