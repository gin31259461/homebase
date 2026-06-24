package gitutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin31259461/homebase/internal/config"
	"github.com/gin31259461/homebase/internal/run"
	"github.com/pelletier/go-toml/v2"
)

type DotfilesMemory struct {
	Repo   string `toml:"repo"`
	Branch string `toml:"branch"`
}

func DotArgs(cfg config.App, args ...string) []string {
	out := []string{"git", "--git-dir=" + cfg.Dotfiles.Dir, "--work-tree=" + config.Expand("~")}
	return append(out, args...)
}

func NormalizeRepo(input string) (string, string, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return "", "", errors.New("empty repository URL")
	}
	if strings.Count(input, "/") == 1 && !strings.Contains(input, ":") {
		slug := strings.TrimSuffix(input, ".git")
		return "git@github.com:" + slug + ".git", "https://github.com/" + slug + ".git", nil
	}
	if strings.HasPrefix(input, "git@github.com:") {
		slug := strings.TrimSuffix(strings.TrimPrefix(input, "git@github.com:"), ".git")
		return "git@github.com:" + slug + ".git", "https://github.com/" + slug + ".git", nil
	}
	if strings.HasPrefix(input, "https://github.com/") {
		slug := strings.TrimSuffix(strings.TrimPrefix(input, "https://github.com/"), ".git")
		return "git@github.com:" + slug + ".git", "https://github.com/" + slug + ".git", nil
	}
	if strings.HasPrefix(input, "git@") {
		return input, "", nil
	}
	return "", "", fmt.Errorf("unsupported repository format: %s", input)
}

func ReadRepoMemory(path string) (DotfilesMemory, error) {
	var mem DotfilesMemory
	b, err := os.ReadFile(path)
	if err != nil {
		return mem, err
	}
	text := strings.TrimSpace(string(b))
	if text == "" {
		return mem, nil
	}
	if strings.Contains(text, "=") {
		if err := toml.Unmarshal(b, &mem); err != nil {
			return mem, err
		}
		return mem, nil
	}
	mem.Repo = text
	return mem, nil
}

func SaveRepoMemory(path, repo, branch string) error {
	mem := DotfilesMemory{Repo: repo, Branch: branch}
	b, err := toml.Marshal(mem)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func RemoteHeadAvailable(r run.Runner, repo string) bool {
	return r.Quiet(
		"git",
		"-c", "core.sshCommand=ssh -o BatchMode=yes -o ConnectTimeout=5",
		"ls-remote", "--exit-code", repo, "HEAD",
	) == nil
}
