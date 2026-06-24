package windows

import (
	"runtime"
	"strings"

	"github.com/gin31259461/homebase/internal/run"
	synccmd "github.com/gin31259461/homebase/internal/sync"
	"github.com/gin31259461/homebase/internal/system"
)

const ID = "windows"

var commandExists = system.CommandExists

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
	return ID
}

func (p Platform) Matches() bool {
	return runtime.GOOS == "windows"
}

func (p Platform) Bootstrap(args []string) error {
	return runBootstrap(args, p.runner)
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
