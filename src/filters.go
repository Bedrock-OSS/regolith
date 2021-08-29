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

type filterDefinition struct {
	filter  func(filter Filter, settings map[string]interface{}, absoluteLocation string)
	install func(filter Filter, path string)
	check   func()
}

var FilterTypes = map[string]filterDefinition{}

func RegisterFilters() {
	RegisterPythonFilter(FilterTypes)
	RegisterNodeJSFilter(FilterTypes)
	RegisterShellFilter(FilterTypes)
}

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

	err = copy.Copy(LoadConfig().Packs.BehaviorFolder, ".regolith/tmp/BP", copy.Options{PreserveTimes: false, Sync: false})
	if err != nil {
		return err
	}

	err = copy.Copy(LoadConfig().Packs.ResourceFolder, ".regolith/tmp/RP", copy.Options{PreserveTimes: false, Sync: false})
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

	// Check whether filter, that the user wants to run meet the requirements
	checked := make(map[string]struct{})
	exists := struct{}{}
	for _, filter := range profile.Filters {
		if filter.RunWith != "" {
			if _, ok := checked[filter.RunWith]; ok {
				if f, ok := FilterTypes[filter.RunWith]; ok {
					checked[filter.RunWith] = exists
					f.check()
				} else {
					Logger.Warnf("Filter type '%s' not supported", filter.RunWith)
				}
			}
		}
	}

	err := Setup()
	if err != nil {
		Logger.Fatal("Unable to setup profile")
	}

	//now, we go through the filters!
	for _, filter := range profile.Filters {
		path, _ := filepath.Abs(".")
		filter.RunFilter(path)
	}

	//copy contents of .regolith/tmp to build
	Logger.Info("Moving files to target directory")
	start := time.Now()
	err = os.RemoveAll("build")
	if err != nil {
		Logger.Fatal("Unable to clean build directory")
	}
	err = os.Rename(".regolith/tmp", "build")
	if err != nil {
		Logger.Fatal("Unable to move output to build directory")
	}
	Logger.Debug("Done in ", time.Since(start))

	// copy the build to the target directory
	Logger.Info("Copying build to target directory")
	start = time.Now()
	ExportProject(profile, project.Name)
	Logger.Debug("Done in ", time.Since(start))

	//Done!
	Logger.Info(color.GreenString("Finished"))
}

func GetExportPaths(exportTarget ExportTarget, name string) (string, string) {
	if exportTarget.Target == "development" {
		comMojang := FindMojangDir()
		return comMojang + "/development_behavior_packs/" + name + "_bp", comMojang + "/development_resource_packs/" + name + "_rp"
	}

	// Throw fatal error that export target isn't valid
	Logger.Fatalf("Export '%s' target not valid", exportTarget.Target)
	return "", ""
}

func ExportProject(profile Profile, name string) {
	var err error
	exportTarget := profile.ExportTarget
	bpPath, rpPath := GetExportPaths(exportTarget, name)

	// Allow clearing output locations, before writing
	// TODO Uncomment this. Is it safe? Can we send to recycle bin?
	// if exportTarget.Clean {
	// 	os.RemoveAll(bpPath")
	// 	os.MkdirAll(bpPath", 0777)
	// 	os.RemoveAll(rpPath")
	// 	os.MkdirAll(rpPath", 0777)
	// }

	Logger.Info("Exporting project to ", bpPath)
	Logger.Info("Exporting project to ", rpPath)

	err = copy.Copy("build/BP/", bpPath, copy.Options{PreserveTimes: false, Sync: false})
	if err != nil {
		Logger.Fatal(color.RedString("Couldn't copy BP files to %s", bpPath), err)
	}

	err = copy.Copy("build/RP/", rpPath, copy.Options{PreserveTimes: false, Sync: false})
	if err != nil {
		Logger.Fatal(color.RedString("Couldn't copy RP files to %s", rpPath), err)
	}
}

// RunFilter Runs the filter by selecting the correct filter type and running it
func (filter *Filter) RunFilter(absoluteLocation string) {
	Logger.Infof("Running filter '%s'", filter.Name)
	start := time.Now()

	if filter.Url != "" {
		RunRemoteFilter(filter.Url, filter.Settings, filter.Arguments)
	} else if filter.Filter != "" {
		RunStandardFilter(*filter)
	} else {
		if f, ok := FilterTypes[filter.RunWith]; ok {
			f.filter(*filter, filter.Settings, absoluteLocation)
		} else {
			Logger.Warnf("Filter type '%s' not supported", filter.RunWith)
		}
		Logger.Info("Executed in ", time.Since(start))
	}
}

func RunStandardFilter(filter Filter) {
	Logger.Infof("RunStandardFilter '%s'", filter.Filter)
	RunRemoteFilter(FilterNameToUrl(filter.Filter), filter.Settings, filter.Arguments)
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

func RunRemoteFilter(url string, settings map[string]interface{}, arguments []string) {
	Logger.Infof("RunRemoteFilter '%s'", url)
	if !IsRemoteFilterCached(url) {
		Logger.Error("Filter is not downloaded! Please run 'regolith install'.")
	}

	path := UrlToPath(url)
	absolutePath, _ := filepath.Abs(path)
	for _, filter := range LoadFiltersFromPath(path).Filters {
		// Join settings from local config to remote definition
		for k, v := range settings {
			filter.Settings[k] = v
		}
		filter.RunFilter(absolutePath)
	}
}

func GetAbsoluteWorkingDirectory() string {
	absoluteWorkingDir, _ := filepath.Abs(".regolith/tmp")
	return absoluteWorkingDir
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
