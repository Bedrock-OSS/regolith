package regolith

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/otiai10/copy"
)

// Common warnings
const (
	gitNotInstalled = "Git is not installed. Git is required to download " +
		"filters.\n You can download Git from https://git-scm.com/downloads"
)

// appDataCachePath is a path to the cache directory relative to the user's
// app data
const appDataCachePath = "regolith/project-cache"

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

func ShortFilterName(name string) string {
	if strings.Contains(name, ":subfilter") {
		return strings.Split(name, ":")[0]
	}
	return name
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

// GetAbsoluteWorkingDirectory returns an absolute path to [dotRegolithPath]/tmp
func GetAbsoluteWorkingDirectory(dotRegolithPath string) string {
	absoluteWorkingDir, _ := filepath.Abs(filepath.Join(dotRegolithPath, "tmp"))
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
func RunSubProcess(command string, args []string, filterDir string, workingDir string, outputLabel string) error {
	Logger.Debugf("Exec: %s %s", command, strings.Join(args, " "))
	cmd := exec.Command(command, args...)
	cmd.Dir = workingDir
	out, _ := cmd.StdoutPipe()
	err, _ := cmd.StderrPipe()
	go LogStd(out, Logger.Infof, outputLabel)
	go LogStd(err, Logger.Errorf, outputLabel)
	env, err1 := CreateEnvironmentVariables(filterDir)
	if err1 != nil {
		return WrapErrorf(err1, "Failed to create environment variables.")
	}
	cmd.Env = env

	return cmd.Run()
}

func LogStd(in io.ReadCloser, logFunc func(template string, args ...interface{}), outputLabel string) {
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		logFunc("[%s] %s", outputLabel, scanner.Text())
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

// move moves files from source to destination. If both source and destination
// are directories, and the destination is empty, it will move thr files from
// source to destination directly (without deleting the destination first).
// Moving the subdirectories to destination one by one instead of deleting it
// and renaming entire directory is important because, the deletion of the
// destination would break observation of the destination directory.
// This function is used by MoveOrCopy.
func move(source, destination string) error {
	// Check if source and destination are directories
	sourceInfo, err1 := os.Stat(source)
	destinationInfo, err2 := os.Stat(destination)
	if err1 == nil && err2 == nil &&
		sourceInfo.IsDir() && destinationInfo.IsDir() {
		// Target must be empty
		if empty, err := isDirEmpty(destination); err != nil {
			return WrapErrorf(
				err, "Failed to check if path %s is an empty directory",
				destination)
		} else if !empty {
			return WrapErrorf(
				err,
				"Cannot move files to %s because the target directory is not "+
					"empty.",
				destination)
		}
		// Move all files in source to destination
		files, err := ioutil.ReadDir(source)
		movedFiles := make([][2]string, 100)
		movingFailed := false
		for _, file := range files {
			src := filepath.Join(source, file.Name())
			dst := filepath.Join(destination, file.Name())
			err = os.Rename(src, dst)
			if err != nil {
				Logger.Warn(
					"Failed to move content of directory %s to %s.\n"+
						"\tOperation failed while moving %s to %s.\n"+
						"Trying to recover to state before the move...",
					source, destination, src, dst)
				movingFailed = true
				break
			}
			movedFiles = append(movedFiles, [2]string{src, dst})
		}
		// If moving failed, rollback the moves
		if movingFailed {
			for _, movePair := range movedFiles {
				err = os.Rename(movePair[1], movePair[0])
				if err != nil {
					// This is a critical error that leaves the file system in
					// an invalid state. It shouldn't happen because it's from
					// moving files, that we had access to just a moment ago.
					Logger.Fatalf(
						"Regolith failed to recover from error which occured "+
							"while moving files from \"%s\" directory to "+
							"\"%s\".\n"+

							"\tRecovery failed while moving \"%s\" to "+
							"\"%s\", with error:\n\t%s\n\n"+

							"\tThis is a critical error that leaves your "+
							"files in unorganized manner.\n"+

							"\tYou can try to recover the files manually "+
							"from:\n"+
							"\t- %s\n"+
							"\t- %s\n",
						source, destination, movePair[1], movePair[0], err,
						source, destination)
				}
			}
		}
		return nil
	}
	// Either source or destination is not a directory,
	// use normal os.Rename
	err := os.Rename(source, destination)
	if err != nil {
		return err
	}
	return nil
}

// MoveOrCopy tries to move the the source to destination first and in case
// of failore it copies the files instead.
func MoveOrCopy(
	source string, destination string, makeReadOnly bool, copyParentAcl bool,
) error {
	if err := move(source, destination); err != nil {
		Logger.Infof(
			"Couldn't move files to \"%s\".\n"+
				"    Trying to copy files instead...",
			destination)
		copyOptions := copy.Options{PreserveTimes: false, Sync: false}
		err := copy.Copy(source, destination, copyOptions)
		if err != nil {
			return WrapErrorf(
				err, "Couldn't copy data files to \"%s\", aborting.",
				destination)
		}
	} else if copyParentAcl { // No errors with moving files but needs ACL copy
		parent := filepath.Dir(destination)
		if _, err := os.Stat(parent); os.IsNotExist(err) {
			return WrapError(
				err,
				"Couldn't copy ACLs - parent directory (used as a source of "+
					"ACL data) doesn't exist.")
		}
		err = copyFileSecurityInfo(parent, destination)
		if err != nil {
			return WrapErrorf(
				err,
				"Counldn't copy ACLs to the target file \"%s\".",
				destination,
			)
		}
	}
	// Make files read only if this option is selected
	if makeReadOnly {
		err := filepath.WalkDir(destination,
			func(s string, d fs.DirEntry, e error) error {
				if e != nil {
					return WrapErrorf(
						e, "Failed to walk directory \"%s\".", destination)
				}
				if !d.IsDir() {
					os.Chmod(s, 0444)
				}
				return nil
			})
		if err != nil {
			Logger.Warnf(
				"Unable to change file permissions of \"%s\" into read-only",
				destination)
		}
	}
	return nil
}

// GetDotRegolith returns the path to the directory where Regolith stores
// its cached data (like filters, Python venvs, etc.). If useAppData is set to
// false it returns relative director: ".regolith" otherwise it returns path
// inside the AppData directory. Based on the hash value of the
// project's root directory. If the path isn't .regolith it also logs a message
// which tells where the data is stored unless the silent flag is set to true.
// The projectRoot path can be relative or absolute and is resolved to an
// absolute path.
func GetDotRegolith(useAppData, silent bool, projectRoot string) (string, error) {
	// App data diabled - use .regolith
	if !useAppData {
		return ".regolith", nil
	}
	// App data enabled - use user cache dir
	userCache, err := os.UserCacheDir()
	if err != nil {
		return "", WrappedError("Unable to get user cache dir")
	}
	// Make sure that projectsRoot is an absolute path
	absoluteProjectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", WrapErrorf(
			err, "Unable to get absolute of %q.", projectRoot)
	}
	// Get the md5 of the project path
	hash := md5.New()
	hash.Write([]byte(absoluteProjectRoot))
	hashInBytes := hash.Sum(nil)
	projectPathHash := hex.EncodeToString(hashInBytes)
	// %userprofile%/AppData/Local/regolith/<md5 of project path>
	dotRegolithPath := filepath.Join(
		userCache, appDataCachePath, projectPathHash)
	if !silent {
		Logger.Infof("Regolith cache will be stored in: %s", dotRegolithPath)
	}
	return dotRegolithPath, nil
}
