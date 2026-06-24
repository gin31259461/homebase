package main

import (
	"fmt"
	"os"

	"github.com/gin31259461/homebase/internal/bootstrap"
	"github.com/gin31259461/homebase/internal/cleanup"
	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/install"
	synccmd "github.com/gin31259461/homebase/internal/sync"
	"github.com/gin31259461/homebase/internal/ui"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	var err error
	switch os.Args[1] {
	case "bootstrap":
		err = bootstrap.Run(os.Args[2:])
	case "install":
		err = install.Run(os.Args[2:])
	case "cleanup":
		err = cleanup.Run(os.Args[2:])
	case "sync":
		err = synccmd.Run(os.Args[2:])
	case "config":
		err = runConfig(os.Args[2:])
	case "help", "-h", "--help":
		usage()
	default:
		err = fmt.Errorf("unknown command: %s", os.Args[1])
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "%s %v\n", ui.WarnStyle.Render("error:"), err)
		os.Exit(1)
	}
}

func runConfig(args []string) error {
	if len(args) != 1 || args[0] != "init" {
		return fmt.Errorf("usage: hb config init")
	}
	return config.Ensure(true)
}

func usage() {
	fmt.Println(`hb - Homebase dotfile and Arch setup manager

Usage:
  hb bootstrap [--yes] [--repo <repo>] [--install]
  hb install   [--group <key>] [--all] [--yes] [--no-setup]
  hb cleanup   [--task <key>] [--all] [--yes]
  hb sync      [-m <message>] [--no-push]
  hb config init

Interactive commands use Bubble Tea by default. Automation should pass --yes
with explicit --group/--task selections or --all.`)
}
