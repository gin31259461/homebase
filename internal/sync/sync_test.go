package sync

import (
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
			diffCall: testutil.ExitError(1),
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

func TestRunFailsWhenDiffCheckErrors(t *testing.T) {
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
			diffCall: testutil.ExitError(2),
		},
	}

	err := RunWithPlatform([]string{"-m", "sync test", "--no-push"}, r, "archlinux")
	if err == nil {
		t.Fatal("RunWithPlatform() error = nil; want failure")
	}
	if !strings.Contains(err.Error(), "check staged changes") {
		t.Fatalf("RunWithPlatform() error = %q; want diff context", err)
	}
}

func TestResolveCommitMessage(t *testing.T) {
	tests := []struct {
		name      string
		flagValue string
		prompt    func() string
		want      string
		wantOK    bool
	}{
		{
			name:      "uses flag value",
			flagValue: " chore: sync ",
			prompt:    func() string { return "ignored" },
			want:      "chore: sync",
			wantOK:    true,
		},
		{
			name:      "uses prompt value",
			flagValue: "",
			prompt:    func() string { return " update config " },
			want:      "update config",
			wantOK:    true,
		},
		{
			name:      "empty prompt aborts",
			flagValue: "",
			prompt:    func() string { return "" },
			want:      "",
			wantOK:    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := resolveCommitMessage(tt.flagValue, tt.prompt)
			if got != tt.want || ok != tt.wantOK {
				t.Fatalf("resolveCommitMessage() = %q, %v; want %q, %v", got, ok, tt.want, tt.wantOK)
			}
		})
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
