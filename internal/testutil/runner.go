package testutil

import (
	"errors"
	"strings"
)

type Runner struct {
	Outputs map[string]string
	Errors  map[string]error
	Calls   []string
}

func (r *Runner) Run(name string, args ...string) error {
	return r.record(name, args...)
}

func (r *Runner) RunIn(dir, name string, args ...string) error {
	return r.record("cd "+dir+" && "+name, args...)
}

func (r *Runner) Quiet(name string, args ...string) error {
	return r.record(name, args...)
}

func (r *Runner) QuietIn(dir, name string, args ...string) error {
	return r.record("cd "+dir+" && "+name, args...)
}

func (r *Runner) Capture(name string, args ...string) (string, error) {
	key := commandKey(name, args...)
	r.Calls = append(r.Calls, key)
	if err := r.Errors[key]; err != nil {
		return "", err
	}
	return r.Outputs[key], nil
}

func (r *Runner) record(name string, args ...string) error {
	key := commandKey(name, args...)
	r.Calls = append(r.Calls, key)
	if err := r.Errors[key]; err != nil {
		return err
	}
	return nil
}

func commandKey(name string, args ...string) string {
	if len(args) == 0 {
		return name
	}
	return name + " " + strings.Join(args, " ")
}

func Err() error {
	return errors.New("fake error")
}

type ExitCodeError struct {
	code int
	err  error
}

func (e ExitCodeError) Error() string {
	return e.err.Error()
}

func (e ExitCodeError) Unwrap() error {
	return e.err
}

func (e ExitCodeError) ExitCode() int {
	return e.code
}

func ExitError(code int) error {
	return ExitCodeError{
		code: code,
		err:  errors.New("fake exit error"),
	}
}
