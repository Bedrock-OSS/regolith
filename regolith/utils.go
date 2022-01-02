package regolith

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fatih/color"
	"github.com/google/go-github/v39/github"
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
		return errors.New(fmt.Sprintf("%s\n[%s]: %s", text, color.RedString("+"), err.Error()))
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

func ParseSemanticVersion(semver string) (major int, minor int, patch int) {
	split := strings.Split(semver, ".")
	length := len(split)
	if length > 0 {
		major, _ = strconv.Atoi(split[0])
	}
	if length > 1 {
		minor, _ = strconv.Atoi(split[1])
	}
	if length > 2 {
		patch, _ = strconv.Atoi(split[2])
	}
	return major, minor, patch
}

// Returns 1 if first version is newer, -1 if older, 0 if the same
func CompareSemanticVersion(ver1 string, ver2 string) int {
	major1, minor1, patch1 := ParseSemanticVersion(ver1)
	major2, minor2, patch2 := ParseSemanticVersion(ver2)
	if major1 > major2 {
		return 1
	} else if major1 < major2 {
		return -1
	} else {
		if minor1 > minor2 {
			return 1
		} else if minor1 < minor2 {
			return -1
		} else {
			if patch1 > patch2 {
				return 1
			} else if patch1 < patch2 {
				return -1
			} else {
				return 0
			}
		}
	}
}

type UpdateStatus struct {
	ShouldUpdate bool
	Url          *string
	Err          *error
}

func CheckUpdate(version string, status chan UpdateStatus) {
	client := github.NewClient(nil)
	// Ignore the error, since it's not critical to regolith
	release, _, err := client.Repositories.GetLatestRelease(context.Background(), "Bedrock-OSS", "regolith")
	if err != nil {
		status <- UpdateStatus{Err: &err}
		return
	}
	status <- UpdateStatus{
		ShouldUpdate: CompareSemanticVersion(*release.TagName, version) == 1,
		Url:          release.HTMLURL,
	}
}
