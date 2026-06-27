package bootstrap_test

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestArchlinuxBootstrapClonesIntoCustomHomebaseDirAndForwardsEnv(t *testing.T) {
	env, logs := newArchlinuxBootstrapTestEnv(t)
	home := env.home
	customDir := filepath.Join(home, "src", "homebase-relocated")
	repo := "https://example.invalid/homebase.git"

	result := runArchlinuxBootstrap(t, env.vars, "--homebase-dir", customDir, "--homebase-repo", repo, "--yes", "--install")
	if result.exitCode != 0 {
		t.Fatalf("expected success, got exit %d stderr=%q", result.exitCode, result.stderr)
	}

	gitLog := readOptionalFile(t, filepath.Join(logs, "git.log"))
	if !strings.Contains(gitLog, "clone --depth=1 "+repo+" "+customDir) {
		t.Fatalf("expected clone log for custom dir, got %q", gitLog)
	}
	if strings.Contains(gitLog, "pull --ff-only") {
		t.Fatalf("expected clone path, got git log %q", gitLog)
	}

	goLog := readOptionalFile(t, filepath.Join(logs, "go.log"))
	expectedBin := filepath.Join(home, ".local", "bin", "hb")
	if !strings.Contains(goLog, "-C "+customDir+" -o "+expectedBin+" ./cmd/hb") {
		t.Fatalf("expected go build in relocated dir, got %q", goLog)
	}

	hbArgs := strings.TrimSpace(readOptionalFile(t, filepath.Join(logs, "hb-args.txt")))
	if hbArgs != "bootstrap --yes --install" {
		t.Fatalf("expected forwarded hb args, got %q", hbArgs)
	}

	hbHomebaseDir := strings.TrimSpace(readOptionalFile(t, filepath.Join(logs, "hb-homebase-dir.txt")))
	if hbHomebaseDir != customDir {
		t.Fatalf("expected HOMEBASE_DIR %q, got %q", customDir, hbHomebaseDir)
	}
}

