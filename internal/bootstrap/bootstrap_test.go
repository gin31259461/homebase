package bootstrap

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/gitutil"
	"github.com/gin31259461/homebase/internal/testutil"
)

func TestRememberClonedRemoteFailsWithoutPersistingMemory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := testBootstrapConfig(home)
	sshRepo := "git@github.com:user/repo.git"
	httpsRepo := "https://github.com/user/repo.git"
	setURLErr := errors.New("set-url failed")
	remoteArgs := gitutil.DotArgs(cfg, "remote", "set-url", "origin", sshRepo)
	r := &testutil.Runner{
		Errors: map[string]error{
			strings.Join(remoteArgs, " "): setURLErr,
		},
	}

	err := rememberClonedRemote(r, cfg, httpsRepo, sshRepo)
	if !errors.Is(err, setURLErr) {
		t.Fatalf("rememberClonedRemote error = %v; want %v", err, setURLErr)
	}
	if _, err := os.Stat(cfg.Dotfiles.MemoryFile); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("repo memory stat error = %v; want not exist", err)
	}
	if len(r.Calls) != 1 || r.Calls[0] != strings.Join(remoteArgs, " ") {
		t.Fatalf("runner calls = %#v; want only remote set-url", r.Calls)
	}
}

func TestRememberClonedRemotePersistsSSHRepoAfterRewrite(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg := testBootstrapConfig(home)
	sshRepo := "git@github.com:user/repo.git"
	httpsRepo := "https://github.com/user/repo.git"
	remoteArgs := gitutil.DotArgs(cfg, "remote", "set-url", "origin", sshRepo)
	r := &testutil.Runner{}

	if err := rememberClonedRemote(r, cfg, httpsRepo, sshRepo); err != nil {
		t.Fatalf("rememberClonedRemote error = %v", err)
	}

	mem, err := gitutil.ReadRepoMemory(cfg.Dotfiles.MemoryFile)
	if err != nil {
		t.Fatalf("ReadRepoMemory error = %v", err)
	}
	if mem.Repo != sshRepo || mem.Branch != cfg.Dotfiles.Branch {
		t.Fatalf("repo memory = %#v; want repo=%q branch=%q", mem, sshRepo, cfg.Dotfiles.Branch)
	}
	if len(r.Calls) != 1 || r.Calls[0] != strings.Join(remoteArgs, " ") {
		t.Fatalf("runner calls = %#v; want only remote set-url", r.Calls)
	}
}

func testBootstrapConfig(home string) config.App {
	var cfg config.App
	cfg.Dotfiles.Dir = filepath.Join(home, ".dotfiles")
	cfg.Dotfiles.MemoryFile = filepath.Join(home, ".dotfiles-repo")
	cfg.Dotfiles.Branch = "main"
	return cfg
}
