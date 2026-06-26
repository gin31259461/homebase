package archlinux

import (
	"os"
	"strings"

	"github.com/gin31259461/homebase/internal/bootstrap"
	"github.com/gin31259461/homebase/internal/run"
	synccmd "github.com/gin31259461/homebase/internal/sync"
)

const ID = "archlinux"

type Platform struct {
	runner run.Runner
}

type stringList []string

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
	return bootstrap.RunWithPlatform(args, p.runner, ID, bootstrap.Hooks{
		InstallBasics: installBasics,
		RunInstall:    runInstall,
	})
}

func (p Platform) Install(args []string) error {
	return runInstall(args, p.runner)
}

func (p Platform) Cleanup(args []string) error {
	return runCleanup(args, p.runner)
}

func (p Platform) Sync(args []string) error {
	return synccmd.RunWithPlatform(args, p.runner, ID)
}

func (s *stringList) String() string { return strings.Join(*s, ",") }
func (s *stringList) Set(v string) error {
	*s = append(*s, v)
	return nil
}
