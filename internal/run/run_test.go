package run

import (
	"errors"
	"testing"
)

func TestExitCode(t *testing.T) {
	t.Run("wrapped exit code error", func(t *testing.T) {
		err := errors.New("plain")
		wrapped := wrappedExitCodeError{err: err, code: 7}
		got, ok := ExitCode(wrapped)
		if !ok || got != 7 {
			t.Fatalf("ExitCode() = %d, %v; want 7, true", got, ok)
		}
	})

	t.Run("plain error", func(t *testing.T) {
		got, ok := ExitCode(errors.New("plain"))
		if ok || got != 0 {
			t.Fatalf("ExitCode() = %d, %v; want 0, false", got, ok)
		}
	})
}

func TestDisplayCommandRedactsSensitiveArgs(t *testing.T) {
	tests := []struct {
		name string
		cmd  string
		args []string
		want string
	}{
		{
			name: "redacts next arg for password flag",
			cmd:  "tool",
			args: []string{"--password", "hunter2", "--user", "abner"},
			want: "$ tool --password <redacted> --user abner",
		},
		{
			name: "redacts assignment value",
			cmd:  "tool",
			args: []string{"--password=hunter2"},
			want: "$ tool --password=<redacted>",
		},
		{
			name: "redacts nmcli psk value",
			cmd:  "nmcli",
			args: []string{"con", "modify", "Arch-Hyprland", "wifi-sec.psk", "ilovearchlinux"},
			want: "$ nmcli con modify Arch-Hyprland wifi-sec.psk <redacted>",
		},
		{
			name: "keeps non sensitive args",
			cmd:  "git",
			args: []string{"status", "--short"},
			want: "$ git status --short",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := displayCommand(tt.cmd, tt.args...); got != tt.want {
				t.Fatalf("displayCommand() = %q; want %q", got, tt.want)
			}
		})
	}
}

func TestRedactArgsDoesNotMutateInput(t *testing.T) {
	args := []string{"--password", "hunter2"}
	_ = redactArgs(args)
	if got := args[1]; got != "hunter2" {
		t.Fatalf("args[1] = %q; want original value", got)
	}
}

type wrappedExitCodeError struct {
	err  error
	code int
}

func (e wrappedExitCodeError) Error() string {
	return e.err.Error()
}

func (e wrappedExitCodeError) Unwrap() error {
	return e.err
}

func (e wrappedExitCodeError) ExitCode() int {
	return e.code
}
