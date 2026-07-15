package main

import (
	"fmt"

	"github.com/gin31259461/homebase/internal/completion"
	"github.com/gin31259461/homebase/internal/platform"
)

func runCompletion(args []string, active platform.Platform) error {
	if len(args) != 1 || args[0] != "zsh" {
		return fmt.Errorf("usage: hb completion zsh")
	}
	script, err := completion.ZshForPlatform(active.ID(), active.SetupHooks())
	if err != nil {
		return err
	}
	fmt.Print(script)
	return nil
}
