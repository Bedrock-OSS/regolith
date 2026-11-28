//go:build !windows
// +build !windows

package regolith

import "os/exec"

// batchFileCommand builds an exec.Cmd that runs the executable at path with
// the given arguments. Batch files exist only on Windows, so this is a plain
// exec.Command on other systems. See exec_windows.go.
func batchFileCommand(path string, args ...string) (*exec.Cmd, error) {
	return exec.Command(path, args...), nil
}
