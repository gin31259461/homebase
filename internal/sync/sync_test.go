package sync

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin31259461/homebase/internal/testutil"
)

func TestRunUsesHomeAsGitWorkingDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	writeFile(t, filepath.Join(home, ".local", "lib", "homebase", "config", "homebase.toml"), `active_platform = "auto"`)
	writeFile(t, filepath.Join(home, ".local", "lib", "homebase", "config", "platforms", "archlinux", "config.toml"), `[dotfiles]
dir = "~/.dotfiles"
branch = "main"
memory_file = "~/.dotfiles-repo"
`)
	writeFile(t, filepath.Join(home, ".local", "lib", "homebase", "config", "platforms", "archlinux", "sync.toml"), `[shell]
paths = [".zshrc"]
`)
	writeFile(t, filepath.Join(home, ".local", "lib", "homebase", "config", "platforms", "archlinux", "cleanup.toml"), ``)
	writeFile(t, filepath.Join(home, ".local", "lib", "homebase", "config", "platforms", "archlinux", "packages.d", "base.toml"), `[core]
label = "Core"
`)

	diffCall := "cd " + home + " && git --git-dir=" + filepath.Join(home, ".dotfiles") + " --work-tree=" + home + " diff --cached --quiet"
	r := &testutil.Runner{
		Errors: map[string]error{
			diffCall: errors.New("changes staged"),
		},
	}
	if err := RunWithPlatform([]string{"-m", "sync test", "--no-push"}, r, "archlinux"); err != nil {
		t.Fatal(err)
	}

	for _, call := range r.Calls {
		if strings.Contains(call, " git ") && !strings.HasPrefix(call, "cd "+home+" && ") {
			t.Fatalf("git call did not run from home: %s", call)
		}
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
