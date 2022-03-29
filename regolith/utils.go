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
	"strconv"
	"strings"

	"github.com/fatih/color"
)

// Common warnings
const (
	gitNotInstalled = "Git is not installed. Git is required to download " +
		"filters.\n You can download Git from https://git-scm.com/downloads"
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

// firstErr returns the first error in a list of errors. If the list is empty
// or all errors are nil, nil is returned.
func firstErr(errors ...error) error {
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}

// wrapErrorStackTrace is used by other wrapped error functions to add a stack
// trace to the error message.
func wrapErrorStackTrace(err error, text string) error {
	text = strings.Replace(text, "\n", color.YellowString("\n   >> "), -1)

	if err != nil {
		text = fmt.Sprintf(
			"%s\n[%s]: %s", text, color.RedString("+"), err.Error())
	}
	if printStackTraces {
		pc, fn, line, _ := runtime.Caller(2)
		text = fmt.Sprintf(
			"%s\n   [%s] %s:%d", text, runtime.FuncForPC(pc).Name(),
			filepath.Base(fn), line)
	}
	return errors.New(text)
}

func FullFilterToNiceFilterName(name string) string {
	if strings.Contains(name, ":subfilter") {
		i, err := strconv.Atoi(strings.Split(name, ":subfilter")[1])
		if err != nil {
			return fmt.Sprintf("the \"%s\" filter", name)
		}
		return NiceFilterName(strings.Split(name, ":")[0], i)
	}
	return fmt.Sprintf("the \"%s\" filter", name)
}

func NiceFilterName(name string, i int) string {
	return fmt.Sprintf("the %s subfilter of \"%s\" filter", nth(i), name)
}

// PassError adds stack trace to an error without any additional text.
func PassError(err error) error {
	text := err.Error()
	if printStackTraces {
		pc, fn, line, _ := runtime.Caller(1)
		text = fmt.Sprintf(
			"%s\n   [%s] %s:%d", text, runtime.FuncForPC(pc).Name(),
			filepath.Base(fn), line)
	}
	return errors.New(text)
}

// NotImplementedError is used by default functions, that need implementation.
func NotImplementedError(text string) error {
	text = fmt.Sprintf("Function not implemented: %s", text)
	return wrapErrorStackTrace(nil, text)
}

// VersionMismatchError is used when cached filter version doesn't match the one required by config.
func VersionMismatchError(id string, requiredVersion string, cachedVersion string) error {
	text := fmt.Sprintf(
		"Installation missmatch for '%s' detected.\n"+
			"Installed version: %s\n"+
			"Required version: %s\n"+
			"Update the filter using: 'regolith update %[1]s'",
		id, cachedVersion, requiredVersion)
	return wrapErrorStackTrace(nil, text)
}

// WrappedError creates an error with a stack trace from text.
func WrappedError(text string) error {
	return wrapErrorStackTrace(nil, text)
}

// WrappedErrorf creates an error with a stack trace from formatted text.
func WrappedErrorf(text string, args ...interface{}) error {
	text = fmt.Sprintf(text, args...)
	return wrapErrorStackTrace(nil, text)
}

// WrapError wraps an error with a stack trace and adds additional text
// information.
func WrapError(err error, text string) error {
	return wrapErrorStackTrace(err, text)
}

// WrapErrorf wraps an error with a stack trace and adds additional formatted
// text information.
func WrapErrorf(err error, text string, args ...interface{}) error {
	return wrapErrorStackTrace(err, fmt.Sprintf(text, args...))
}

func CreateDirectoryIfNotExists(directory string, mustSucceed bool) error {
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		err = os.MkdirAll(directory, 0666)
		if err != nil {
			if mustSucceed {
				return WrapErrorf(
					err, "Failed to create directory %s.", directory)
			} else {
				Logger.Warnf(
					"Failed to create directory %s: %s.", directory,
					err.Error())
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

// CreateEnvironmentVariables creates an array of environment variables including custom ones
func CreateEnvironmentVariables(filterDir string) ([]string, error) {
	projectDir, err := os.Getwd()
	if err != nil {
		return nil, WrapErrorf(err, "Failed to get current working directory.")
	}
	projectDir, err = filepath.Abs(projectDir)
	if err != nil {
		return nil, WrapErrorf(err, "Failed to get absolute path to current working directory.")
	}
	return append(os.Environ(), "FILTER_DIR="+filterDir, "ROOT_DIR="+projectDir), nil
}

// RunSubProcess runs a sub-process with specified arguments and working
// directory
func RunSubProcess(command string, args []string, filterDir string, workingDir string) error {
	Logger.Debugf("Exec: %s %s", command, strings.Join(args, " "))
	cmd := exec.Command(command, args...)
	cmd.Dir = workingDir
	out, _ := cmd.StdoutPipe()
	err, _ := cmd.StderrPipe()
	go LogStd(out, Logger.Infof)
	go LogStd(err, Logger.Errorf)
	env, err1 := CreateEnvironmentVariables(filterDir)
	if err1 != nil {
		return WrapErrorf(err1, "Failed to create environment variables.")
	}
	cmd.Env = env

	return cmd.Run()
}

func LogStd(in io.ReadCloser, logFunc func(template string, args ...interface{})) {
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		logFunc("[Filter] %s", scanner.Text())
	}
}

// isDirEmpty checks whether the path points at empty directory. If the path
// is not a directory or info about the path can't be obtaioned for some reason
// it returns false. If the path is a directory and it is empty, it returns
// true.
func isDirEmpty(path string) (bool, error) {
	if stat, err := os.Stat(path); os.IsNotExist(err) {
		return false, WrappedErrorf("Path %q does not exist.", path)
	} else if !stat.IsDir() {
		return false, WrappedErrorf("Path %q is not a directory.", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return false, WrapErrorf(err, "Failed to open %q.", path)
	}
	defer f.Close()
	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	} else if err != nil {
		return false, PassError(err)
	}
	// err is nil -> not empty
	return false, nil
}
