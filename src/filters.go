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

// SetupTmpFiles copies the source RP and BP to .regolith/tmp path to create
// the workspace for the filters.
func SetupTmpFiles() error {
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

// RunProfile loads the profile from config.json and runs it. The profileName
// is the name of the profile which should be loaded from the configuration.
func RunProfile(profileName string) {
	Logger.Info("Running profile: ", profileName)
	project := LoadConfig()
	profile := project.Profiles[profileName]

	if profile.Unsafe {
		Logger.Info("Warning! Profile flagged as unsafe. Exercise caution!")
	}

	// Check whether every filter, uses a supported filter type
	checked := make(map[string]struct{})
	exists := struct{}{}
	for _, filter := range profile.Filters {
		if filter.RunWith != "" {
			if _, ok := checked[filter.RunWith]; !ok {
				if f, ok := FilterTypes[filter.RunWith]; ok {
					checked[filter.RunWith] = exists
					f.check()
				} else {
					Logger.Warnf("Filter type '%s' not supported", filter.RunWith)
				}
			}
		}
	}

	// Prepare tmp files
	err := SetupTmpFiles()
	if err != nil {
		Logger.Fatal("Unable to setup profile")
	}

	// Run the filters!
	for _, filter := range profile.Filters {
		path, _ := filepath.Abs(".")
		filter.RunFilter(path)
	}

	// Copy contents of .regolith/tmp to build
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
	Logger.Info(color.GreenString("Finished"))
}

// GetExportPaths returns file paths for exporting behavior pack and
// resource pack based on exportTarget (a structure with data related to
// export settings) and the name of the project.
func GetExportPaths(exportTarget ExportTarget, name string) (bpPath string, rpPath string) {
	if exportTarget.Target == "development" {
		comMojang := FindMojangDir()
		// TODO - I don't like the _rp and _bp sufixes. Can we get rid of that?
		// I for example always name my packs "0".
		bpPath = comMojang + "/development_behavior_packs/" + name + "_bp"
		rpPath = comMojang + "/development_resource_packs/" + name + "_rp"
		return
	} else if exportTarget.Target == "exact" {
		bpPath = exportTarget.BpPath
		rpPath = exportTarget.RpPath
		return
	}

	// Throw fatal error that export target isn't valid
	Logger.Fatalf("Export '%s' target not valid", exportTarget.Target)
	// Unreachable code
	return
}

// ExportProject copies files from the build paths (build/BP and build/RP) into
// the project's export target and its name. The paths are generated with
// GetExportPaths.
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

// RunFilter determinates whether the filter is remote, standard (from standard
// library) or local and executes it using the proper function. The
// absoluteLocation is an absolute path to the root folder of the filter.
// In case of local filters it's a root path of the project.
func (filter *Filter) RunFilter(absoluteLocation string) {
	Logger.Infof("Running filter '%s'", filter.Name)
	start := time.Now()

	if filter.Url != "" {
		RunRemoteFilter(filter.Url, *filter)
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

// RunStandardFilter runs a filter from standard Bedrock-OSS library. The
// function doesn't test if the filter passed on input is standard.
func RunStandardFilter(filter Filter) {
	Logger.Infof("RunStandardFilter '%s'", filter.Filter)
	RunRemoteFilter(FilterNameToUrl(filter.Filter), filter)
}

// LoadFiltersFromPath returns a Profile with list of filters loaded from
// filters.json from input file path. The path should point at a directory
// with filters.json file in it, not at the file itself.
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

// RunRemoteFilter runs loads and runs the content of filter.json from in
// regolith cache. The url is the URL of the filter from which the filter
// was downloaded (used to specify its path in the cache). The parentFilter is
// is a filter that caused the downloading. Some of the properties of
// parentFilter are propagated to its children.
func RunRemoteFilter(url string, parentFilter Filter) {
	settings := parentFilter.Settings
	// TODO - I think this also should be used somehow:
	// arguments := parentFilter.Arguments
	Logger.Infof("RunRemoteFilter '%s'", url)
	if !IsRemoteFilterCached(url) {
		Logger.Error("Filter is not downloaded! Please run 'regolith install'.")
	}

	path := UrlToPath(url)
	absolutePath, _ := filepath.Abs(path)
	for _, filter := range LoadFiltersFromPath(path).Filters {
		// Overwrite the venvSlot with the parent value
		filter.VenvSlot = parentFilter.VenvSlot
		// Join settings from local config to remote definition
		for k, v := range settings {
			filter.Settings[k] = v
		}
		filter.RunFilter(absolutePath)
	}
}

// GetAbsoluteWorkingDirectory returns an absolute path to .regolith/tmp
func GetAbsoluteWorkingDirectory() string {
	absoluteWorkingDir, _ := filepath.Abs(".regolith/tmp")
	return absoluteWorkingDir
}

// RunSubProcess runs a sub-process with specified arguments and working
// directory
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