func TestArchlinuxBootstrapRefusesToReplaceNonEmptyCustomHomebaseDir(t *testing.T) {
	env, logs := newArchlinuxBootstrapTestEnv(t)
	customDir := filepath.Join(env.home, "src", "keep-me")
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sentinel := filepath.Join(customDir, "sentinel.txt")
	if err := os.WriteFile(sentinel, []byte("preserve"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := runArchlinuxBootstrap(t, env.vars, "--homebase-dir", customDir, "--yes")
	if result.exitCode == 0 {
		t.Fatalf("expected failure for non-empty custom dir, stdout=%q", result.stdout)
	}
	if !strings.Contains(result.stderr, "refusing to replace non-empty custom HOMEBASE_DIR") {
		t.Fatalf("expected safety error, got stderr=%q", result.stderr)
	}

	if got := string(mustReadFile(t, sentinel)); got != "preserve" {
		t.Fatalf("expected sentinel to remain unchanged, got %q", got)
	}

	gitLog := readOptionalFile(t, filepath.Join(logs, "git.log"))
	if strings.Contains(gitLog, "clone") || strings.Contains(gitLog, "pull --ff-only") {
		t.Fatalf("expected no git mutation, got %q", gitLog)
	}
	if _, err := os.Stat(filepath.Join(logs, "hb-args.txt")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected hb not to run, stat err=%v", err)
	}
}

func TestArchlinuxBootstrapUsesExistingSourceCheckout(t *testing.T) {
	env, logs := newArchlinuxBootstrapTestEnv(t)
	customDir := filepath.Join(env.home, "src", "existing-source")
	if err := os.MkdirAll(customDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(customDir, "go.mod"), []byte("module example.com/homebase\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := runArchlinuxBootstrap(t, env.vars, "--homebase-dir", customDir, "--yes")
	if result.exitCode != 0 {
		t.Fatalf("expected success, got exit %d stderr=%q", result.exitCode, result.stderr)
	}
	if !strings.Contains(result.stdout, "Using existing Homebase source at "+customDir) {
		t.Fatalf("expected source reuse message, got stdout=%q", result.stdout)
	}

	gitLog := readOptionalFile(t, filepath.Join(logs, "git.log"))
	if gitLog != "" {
		t.Fatalf("expected no git operations for existing source, got %q", gitLog)
	}

	hbHomebaseDir := strings.TrimSpace(readOptionalFile(t, filepath.Join(logs, "hb-homebase-dir.txt")))
	if hbHomebaseDir != customDir {
		t.Fatalf("expected HOMEBASE_DIR %q, got %q", customDir, hbHomebaseDir)
	}
}

func TestArchlinuxBootstrapPullsExistingGitCheckout(t *testing.T) {
	env, logs := newArchlinuxBootstrapTestEnv(t)
	customDir := filepath.Join(env.home, "src", "existing-git")
	if err := os.MkdirAll(filepath.Join(customDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	result := runArchlinuxBootstrap(t, env.vars, "--homebase-dir", customDir, "--yes")
	if result.exitCode != 0 {
		t.Fatalf("expected success, got exit %d stderr=%q", result.exitCode, result.stderr)
	}

	gitLog := readOptionalFile(t, filepath.Join(logs, "git.log"))
	if !strings.Contains(gitLog, "-C "+customDir+" pull --ff-only") {
		t.Fatalf("expected git pull for existing checkout, got %q", gitLog)
	}
	if strings.Contains(gitLog, "clone") {
		t.Fatalf("expected no clone for existing checkout, got %q", gitLog)
	}

	hbHomebaseDir := strings.TrimSpace(readOptionalFile(t, filepath.Join(logs, "hb-homebase-dir.txt")))
	if hbHomebaseDir != customDir {
		t.Fatalf("expected HOMEBASE_DIR %q, got %q", customDir, hbHomebaseDir)
	}
}

type archlinuxBootstrapEnv struct {
	home string
	vars []string
}

type scriptResult struct {
	stdout   string
	stderr   string
	exitCode int
}

func newArchlinuxBootstrapTestEnv(t *testing.T) (archlinuxBootstrapEnv, string) {
	t.Helper()

	root := t.TempDir()
	home := filepath.Join(root, "home")
	logs := filepath.Join(root, "logs")
	fakeBin := filepath.Join(root, "fake-bin")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(logs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(fakeBin, 0o755); err != nil {
		t.Fatal(err)
	}

	hbStub := filepath.Join(root, "hb-stub")
	writeExecutable(t, hbStub, `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" > "$TEST_LOG_DIR/hb-args.txt"
printf '%s\n' "${HOMEBASE_DIR:-}" > "$TEST_LOG_DIR/hb-homebase-dir.txt"
`)

	writeExecutable(t, filepath.Join(fakeBin, "sudo"), `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "$TEST_LOG_DIR/sudo.log"
`)

	writeExecutable(t, filepath.Join(fakeBin, "git"), `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "$TEST_LOG_DIR/git.log"
if [[ "${1:-}" == "clone" ]]; then
  target="${@: -1}"
  mkdir -p "$target/.git"
  : > "$target/go.mod"
fi
`)

	writeExecutable(t, filepath.Join(fakeBin, "go"), `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$*" >> "$TEST_LOG_DIR/go.log"
out=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    -o)
      out="$2"
      shift 2
      ;;
    *)
      shift
      ;;
  esac
done
[[ -n "$out" ]] || exit 1
mkdir -p "$(dirname "$out")"
cp "$TEST_HB_STUB" "$out"
chmod +x "$out"
`)

	env := archlinuxBootstrapEnv{
		home: home,
		vars: []string{
			"HOME=" + home,
			"PATH=" + fakeBin + string(os.PathListSeparator) + os.Getenv("PATH"),
			"HOMEBASE_BOOTSTRAP_SKIP_PLATFORM_CHECK=1",
			"TEST_HB_STUB=" + hbStub,
			"TEST_LOG_DIR=" + logs,
		},
	}
	return env, logs
}

func runArchlinuxBootstrap(t *testing.T, env []string, args ...string) scriptResult {
	t.Helper()

	if runtime.GOOS == "windows" {
		t.Skip("archlinux bootstrap tests require bash")
	}
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("bash not available")
	}

	scriptPath, err := filepath.Abs("archlinux.sh")
	if err != nil {
		t.Fatal(err)
	}

	cmdArgs := append([]string{scriptPath}, args...)
	cmd := exec.Command("bash", cmdArgs...)
	cmd.Env = env
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	result := scriptResult{
		stdout: stdout.String(),
		stderr: stderr.String(),
	}

	err = cmd.Run()
	result.stdout = stdout.String()
	result.stderr = stderr.String()
	if err == nil {
		return result
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		result.exitCode = exitErr.ExitCode()
		return result
	}
	t.Fatalf("run archlinux bootstrap: %v", err)
	return scriptResult{}
}

func writeExecutable(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o755); err != nil {
		t.Fatal(err)
	}
}

func readOptionalFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return ""
	}
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
