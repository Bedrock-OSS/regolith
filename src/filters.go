package src

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/otiai10/copy"
)

type filterDefinition struct {
	filter  func(filter Filter, settings map[string]interface{}, absoluteLocation string) error
	install func(filter Filter, path string) error
	check   func() error
}

var FilterTypes = map[string]filterDefinition{}

func RegisterFilters() {
	RegisterPythonFilter(FilterTypes)
	RegisterNodeJSFilter(FilterTypes)
	RegisterShellFilter(FilterTypes)
}

// the workspace for the filters.
func SetupTmpFiles(config Config, profile Profile) error {
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

	err = copy.Copy(config.Packs.BehaviorFolder, ".regolith/tmp/BP", copy.Options{PreserveTimes: false, Sync: false})
	if err != nil {
		return err
	}

	err = copy.Copy(config.Packs.ResourceFolder, ".regolith/tmp/RP", copy.Options{PreserveTimes: false, Sync: false})
	if err != nil {
		return err
	}
	if profile.DataPath != "" { // datapath copied only if specified
		err = copy.Copy(profile.DataPath, ".regolith/tmp/data", copy.Options{PreserveTimes: false, Sync: false})
		if err != nil {
			return err
		}
	} else { // create empty data path
		err = os.MkdirAll(".regolith/data", 0777)
		if err != nil {
			return err
		}
	}

	Logger.Debug("Setup done in ", time.Since(start))
	return nil
}

// RunProfile loads the profile from config.json and runs it. The profileName
// is the name of the profile which should be loaded from the configuration.
func RunProfile(profileName string) error {
	Logger.Info("Running profile: ", profileName)
	project, err := LoadConfig()
	if err != nil {
		return wrapError("Failed to load project config", err)
	}
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
					err := f.check()
					if err != nil {
						return err
					}
				} else {
					Logger.Warnf("Filter type '%s' not supported", filter.RunWith)
				}
			}
		}
	}

	// Prepare tmp files
	err = SetupTmpFiles(*project, profile)
	if err != nil {
		return wrapError("Unable to setup profile", err)
	}

	// Run the filters!
	for _, filter := range profile.Filters {
		path, _ := filepath.Abs(".")
		err := filter.RunFilter(path)
		if err != nil {
			return wrapError(fmt.Sprintf("Running filter '%s' failed", filter.Name), err)
		}
	}

	// Copy contents of .regolith/tmp to build
	Logger.Info("Moving files to target directory")
	start := time.Now()
	err = os.RemoveAll("build")
	if err != nil {
		return wrapError("Unable to clean build directory", err)
	}
	err = os.RemoveAll(".regolith/tmp/data")
	if err != nil {
		return wrapError("Unable to clean .regolith/tmp/data directory", err)
	}
	err = os.Rename(".regolith/tmp", "build")
	if err != nil {
		return wrapError("Unable to move output to build directory", err)
	}
	Logger.Debug("Done in ", time.Since(start))

	// copy the build to the target directory
	if profile.ExportTarget.Target != "none" {
		Logger.Info("Copying build to target directory")
		start = time.Now()
		err = ExportProject(profile, project.Name)
		if err != nil {
			return wrapError("Exporting project failed", err)
		}
		Logger.Debug("Done in ", time.Since(start))
	}
	return nil
}

// GetExportPaths returns file paths for exporting behavior pack and
// resource pack based on exportTarget (a structure with data related to
// export settings) and the name of the project.
func GetExportPaths(exportTarget ExportTarget, name string) (bpPath string, rpPath string, err error) {
	if exportTarget.Target == "development" {
		comMojang, err := FindMojangDir()
		if err != nil {
			return "", "", wrapError("Failed to find com.mojang directory", err)
		}
		// TODO - I don't like the _rp and _bp sufixes. Can we get rid of that?
		// I for example always name my packs "0".
		bpPath = comMojang + "/development_behavior_packs/" + name + "_bp"
		rpPath = comMojang + "/development_resource_packs/" + name + "_rp"
	} else if exportTarget.Target == "exact" {
		bpPath = exportTarget.BpPath
		rpPath = exportTarget.RpPath
	} else if exportTarget.Target == "world" {
		if exportTarget.WorldPath != "" {
			if exportTarget.WorldName != "" {
				return "", "", errors.New("Using both \"worldName\" and \"worldPath\" is not allowed.")
			}
			bpPath = filepath.Join(exportTarget.WorldPath, "behavior_packs", name+"_bp")
			rpPath = filepath.Join(exportTarget.WorldPath, "resource_packs", name+"_rp")
		} else if exportTarget.WorldName != "" {
			dir, err := FindMojangDir()
			if err != nil {
				return "", "", wrapError("Failed to find com.mojang directory", err)
			}
			worlds, err := ListWorlds(dir)
			if err != nil {
				return "", "", wrapError("Failed to list worlds", err)
			}
			for _, world := range worlds {
				if world.Name == exportTarget.WorldName {
					bpPath = filepath.Join(world.Path, "behavior_packs", name+"_bp")
					rpPath = filepath.Join(world.Path, "resource_packs", name+"_rp")
				}
			}
		} else {
			err = errors.New("The \"world\" export target requires either a \"worldName\" or \"worldPath\" property")
		}
	} else {
		err = errors.New(fmt.Sprintf("Export '%s' target not valid", exportTarget.Target))
	}
	return
}

