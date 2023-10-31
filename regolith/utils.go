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
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Bedrock-OSS/go-burrito/burrito"
	"github.com/nightlyone/lockfile"
)

// appDataProjectCachePath is a path to the project cache directory relative to the user's
// app data
const appDataProjectCachePath = "regolith/project-cache"

// appDataResolverCachePath is a path to the resolver cache directory relative to the user's
// app data
const appDataResolverCachePath = "regolith/resolver-cache"

// appDataResolverCachePath is a path to the resolver cache directory relative to the user's
// app data
const appDataFilterCachePath = "regolith/filter-cache"

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

// RunGitProcess runs a git command with specified arguments and working
// directory
func RunGitProcess(args []string, workingDir string) ([]string, error) {
	Logger.Debugf("Exec: git %s", strings.Join(args, " "))
	cmd := exec.Command("git", args...)
	cmd.Dir = workingDir
	out, _ := cmd.StdoutPipe()
	err, _ := cmd.StderrPipe()
	completeOutput := make([]string, 0)
	logFunc := func(template string, args ...interface{}) {
		completeOutput = append(completeOutput, fmt.Sprintf(template, args...))
	}
	go LogStd(out, logFunc, "git")
	go LogStd(err, logFunc, "git")

	return completeOutput, cmd.Run()
}

// LogStd logs the output of a sub-process
func LogStd(in io.ReadCloser, logFunc func(template string, args ...interface{}), outputLabel string) {
	scanner := bufio.NewScanner(in)
	for scanner.Scan() {
		logFunc("[%s] %s", outputLabel, scanner.Text())
	}
}

// getResolverCache gets the appDataResolverCachePath from the app data folder
func getResolverCache(resolver string) (string, error) {
	return getAppDataCachePath(appDataResolverCachePath, resolver)
}

// getFilterCache gets the appDataFilterCachePath from the app data folder
func getFilterCache(url string) (string, error) {
	path, err := getAppDataCachePath(appDataFilterCachePath, url)
	if err == nil {
		Logger.Debugf("Regolith filter cache for %s is in:\n\t%s", url, path)
	}
	return path, err
}

// getAppDataCachePath gets the dotRegolithPath from th app data folder
func getAppDataCachePath(basePath, cacheId string) (string, error) {
	// App data enabled - use user cache dir
	userCache, err := os.UserCacheDir()
	if err != nil {
		return "", burrito.WrappedError(osUserCacheDirError)
	}
	// Get the md5 of the project path
	hash := md5.New()
	hash.Write([]byte(cacheId))
	hashInBytes := hash.Sum(nil)
	projectPathHash := hex.EncodeToString(hashInBytes)
	// %userprofile%/AppData/Local/<base path>/<md5 of cache ID>
	cachePath := filepath.Join(
		userCache, basePath, projectPathHash)
	return cachePath, nil
}

// getAppDataDotRegolith gets the dotRegolithPath from the app data folder
func getAppDataDotRegolith(projectRoot string) (string, error) {
	// Make sure that projectsRoot is an absolute path
	absoluteProjectRoot, err := filepath.Abs(projectRoot)
	if err != nil {
		return "", burrito.WrapErrorf(err, filepathAbsError, projectRoot)
	}
	path, err := getAppDataCachePath(appDataProjectCachePath, absoluteProjectRoot)
	if err != nil {
		return "", burrito.PassError(err)
	}
	Logger.Infof("Regolith project cache is in:\n\t%s", path)
	return path, nil
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
func GetDotRegolith(projectRoot string) (string, error) {
	// App data disabled - use .regolith
	userConfig, err := getCombinedUserConfig()
	if err != nil {
		return "", burrito.WrapError(err, getUserConfigError)
	}
	if !*userConfig.UseProjectAppDataStorage {
		return ".regolith", nil
	}
	return getAppDataDotRegolith(projectRoot)
}

// acquireSessionLock creates a lock file in specified directory and
// returns a function that releases the lock.
// The path should point to the .regolith directory.
func acquireSessionLock(dotRegolithPath string) (func() error, error) {
	// Create dotRegolithPath if it doesn't exist
	err := os.MkdirAll(dotRegolithPath, 0755)
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

func MeasureStart(name string, args ...interface{}) {
	if !EnableTimings {
		return
	}
	if lastMeasure != nil {
		MeasureEnd()
	}
	_, fn, line, _ := runtime.Caller(1)
	lastMeasure = &measure{
		Name:      fmt.Sprintf(name, args...),
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

func stringInSlice(a string, list []string) bool {
	for _, b := range list {
		if b == a {
			return true
		}
	}
	return false
}

// FindByJSONPath finds a value in a JSON element by a simple path. Returns nil and an error if the path is not found or invalid.
func FindByJSONPath[T any](obj interface{}, path string) (T, error) {
	var empty T
	if obj == nil {
		return empty, burrito.WrappedErrorf("Object is empty")
	}
	// Split the path into parts
	parts, err := splitEscapedString(path)
	if err != nil {
		return empty, burrito.WrapErrorf(err, "Invalid path %s", path)
	}
	// Find the value
	value := obj
	currentPath := ""
	for _, part := range parts {
		if part == "" {
			continue
		}
		currentPath += part + "->"
		if m, ok := value.(map[string]interface{}); ok {
			value = m[part]
			if value == nil {
				return empty, burrito.WrappedErrorf(jsonPathMissingError, currentPath[:len(currentPath)-2])
			}
			continue
		}
		if a, ok := value.([]interface{}); ok {
			index, err := strconv.Atoi(part)
			if err != nil {
				return empty, burrito.WrapErrorf(err, "Invalid index %s at %s", part, currentPath[:len(currentPath)-2])
			}
			if index < 0 || index >= len(a) {
				return empty, burrito.WrappedErrorf("Index %i is out of bounds at %s", index, currentPath[:len(currentPath)-2])
			}
			value = a[index]
			if value == nil {
				return empty, burrito.WrappedErrorf(jsonPathMissingError, currentPath[:len(currentPath)-2])
			}
			continue
		}
		return empty, burrito.WrappedErrorf(jsonPathTypeError, currentPath[:len(currentPath)-2], "object or array")
	}
	if s, ok := value.(T); ok {
		return s, nil
	}
	return empty, burrito.WrappedErrorf(jsonPathTypeError, path, reflect.TypeOf(empty).String())
}

func splitEscapedString(s string) ([]string, error) {
	parts := make([]string, 0)
	var sb strings.Builder
	escape := false
	for _, c := range s {
		if escape {
			if c != '\\' && c != '/' {
				return nil, burrito.WrappedErrorf("Invalid escape sequence \\%c", c)
			}
			sb.WriteRune(c)
			escape = false
			continue
		}
		if c == '\\' {
			escape = true
			continue
		}
		if c == '/' {
			if sb.String() != "" {
				parts = append(parts, sb.String())
			}
			sb.Reset()
			continue
		}
		sb.WriteRune(c)
	}
	if escape {
		return nil, burrito.WrappedErrorf("Invalid escape sequence \\")
	}
	if sb.String() != "" {
		parts = append(parts, sb.String())
	}
	return parts, nil
}

func EscapePathPart(s string) string {
	var sb strings.Builder
	for _, c := range s {
		if c == '\\' || c == '/' {
			sb.WriteRune('\\')
		}
		sb.WriteRune(c)
	}
	return sb.String()
}
