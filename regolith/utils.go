package regolith

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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

// nth returns the ordinal numeral of the index of a table. For example:
// nth(0) returns "1st", nth(1) returns "2nd", etc.
func nth(i int) string {
	i += 1
	j := i % 100
	if j > 10 && j < 20 {
		return fmt.Sprintf("%dth", i)
	}
	switch j % 10 {
	case 1:
		return fmt.Sprintf("%dst", i)
	case 2:
		return fmt.Sprintf("%dnd", i)
	case 3:
		return fmt.Sprintf("%drd", i)
	}
	return fmt.Sprintf("%dth", i)
}

func wrapErrorStackTrace(err error, text string) error {
	if err != nil {
		text = fmt.Sprintf(
			"%s\n[%s]: %s", text, color.RedString("+"), err.Error())
	}
	if printStackTraces {
		pc, fn, line, _ := runtime.Caller(2)
		text = fmt.Sprintf("%s\n    %s; %s:%d", text, runtime.FuncForPC(pc).Name(), fn, line)
	}
	return errors.New(text)
}

func WrapError(err error, text string) error {
	return wrapErrorStackTrace(err, text)
}

func WrapErrorf(err error, text string, args ...interface{}) error {
	return wrapErrorStackTrace(err, fmt.Sprintf(text, args...))
}

func CreateDirectoryIfNotExists(directory string, mustSucceed bool) error {
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		err = os.MkdirAll(directory, 0666)
		if err != nil {
			if mustSucceed {
				return WrapErrorf(
					err, "Failed to create directory %s", directory)
			} else {
				Logger.Warnf("Failed to create directory %s: %s", directory, err.Error())
				return nil
			}
		}
	}
	return nil
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
