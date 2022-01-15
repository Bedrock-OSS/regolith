package regolith

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
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
	err = setup_tmp_directory(profile.DataPath, "data", "data folder")
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
	config := LoadConfig()

	profile := config.Profiles[profileName]

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
	err = ExportProject(profile, config.Name)
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

// FilterCollectionFromFilterJson returns a collection of filters from a
// "filter.json" file of a remote filter.
func FilterCollectionFromFilterJson(path string) (*FilterCollection, error) {
	file, err := ioutil.ReadFile(path)

	if err != nil {
		return nil, wrapError(
			fmt.Sprintf("Couldn't read %q", path),
			err,
		)
	}

	var result *FilterCollection
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

// FilterCollection is a list of filters
type FilterCollection struct {
	Filters []Filter `json:"filters,omitempty"`
}

type Profile struct {
	FilterCollection
	ExportTarget ExportTarget `json:"export,omitempty"`
	DataPath     string       `json:"dataPath,omitempty"`
}

// LoadFilterJsonProfile loads a profile from path to filter.json file of
// a remote filter and propagates the properties of the parent filter (the
// filter in config.json or other remote filter that caused creation of this
// profile).and the parent profile to the returned profile.
func LoadFilterJsonProfile(
	filterJsonPath string, parentFilter Filter, parentProfile Profile,
) (*Profile, error) {
	// Open file
	file, err := ioutil.ReadFile(filterJsonPath)
	if err != nil {
		return nil, wrapError(fmt.Sprintf(
			"Couldn't find %s", filterJsonPath), err)
	}
	// Load data into Profile struct
	var remoteProfile Profile
	err = json.Unmarshal(file, &remoteProfile)
	if err != nil {
		return nil, wrapError(fmt.Sprintf(
			"Couldn't load %s: ", filterJsonPath), err)
	}
	// Propagate venvSlot property
	for subfilter := range remoteProfile.Filters {
		remoteProfile.Filters[subfilter].VenvSlot = parentFilter.VenvSlot
	}
	remoteProfile.DataPath = parentProfile.DataPath
	remoteProfile.ExportTarget = parentProfile.ExportTarget
	return &remoteProfile, nil
}

// Install installs all of the filters in the profile including the nested ones
func (p *Profile) Install(isForced bool) error {
	return p.installFilters(isForced, p.Filters)
}

// installFilters provides a recursive function to install all filters in the
// profile. This function is not exposed outside of the regolith package. Use
// Install() instead.
func (p *Profile) installFilters(isForced bool, filters []Filter) error {
	for _, filter := range filters {
		err := p.installFilter(isForced, filter)
		if err != nil {
			return err
		}
	}
	return nil
}

// installFilter installs a single filter.
// - Downloads the filter if it is remote
// - Installs dependencies
// - Copies the filter's data to the data folder
// - Handles additional filters within the 'filters.json' file
func (p *Profile) installFilter(isForced bool, filter Filter) error {
	var err error

	if filter.IsRemote() {
		filterDirectory, err := filter.Download(isForced)
		if err != nil {
			return wrapError("could not download filter: ", err)
		}
		filterCollection, err := FilterCollectionFromFilterJson(
			filepath.Join(filterDirectory, "filter.json"))
		if err != nil {
			return fmt.Errorf(
				"could not load \"filter.json\" from path %q, while checking"+
					" for recursive dependencies", filterDirectory,
			)
		}
		p.installFilters(isForced, filterCollection.Filters)
	}

	p.copyFilterData(filter)
	err = filter.DownloadDependencies()
	if err != nil {
		return wrapError("Could not download dependencies: ", err)
	}

	return nil
}

// copyFilterData copies the filter's data to the data folder.
func (p *Profile) copyFilterData(filter Filter) {
	// Move filters 'data' folder contents into 'data'
	// If the localDataPath already exists, we must not overwrite
	// Additionally, if the remote data path doesn't exist, we don't need
	// to do anything
	filterName := filter.GetIdName()
	remoteDataPath := path.Join(filter.GetDownloadPath(), "data")
	localDataPath := path.Join(p.DataPath, filterName)
	if _, err := os.Stat(localDataPath); err == nil {
		Logger.Warnf("Filter %s already has data in the 'data' folder. \n"+
			"You may manually delete this data and reinstall if you "+
			"would like these configuration files to be updated.",
			filterName)
	} else if _, err := os.Stat(remoteDataPath); err == nil {
		// Ensure folder exists
		err = os.MkdirAll(localDataPath, 0666)
		if err != nil {
			Logger.Error("Could not create filter data folder", err) // TODO - I don't think this should break the entire install
		}

		// Copy 'data' to dataPath
		if p.DataPath != "" {
			err = copy.Copy(
				remoteDataPath, localDataPath,
				copy.Options{PreserveTimes: false, Sync: false})
			if err != nil {
				Logger.Error("Could not initialize filter data", err) // TODO - I don't think this should break the entire install
			}
		} else {
			Logger.Warnf("Filter %s has installation data, but the "+
				"dataPath is not set. Skipping.", filterName)
		}
	}
}
