package src

import (
	"go.uber.org/zap"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/otiai10/copy"
)

func Setup() error {
	start := time.Now()
	// Setup Directories
	Logger.Debug("Cleaning .tmp and build folders")
	err := os.RemoveAll(".tmp")
	if err != nil {
		return err
	}
	err = os.RemoveAll("build")
	if err != nil {
		return err
	}
	err = os.Mkdir("build", 777)
	if err != nil {
		return err
	}
	err = os.Mkdir(".tmp", 777)
	if err != nil {
		return err
	}

	// Copy the contents of the `regolith` folder to `.tmp`
	Logger.Debug("Copying project files to .tmp")
	err = copy.Copy("regolith", ".tmp", copy.Options{PreserveTimes: false, Sync: false})
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

	Setup()

	//now, we go through the filters!
	for _, filter := range profile.Filters {
		RunFilter(filter)
	}

	//copy contents of .tmp to build
	Logger.Info("Moving files to target directory")
	start := time.Now()
	os.Rename(".tmp", "build")
	Logger.Debug("Done in ", time.Since(start))
	//Done!
	Logger.Info(color.GreenString("Finished"))
}

// Runs the filter by selecting the correct filter and running it
func RunFilter(filter Filter) {
	Logger.Infof("Running filter '%s'", filter.Name)
	start := time.Now()
	absoluteWorkingDir, _ := filepath.Abs(".tmp")
	switch filter.RunWith {
	case "python":
		RunPythonFilter(filter, absoluteWorkingDir)
	default:
		Logger.Warnf("Filter type '%s' not supported", filter.RunWith)
	}
	Logger.Info("Executed in ", time.Since(start))
}

func RunPythonFilter(filter Filter, workingDir string) {
	absoluteLocation, _ := filepath.Abs(filter.Location)
	RunSubProcess(filter.RunWith, append([]string{"-u", absoluteLocation}, filter.Arguments...), workingDir)
}

func RunSubProcess(command string, args []string, workingDir string) {
	cmd := exec.Command(command, args...)
	cmd.Dir = workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()

	if err != nil {
		Logger.Fatal(zap.Error(err))
	}
}
