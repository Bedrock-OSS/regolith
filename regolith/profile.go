package regolith

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"github.com/otiai10/copy"
)

// SetupTmpFiles set up the workspace for the filters.
func SetupTmpFiles(config Config, profile Profile) error {
	start := time.Now()
	// Setup Directories
	Logger.Debug("Cleaning .regolith/tmp")
	err := os.RemoveAll(".regolith/tmp")
	if err != nil {
		return err
	}

	err = os.MkdirAll(".regolith/tmp", 0666)
	if err != nil {
		return err
	}

	// Copy the contents of the 'regolith' folder to '.regolith/tmp'
	Logger.Debug("Copying project files to .regolith/tmp")
	// Avoid repetetive code of preparing ResourceFolder, BehaviorFolder
	// and DataPath with a closure
	setup_tmp_directory := func(
		path, short_name, descriptive_name string,
	) error {
		if path != "" {
			stats, err := os.Stat(path)
			if err != nil {
				if os.IsNotExist(err) {
					Logger.Warnf(
						"%s %q does not exist", descriptive_name, path)
					err = os.MkdirAll(
						fmt.Sprintf(".regolith/tmp/%s", short_name), 0666)
					if err != nil {
						return err
					}
				}
			} else if stats.IsDir() {
				err = copy.Copy(
					path, fmt.Sprintf(".regolith/tmp/%s", short_name),
					copy.Options{PreserveTimes: false, Sync: false})
				if err != nil {
					return err
				}
			} else { // The folder paths leads to a file
				return fmt.Errorf(
					"%s path %q is not a directory",
					descriptive_name, path)
			}
		} else {
			err = os.MkdirAll(
				fmt.Sprintf(".regolith/tmp/%s", short_name), 0666)
			if err != nil {
				return err
			}
		}
		return nil
	}

	err = setup_tmp_directory(config.ResourceFolder, "RP", "resource folder")
	if err != nil {
		return err
	}
	err = setup_tmp_directory(config.BehaviorFolder, "BP", "behavior folder")
	if err != nil {
		return err
	}
	err = setup_tmp_directory(config.DataPath, "data", "data folder")
	if err != nil {
		return err
	}

	Logger.Debug("Setup done in ", time.Since(start))
	return nil
}

// RunProfile loads the profile from config.json and runs it. The profileName
// is the name of the profile which should be loaded from the configuration.
func RunProfile(profileName string) error {
	Logger.Info("Running profile: ", profileName)
	config := ConfigFromObject(LoadConfigAsMap())

	profile := config.Profiles[profileName]

	// Check whether every filter, uses a supported filter type
	for _, f := range profile.Filters {
		err := f.Check()
		if err != nil {
			return err
		}
	}

	// Prepare tmp files
	err := SetupTmpFiles(*config, profile)
	if err != nil {
		return wrapError("Unable to setup profile", err)
	}

	// Run the filters!
	for filter := range profile.Filters {
		filter := profile.Filters[filter]
		path, _ := filepath.Abs(".")
		err := filter.Run(path)
		if err != nil {
			return wrapError(fmt.Sprintf("%s failed", filter.GetFriendlyName()), err)
		}
	}

	// Export files
	Logger.Info("Moving files to target directory")
	start := time.Now()
	err = ExportProject(profile, config.Name, config.DataPath)
	if err != nil {
		return wrapError("Exporting project failed", err)
	}
	Logger.Debug("Done in ", time.Since(start))
	// Clear the tmp/data path
	err = os.RemoveAll(".regolith/tmp/data")
	if err != nil {
		return wrapError("Unable to clean .regolith/tmp/data directory", err)
	}
	return nil
}

// SubfilterCollection returns a collection of filters from a
// "filter.json" file of a remote filter.
func (f *RemoteFilter) SubfilterCollection() (*FilterCollection, error) {
	path := filepath.Join(f.GetDownloadPath(), "filter.json")
	result := &FilterCollection{Filters: []FilterRunner{}}
	file, err := ioutil.ReadFile(path)

	if err != nil {
		return nil, wrapError(
			fmt.Sprintf("Couldn't read %q", path),
			err,
		)
	}

	var filterCollection map[string]interface{}
	err = json.Unmarshal(file, &filterCollection)
	if err != nil {
		return nil, wrapError(
			fmt.Sprintf(
				"couldn't load %s! Does the file contain correct json?", path),
			err)
	}
	// Filters
	filters, ok := filterCollection["filters"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("could not parse filters of %q", path)
	}
	for i, filter := range filters {
		filter, ok := filter.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("could not parse filter %v of %q", i, path)
		}
		// Using the same JSON data to create both the filter
		// definiton (installer) and the filter (runner)
		filterInstaller := FilterInstallerFromObject(
			fmt.Sprintf("%v:subfilter%v", f.Id, i), filter)
		filterRunner := filterInstaller.CreateFilterRunner(filter)
		if _, ok := filterRunner.(*RemoteFilter); ok {
			// TODO - we could possibly implement recursive filters here
			return nil, fmt.Errorf(
				"remote filters are not allowed in subfilters. Remote filter"+
					" %q subfilter %v", f.Id, i)
		}
		filterRunner.CopyArguments(f)
		result.Filters = append(result.Filters, filterRunner)
	}
	return result, nil
}

// FilterCollection is a list of filters
type FilterCollection struct {
	Filters []FilterRunner `json:"filters,omitempty"`
}

type Profile struct {
	FilterCollection
	ExportTarget ExportTarget `json:"export,omitempty"`
}

func ProfileFromObject(
	profileName string, obj map[string]interface{},
	filterDefinitions map[string]FilterInstaller,
) Profile {
	result := Profile{}
	// Filters
	filters, ok := obj["filters"].([]interface{})
	if !ok {
		Logger.Fatalf("Could not parse filters of profile %q", profileName)
	}
	for i, filter := range filters {
		filter, ok := filter.(map[string]interface{})
		if !ok {
			Logger.Fatalf(
				"Could not parse filter %s of profile %q", i, profileName,
			)
		}
		result.Filters = append(
			result.Filters, FilterRunnerFromObjectAndDefinitions(filter, filterDefinitions))
	}
	// ExportTarget
	export, ok := obj["export"].(map[string]interface{})
	if !ok {
		Logger.Fatalf("Could not parse export property of profile %q", profileName)
	}
	result.ExportTarget = ExportTargetFromObject(export)
	return result
}
