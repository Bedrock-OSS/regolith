package src

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"go.uber.org/zap"

	"github.com/fatih/color"
	"github.com/otiai10/copy"
)

func Setup() error {
	start := time.Now()
	// Setup Directories
	Logger.Debug("Cleaning .regolith/tmp")
	err := os.RemoveAll(".regolith/tmp")
	if err != nil {
		return err
	}

	err = os.MkdirAll(".regolith/tmp", 0777)
	if err != nil {
		return err
	}

	// Copy the contents of the `regolith` folder to `.regolith/tmp`
	Logger.Debug("Copying project files to .regolith/tmp")
	err = copy.Copy("regolith", ".regolith/tmp", copy.Options{PreserveTimes: false, Sync: false})
	if err != nil {
		return err
	}
	Logger.Debug("Setup done in ", time.Since(start))
	return nil
}

func RunProfile(profileName string) {
	Logger.Info("Running profile: ", profileName)
	project := LoadConfig()
	// The first arg specifies the profile of the manifest used
	profile := project.Profiles[profileName]
	if profile.Unsafe {
		Logger.Info("Warning! Profile flagged as unsafe. Exercise caution!")
	}

	err := Setup()
	if err != nil {
		Logger.Fatal("Unable to setup profile")
	}

	//now, we go through the filters!
	for _, filter := range profile.Filters {
		path, _ := filepath.Abs(".")
		RunFilter(filter, path)
	}

	//copy contents of .regolith/tmp to build
	Logger.Info("Moving files to target directory")
	start := time.Now()
	os.RemoveAll("build")
	os.Rename(".regolith/tmp", "build")
	Logger.Debug("Done in ", time.Since(start))
	//Done!
	Logger.Info(color.GreenString("Finished"))
}

// Runs the filter by selecting the correct filter type and running it
func RunFilter(filter Filter, absoluteLocation string) {
	Logger.Infof("Running filter '%s'", filter.Name)
	start := time.Now()

	if filter.Url != "" {
		RunRemoteFilter(filter.Url, filter.Arguments)
	} else if filter.Filter != "" {
		RunStandardFilter(filter, filter.Arguments)
	} else {
		switch filter.RunWith {
		case "python":
			RunPythonFilter(filter, absoluteLocation+filter.Location)
		default:
			Logger.Warnf("Filter type '%s' not supported", filter.RunWith)
		}
		Logger.Info("Executed in ", time.Since(start))
	}
}

func RunStandardFilter(filter Filter, arguments []string) {
	Logger.Infof("RunStandardFilter '%s'", filter.Filter)
	RunRemoteFilter(FilterNameToUrl(filter.Filter), arguments)
}

func LoadFiltersFromPath(path string) Profile {
	path = path + "/filter.json"
	file, err := ioutil.ReadFile(path)

	if err != nil {
		log.Fatal(color.RedString("Couldn't find %s! Consider running 'regolith install'", path), err)
	}

	var result Profile
	err = json.Unmarshal(file, &result)
	if err != nil {
		log.Fatal(color.RedString("Couldn't load %s: ", path), err)
	}
	return result
}

func RunRemoteFilter(url string, arguments []string) {
	Logger.Infof("RunRemoteFilter '%s'", url)
	if !IsRemoteFilterCached(url) {
		Logger.Error("Filter is not downloaded! Please run 'regolith install'.")
	}

	path := UrlToPath(url)
	absolutePath, _ := filepath.Abs(path)
	for _, filter := range LoadFiltersFromPath(path).Filters {
		RunFilter(filter, absolutePath)
	}
}

func RunPythonFilter(filter Filter, absoluteLocation string) {
	RunSubProcess(filter.RunWith, append([]string{"-u", absoluteLocation}, filter.Arguments...))
}

func GetAbsoluteWorkingDirectory() string {
	absoluteWorkingDir, _ := filepath.Abs(".regolith/tmp")
	return absoluteWorkingDir
}

func RunSubProcess(command string, args []string) {
	cmd := exec.Command(command, args...)
	cmd.Dir = GetAbsoluteWorkingDirectory()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	if err != nil {
		Logger.Fatal(zap.Error(err))
	}
}