// ExportProject copies files from the build paths (build/BP and build/RP) into
// the project's export target and its name. The paths are generated with
// GetExportPaths.
func ExportProject(profile Profile, name string) error {
	exportTarget := profile.ExportTarget
	bpPath, rpPath, err := GetExportPaths(exportTarget, name)
	if err != nil {
		return err
	}

	// Loading edited_files.json or creating empty object
	editedFiles := LoadEditedFiles()
	err = editedFiles.CheckDeletionSafety(rpPath, bpPath)
	if err != nil {
		return errors.New(
			"Exporting project was aborted because it could remove some " +
				"files you want to keep: " + err.Error() + ". If you are trying to run " +
				"Regolith for the first time on this project make sure that the " +
				"export paths are empty. Otherwise, you can check \"" +
				EditedFilesPath + "\" file to see if it contains the full list of " +
				"the files that can be reomved.")
	}
	// Clearing output locations
	// Spooky, I hope file protection works, and it won't do any damage
	err = os.RemoveAll(bpPath)
	if err != nil {
		return wrapError("Failed to clear behavior pack build output", err)
	}
	err = os.MkdirAll(bpPath, 0777)
	if err != nil {
		return wrapError("Failed to create required directories for behavior pack build output", err)
	}
	err = os.RemoveAll(rpPath)
	if err != nil {
		return wrapError("Failed to clear resource pack build output", err)
	}
	err = os.MkdirAll(rpPath, 0777)
	if err != nil {
		return wrapError("Failed to create required directories for resource pack build output", err)
	}

	Logger.Info("Exporting project to ", bpPath)
	Logger.Info("Exporting project to ", rpPath)

	err = copy.Copy("build/BP/", bpPath, copy.Options{PreserveTimes: false, Sync: false})
	if err != nil {
		return wrapError(fmt.Sprintf("Couldn't copy BP files to %s", bpPath), err)
	}

	err = copy.Copy("build/RP/", rpPath, copy.Options{PreserveTimes: false, Sync: false})
	if err != nil {
		return wrapError(fmt.Sprintf("Couldn't copy RP files to %s", rpPath), err)
	}

	// Create new edited_files.json
	editedFiles, err = NewEditedFiles(rpPath, bpPath)
	if err != nil {
		return err
	}
	err = editedFiles.Dump()
	return err
}

// RunFilter determinates whether the filter is remote, standard (from standard
// library) or local and executes it using the proper function. The
// absoluteLocation is an absolute path to the root folder of the filter.
// In case of local filters it's a root path of the project.
func (filter *Filter) RunFilter(absoluteLocation string) error {
	Logger.Infof("Running filter '%s'", filter.Name)
	start := time.Now()

	if filter.Url != "" {
		err := RunRemoteFilter(filter.Url, *filter)
		if err != nil {
			return err
		}
	} else if filter.Filter != "" {
		err := RunStandardFilter(*filter)
		if err != nil {
			return err
		}
	} else {
		if f, ok := FilterTypes[filter.RunWith]; ok {
			err := f.filter(*filter, filter.Settings, absoluteLocation)
			if err != nil {
				return err
			}
		} else {
			Logger.Warnf("Filter type '%s' not supported", filter.RunWith)
		}
		Logger.Info("Executed in ", time.Since(start))
	}
	return nil
}

// RunStandardFilter runs a filter from standard Bedrock-OSS library. The
// function doesn't test if the filter passed on input is standard.
func RunStandardFilter(filter Filter) error {
	Logger.Infof("RunStandardFilter '%s'", filter.Filter)
	return RunRemoteFilter(FilterNameToUrl(filter.Filter), filter)
}

// LoadFiltersFromPath returns a Profile with list of filters loaded from
// filters.json from input file path. The path should point at a directory
// with filters.json file in it, not at the file itself.
func LoadFiltersFromPath(path string) (*Profile, error) {
	path = path + "/filter.json"
	file, err := ioutil.ReadFile(path)

	if err != nil {
		return nil, wrapError(fmt.Sprintf("Couldn't find %s! Consider running 'regolith install'", path), err)
	}

	var result *Profile
	err = json.Unmarshal(file, &result)
	if err != nil {
		return nil, wrapError(fmt.Sprintf("Couldn't load %s: ", path), err)
	}
	// Replace nil filter settings with empty map
	for fk := range result.Filters {
		if result.Filters[fk].Settings == nil {
			result.Filters[fk].Settings = make(map[string]interface{})
		}
	}
	return result, nil
}

// RunRemoteFilter runs loads and runs the content of filter.json from in
// regolith cache. The url is the URL of the filter from which the filter
// was downloaded (used to specify its path in the cache). The parentFilter is
// is a filter that caused the downloading. Some of the properties of
// parentFilter are propagated to its children.
func RunRemoteFilter(url string, parentFilter Filter) error {
	settings := parentFilter.Settings
	// TODO - I think this also should be used somehow:
	// arguments := parentFilter.Arguments
	Logger.Infof("RunRemoteFilter '%s'", url)
	if !IsRemoteFilterCached(url) {
		return errors.New("Filter is not downloaded! Please run 'regolith install'.")
	}

	path := UrlToPath(url)
	absolutePath, _ := filepath.Abs(path)
	profile, err := LoadFiltersFromPath(path)
	if err != nil {
		return err
	}
	for _, filter := range profile.Filters {
		// Overwrite the venvSlot with the parent value
		filter.VenvSlot = parentFilter.VenvSlot
		// Join settings from local config to remote definition
		for k, v := range settings {
			filter.Settings[k] = v
		}
		err := filter.RunFilter(absolutePath)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetAbsoluteWorkingDirectory returns an absolute path to .regolith/tmp
func GetAbsoluteWorkingDirectory() string {
	absoluteWorkingDir, _ := filepath.Abs(".regolith/tmp")
	return absoluteWorkingDir
}

// RunSubProcess runs a sub-process with specified arguments and working
// directory
func RunSubProcess(command string, args []string, workingDir string) error {
	cmd := exec.Command(command, args...)
	cmd.Dir = workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
