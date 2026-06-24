package gitutil

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin31259461/homebase/internal/testutil"
)

func TestNormalizeRepo(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantSSH   string
		wantHTTPS string
		wantErr   bool
	}{
		{"short github", "user/repo", "git@github.com:user/repo.git", "https://github.com/user/repo.git", false},
		{"github ssh", "git@github.com:user/repo.git", "git@github.com:user/repo.git", "https://github.com/user/repo.git", false},
		{"github https", "https://github.com/user/repo.git", "git@github.com:user/repo.git", "https://github.com/user/repo.git", false},
		{"other ssh", "git@example.com:user/repo.git", "git@example.com:user/repo.git", "", false},
		{"bad", "https://example.com/user/repo.git", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSSH, gotHTTPS, err := NormalizeRepo(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if gotSSH != tt.wantSSH || gotHTTPS != tt.wantHTTPS {
				t.Fatalf("NormalizeRepo() = %q, %q; want %q, %q", gotSSH, gotHTTPS, tt.wantSSH, tt.wantHTTPS)
			}
		})
	}
}

func TestRepoMemoryPlainTextAndTOML(t *testing.T) {
	dir := t.TempDir()
	plain := filepath.Join(dir, "plain")
	if err := os.WriteFile(plain, []byte("git@github.com:user/repo.git\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mem, err := ReadRepoMemory(plain)
	if err != nil {
		t.Fatal(err)
	}
	if mem.Repo != "git@github.com:user/repo.git" {
		t.Fatalf("plain repo = %q", mem.Repo)
	}

	tomlPath := filepath.Join(dir, "repo.toml")
	if err := SaveRepoMemory(tomlPath, "git@github.com:user/repo.git", "main"); err != nil {
		t.Fatal(err)
	}
	mem, err = ReadRepoMemory(tomlPath)
	if err != nil {
		t.Fatal(err)
	}
	if mem.Repo != "git@github.com:user/repo.git" || mem.Branch != "main" {
		t.Fatalf("toml memory = %#v", mem)
	}
}

func TestRemoteHeadAvailable(t *testing.T) {
	repo := "git@github.com:user/repo.git"
	if !RemoteHeadAvailable(&testutil.Runner{}, repo) {
		t.Fatal("expected available repo when runner command succeeds")
	}
	r := &testutil.Runner{
		Errors: map[string]error{
			"git -c core.sshCommand=ssh -o BatchMode=yes -o ConnectTimeout=5 ls-remote --exit-code git@github.com:user/repo.git HEAD": errors.New("unavailable"),
		},
	}
	if RemoteHeadAvailable(r, repo) {
		t.Fatal("expected unavailable repo when runner command fails")
	}
}
