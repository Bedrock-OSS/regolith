package regolith

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func StringArrayContains(arr []string, str string) bool {
	for _, a := range arr {
		if a == str {
			return true
		}
	}
	return false
}

func wrapError(text string, err error) error {
	return errors.New(fmt.Sprintf("%s\nCaused by: %s", text, err.Error()))
}

// GetAbsoluteWorkingDirectory returns an absolute path to .regolith/tmp
func GetAbsoluteWorkingDirectory() string {
	absoluteWorkingDir, _ := filepath.Abs(".regolith/tmp")
	return absoluteWorkingDir
}

// RunSubProcess runs a sub-process with specified arguments and working
// directory
func RunSubProcess(command string, args []string, absoluteLocation string, workingDir string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "FILTER_DIR="+absoluteLocation)

	return cmd.Run()
}
