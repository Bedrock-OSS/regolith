package regolith

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
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
	if err != nil {
		return fmt.Errorf("%s\n[%s]: %s", text, color.RedString("+"), err.Error())
	}
	return errors.New(text)
}

func CreateDirectoryIfNotExists(directory string, mustSucceed bool) {
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		err = os.MkdirAll(directory, 0666)
		if err != nil {
			if mustSucceed {
				Logger.Fatalf("Failed to create directory %s: %s", directory, err.Error())
			} else {
				Logger.Warnf("Failed to create directory %s: %s", directory, err.Error())
			}
		}
	}
}

// GetAbsoluteWorkingDirectory returns an absolute path to .regolith/tmp
func GetAbsoluteWorkingDirectory() string {
	absoluteWorkingDir, _ := filepath.Abs(".regolith/tmp")
	return absoluteWorkingDir
}

// RunSubProcess runs a sub-process with specified arguments and working
// directory
func RunSubProcess(command string, args []string, absoluteLocation string, workingDir string) error {
	Logger.Debugf("Exec: %s %s", command, strings.Join(args, " "))
	cmd := exec.Command(command, args...)
	cmd.Dir = workingDir
	out, _ := cmd.StdoutPipe()
	err, _ := cmd.StderrPipe()
	go LogStd(out, Logger.Infof)
	go LogStd(err, Logger.Errorf)
	cmd.Env = append(os.Environ(), "FILTER_DIR="+absoluteLocation)

	return cmd.Run()
}

func LogStd(in io.ReadCloser, logFunc func(template string, args ...interface{})) {
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		logFunc("[Filter] %s", scanner.Text())
	}
}
