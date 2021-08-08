package src

import (
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
	Logger.Debug("Cleaning .regolith/tmp and build folders")
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

func GatherDependencies() []string {
	project := LoadConfig()
	var dependencies []string
	for _, profile := range project.Profiles {
		for _, filter := range profile.Filters {
			dependencies = append(dependencies, GatherDependency(filter))
		}
	}
	return dependencies
}

func GatherDependency(filter Filter) string {
	Logger.Info("TODO")
	return "TODO"
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

	//copy contents of .regolith/tmp to build
	Logger.Info("Moving files to target directory")
	start := time.Now()
	os.RemoveAll("build")
	os.Rename(".regolith/tmp", "build")
	Logger.Debug("Done in ", time.Since(start))
	//Done!
	Logger.Info(color.GreenString("Finished"))
}

// Runs the filter by selecting the correct filter and running it
func RunFilter(filter Filter) {
	Logger.Infof("Running filter '%s'", filter.Name)
	start := time.Now()
	absoluteWorkingDir, _ := filepath.Abs(".regolith/tmp")
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
