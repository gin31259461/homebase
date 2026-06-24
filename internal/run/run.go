package run

import (
	"bytes"
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
	fmt.Printf("$ %s %s\n", name, strings.Join(args, " "))
	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (ExecRunner) RunIn(dir, name string, args ...string) error {
	fmt.Printf("$ %s %s\n", name, strings.Join(args, " "))
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
