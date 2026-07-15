package completion

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestZshCompletesCommandsFlagsAndConfiguredKeys(t *testing.T) {
	script := zsh(
		[]candidate{{key: "shell", label: "Shell & Prompt"}},
		[]candidate{{key: "npm-cache", label: "npm cache"}},
		[]candidate{{key: "system", label: "System and mkinitcpio"}},
	)

	for _, want := range []string{
		"#compdef hb",
		"'setup:run or repair setup hooks'",
		"'*--group[select a package group]:package group:->groups'",
		"'*--hook[select a setup hook]:setup hook:->hooks'",
		"'*--task[select a cleanup task]:cleanup task:->tasks'",
		"'shell:Shell & Prompt'",
		"'npm-cache:npm cache'",
		"'system:System and mkinitcpio'",
	} {
		if !strings.Contains(script, want) {
			t.Fatalf("Zsh completion is missing %q\n%s", want, script)
		}
	}
}

func TestZshCompletionHasValidSyntax(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh is not installed")
	}
	script := zsh(
		[]candidate{{key: "owner", label: "Owner's tools"}},
		[]candidate{{key: "cache", label: "Cache"}},
		[]candidate{{key: "shell", label: "Shell"}},
	)
	path := filepath.Join(t.TempDir(), "_hb")
	if err := os.WriteFile(path, []byte(script), 0o644); err != nil {
		t.Fatal(err)
	}
	if output, err := exec.Command("zsh", "-n", path).CombinedOutput(); err != nil {
		t.Fatalf("zsh -n: %v\n%s", err, output)
	}
}

func TestZshCompletionDispatchesSubcommandArguments(t *testing.T) {
	if _, err := exec.LookPath("zsh"); err != nil {
		t.Skip("zsh is not installed")
	}
	script := zsh(
		[]candidate{{key: "shell", label: "Shell"}},
		[]candidate{{key: "cache", label: "Cache"}},
		[]candidate{{key: "system", label: "System"}},
	)
	harness := `
_arguments() { print -r -- "arguments:$words[1]:$CURRENT:$*" }
_describe() { print -r -- "describe:$1" }
_values() { print -r -- "values:$*" }

words=(hb install --)
CURRENT=3
_hb || print -r -- "dispatch-failed:install"

words=(hb setup --)
CURRENT=3
_hb || print -r -- "dispatch-failed:setup"

words=(hb config init --)
CURRENT=4
_hb
`
	path := filepath.Join(t.TempDir(), "completion-dispatch.zsh")
	if err := os.WriteFile(path, []byte(script+harness), 0o644); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command("zsh", path).CombinedOutput()
	if err != nil {
		t.Fatalf("completion harness: %v\n%s", err, output)
	}
	for _, want := range []string{
		"arguments:install:2:",
		"--group[select a package group]",
		"arguments:init:2:",
		"--force[overwrite existing config]",
	} {
		if !strings.Contains(string(output), want) {
			t.Fatalf("completion dispatch output is missing %q\n%s", want, output)
		}
	}
	if strings.Contains(string(output), "dispatch-failed") {
		t.Fatalf("completion returned failure after adding argument matches\n%s", output)
	}
}
