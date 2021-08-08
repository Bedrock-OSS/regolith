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
		RunFilter(filter)
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
func RunFilter(filter Filter) {
	Logger.Infof("Running filter '%s'", filter.Name)
	start := time.Now()

	// Run via online filter
	if filter.Url != "" {
		RunRemoteFilter(filter.Url, filter.Arguments)
	}

	// Run via standard filter
	if filter.Filter != "" {
		RunStandardFilter(filter, filter.Arguments)
	}

	// Run based on run-target
	switch filter.RunWith {
	case "python":
		RunPythonFilter(filter)
	default:
		Logger.Warnf("Filter type '%s' not supported", filter.RunWith)
	}
	Logger.Info("Executed in ", time.Since(start))
}

func RunStandardFilter(filter Filter, arguments []string) {
	url := "https://github.com/Bedrock-OSS/regolith-filters/" + filter.Filter
	RunRemoteFilter(url, arguments)
}

func LoadFiltersFromPath(path string) Profile {
	file, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(color.RedString("Couldn't find %s! Consider running 'regolith init'", ManifestName))
	}
	var result Profile
	err = json.Unmarshal(file, &result)
	if err != nil {
		log.Fatal(color.RedString("Couldn't load %s: ", path), err)
	}
	return result
}

func RunRemoteFilter(url string, arguments []string) {
	if !IsRemoteFilterCached(url) {
		Logger.Error("Filter is not downloaded! Please run 'regolith install'.")
	}

	for _, filter := range LoadFiltersFromPath(UrlToPath(url)).Filters {
		RunFilter(filter)
	}
}

func RunPythonFilter(filter Filter) {
	absoluteLocation, _ := filepath.Abs(filter.Location)
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
