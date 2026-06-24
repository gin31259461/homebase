package setup

import "os/exec"

func systemExecLookPath(file string) (string, error) {
	return exec.LookPath(file)
}
