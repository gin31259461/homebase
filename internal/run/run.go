package run

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

type Runner interface {
	Run(name string, args ...string) error
	RunIn(dir, name string, args ...string) error
	Quiet(name string, args ...string) error
	QuietIn(dir, name string, args ...string) error
	Capture(name string, args ...string) (string, error)
}

type ExecRunner struct{}

func New() ExecRunner {
	return ExecRunner{}
}

func (ExecRunner) Run(name string, args ...string) error {
	fmt.Println(displayCommand(name, args...))
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (ExecRunner) RunIn(dir, name string, args ...string) error {
	fmt.Println(displayCommand(name, args...))
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (ExecRunner) Quiet(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

func (ExecRunner) QuietIn(dir, name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	return cmd.Run()
}

func (ExecRunner) Capture(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = io.Discard
	err := cmd.Run()
	return out.String(), err
}

type exitCoder interface {
	ExitCode() int
}

func ExitCode(err error) (int, bool) {
	if err == nil {
		return 0, false
	}
	var coder exitCoder
	if !errors.As(err, &coder) {
		return 0, false
	}
	code := coder.ExitCode()
	if code < 0 {
		return 0, false
	}
	return code, true
}

func displayCommand(name string, args ...string) string {
	redacted := redactArgs(args)
	if len(redacted) == 0 {
		return "$ " + name
	}
	return "$ " + name + " " + strings.Join(redacted, " ")
}

func redactArgs(args []string) []string {
	redacted := append([]string(nil), args...)
	redactNext := false
	for i, arg := range redacted {
		if redactNext {
			redacted[i] = "<redacted>"
			redactNext = false
			continue
		}
		if key, ok := sensitiveAssignmentKey(arg); ok {
			redacted[i] = key + "=<redacted>"
			continue
		}
		if isSensitiveArgName(arg) {
			redactNext = true
		}
	}
	return redacted
}

func sensitiveAssignmentKey(arg string) (string, bool) {
	key, _, ok := strings.Cut(arg, "=")
	if !ok || !isSensitiveArgName(key) {
		return "", false
	}
	return key, true
}

func isSensitiveArgName(arg string) bool {
	key := strings.TrimLeft(strings.ToLower(arg), "-/")
	switch {
	case key == "password":
		return true
	case key == "passwd":
		return true
	case key == "passphrase":
		return true
	case key == "psk":
		return true
	case strings.HasSuffix(key, ".password"):
		return true
	case strings.HasSuffix(key, ".passwd"):
		return true
	case strings.HasSuffix(key, ".passphrase"):
		return true
	case strings.HasSuffix(key, ".psk"):
		return true
	default:
		return false
	}
}
