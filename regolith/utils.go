package regolith

import (
	"bufio"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Bedrock-OSS/go-burrito/burrito"
	"github.com/nightlyone/lockfile"
)

// appDataCachePath is a path to the cache directory relative to the user's
// app data
const appDataCachePath = "regolith/project-cache"

var Version = "unversioned"

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

func FullFilterToNiceFilterName(name string) string {
	if strings.Contains(name, ":subfilter") {
		i, err := strconv.Atoi(strings.Split(name, ":subfilter")[1])
		if err != nil {
			return fmt.Sprintf("the \"%s\" filter", name)
		}
		return NiceSubfilterName(strings.Split(name, ":")[0], i)
	}
	return fmt.Sprintf("the \"%s\" filter", name)
}

func ShortFilterName(name string) string {
	if strings.Contains(name, ":subfilter") {
		return strings.Split(name, ":")[0]
	}
	return name
}

func NiceSubfilterName(name string, i int) string {
	return fmt.Sprintf("the %s subfilter of \"%s\" filter", nth(i), name)
}

// NotImplementedError is used by default functions, that need implementation.
func NotImplementedError(text string) error {
	text = fmt.Sprintf("Function not implemented: %s", text)
	return burrito.WrappedError(text)
}

// CreateDirectoryIfNotExists creates a directory if it doesn't exist. If
// the directory already exists, it does nothing and returns nil.
func CreateDirectoryIfNotExists(directory string) error {
	if _, err := os.Stat(directory); os.IsNotExist(err) {
		err = os.MkdirAll(directory, 0755)
		if err != nil {
			// Error outside of this function should tell about the path
			return burrito.PassError(err)
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
		return nil, burrito.WrapErrorf(err, osGetwdError)
	}
	return append(os.Environ(), fmt.Sprintf("FILTER_DIR=%s", filterDir), fmt.Sprintf("ROOT_DIR=%s", projectDir), fmt.Sprintf("DEBUG=%t", burrito.PrintStackTrace)), nil
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
		return burrito.WrapErrorf(
			err1,
			"Failed to create FILTER_DIR and ROOT_DIR environment variables.")
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

// getAppDataDotRegolith gets the dotRegolithPath from th app data folder
func getAppDataDotRegolith(silent bool, projectRoot string) (string, error) {
	// App data enabled - use user cache dir
	userCache, err := os.UserCacheDir()
	if err != nil {
		return "", burrito.WrappedError(osUserCacheDirError)
	}
	// Make sure that projectsRoot is an absolute path
	absoluteProjectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", burrito.WrapErrorf(err, filepathAbsError, projectRoot)
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
		Logger.Infof(
			"Regolith project cache is in:\n\t%s",
			dotRegolithPath)
	}
	return dotRegolithPath, nil
}

// GetDotRegolith returns the path to the directory where Regolith stores
// its cached data (like filters, Python venvs, etc.). If user config setting
// for using app data by profiles is set to false it returns relative
// directory: ".regolith" otherwise it returns path inside the AppData directory.
// Based on the hash value of the project's root directory. If the path isn't
// .regolith it also logs a message which tells where the data is stored
// unless the silent flag is set to true. The projectRoot path can be relative
// or absolute and is resolved to an
// absolute path.
func GetDotRegolith(silent bool, projectRoot string) (string, error) {
	// App data disabled - use .regolith
	userConfig, err := getCombinedUserConfig()
	if err != nil {
		return "", burrito.WrapError(err, getUserConfigError)
	}
	if !*userConfig.UseProjectAppDataStorage {
		return ".regolith", nil
	}
	return getAppDataDotRegolith(silent, projectRoot)
}

// acquireSessionLock creates a lock file in specified directory and
// returns a function that releases the lock.
// The path should point to the .regolith directory.
func acquireSessionLock(dotRegolithPath string) (func() error, error) {
	// Create dotRegolithPath if it doesn't exist
	err := CreateDirectoryIfNotExists(dotRegolithPath)
	if err != nil {
		return nil, burrito.WrapErrorf(err, osMkdirError, dotRegolithPath)
	}
	// Get the session lock
	sessionLockPath, err := filepath.Abs(filepath.Join(dotRegolithPath, "session_lock"))
	if err != nil {
		return nil, burrito.WrapError(err, "Could not get the absolute path to the session_lock file.")
	}
	sessionLock, err := lockfile.New(sessionLockPath)
	if err != nil {
		return nil, burrito.WrapError(err, "Could not create session_lock file.")
	}
	err = sessionLock.TryLock()
	if err != nil {
		return nil, burrito.WrapError(
			err, "Could not lock the session_lock file. Is another instance of regolith running?")
	}
	unlockFunc := func() error {
		return sessionLock.Unlock()
	}
	return unlockFunc, nil
}

func splitPath(path string) []string {
	parts := make([]string, 0)
	for true {
		part := ""
		path, part = filepath.Split(path)
		if strings.HasSuffix(path, "/") || strings.HasSuffix(path, "\\") {
			path = path[0 : len(path)-1]
		}
		if path == "" && part != "" {
			parts = append([]string{part}, parts...)
			break
		}
		if part == "" || path == "" {
			break
		}
		parts = append([]string{part}, parts...)
	}
	return parts
}

func ResolvePath(path string) (string, error) {
	// Resolve the path
	parts := splitPath(path)
	for i, part := range parts {
		if strings.HasPrefix(part, "%") && strings.HasSuffix(part, "%") {
			envVar := part[1 : len(part)-1]
			envVarValue, exists := os.LookupEnv(envVar)
			if !exists {
				return "", burrito.WrapErrorf(
					os.ErrNotExist,
					"Environment variable %s does not exist.",
					envVar)
			}
			parts[i] = envVarValue
		}
	}
	return filepath.Clean(filepath.Join(parts...)), nil
}

type measure struct {
	// Name of the measure
	Name string
	// Location of the measure
	Location string
	// Start time of the measure
	StartTime time.Time
}

var lastMeasure *measure
var EnableTimings = false

func MeasureStart(name string) {
	if !EnableTimings {
		return
	}
	if lastMeasure != nil {
		MeasureEnd()
	}
	_, fn, line, _ := runtime.Caller(1)
	lastMeasure = &measure{
		Name:      name,
		StartTime: time.Now(),
		Location:  fmt.Sprintf("%s:%d", filepath.Base(fn), line),
	}
}

func MeasureEnd() {
	if !EnableTimings {
		return
	}
	if lastMeasure == nil {
		return
	}
	duration := time.Since(lastMeasure.StartTime)
	Logger.Infof("%s took %s (%s)", lastMeasure.Name, duration, lastMeasure.Location)
	lastMeasure = nil
}
