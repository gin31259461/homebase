package platform

import (
	"fmt"

	"github.com/gin31259461/homebase/internal/config"
)

type Platform interface {
	ID() string
	Family() string
	Matches() bool
	Bootstrap(args []string) error
	Install(args []string) error
	Cleanup(args []string) error
	Sync(args []string) error
}

func Detect(platforms []Platform) (Platform, error) {
	if err := config.Ensure(false); err != nil {
		return nil, err
	}
	global, err := config.LoadGlobal()
	if err != nil {
		return nil, err
	}
	if global.ActivePlatform != "auto" {
		id := ResolveAlias(global.ActivePlatform, global.PlatformAliases)
		for _, p := range platforms {
			if p.ID() == id || p.Family() == id {
				return p, nil
			}
		}
		return nil, fmt.Errorf("configured platform %q is not available", global.ActivePlatform)
	}
	for _, p := range platforms {
		if p.Matches() {
			return p, nil
		}
	}
	return nil, fmt.Errorf("no supported platform detected")
}

func ResolveAlias(id string, aliases map[string]string) string {
	if aliases == nil {
		return id
	}
	if resolved, ok := aliases[id]; ok {
		return resolved
	}
	return id
}
