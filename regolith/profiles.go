package regolith

import (
	"fmt"
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

	err = copy.Copy(config.Packs.BehaviorFolder, ".regolith/tmp/BP", copy.Options{PreserveTimes: false, Sync: false})
	if err != nil {
		return err
	}

	err = copy.Copy(config.Packs.ResourceFolder, ".regolith/tmp/RP", copy.Options{PreserveTimes: false, Sync: false})
	if err != nil {
		return err
	}

	// Copy the contents of 'data' folder to '.regolith/tmp'
	if profile.DataPath != "" { // datapath copied only if specified
		err = copy.Copy(profile.DataPath, ".regolith/tmp/data", copy.Options{PreserveTimes: false, Sync: false})
		if err != nil {
			return err
		}
	} else { // create empty data path otherwise.
		err = os.MkdirAll(".regolith/data", 0666)
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
		Logger.Warn("Profile flagged as unsafe. Exercise caution!")
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
	for filter := range profile.Filters {
		filter := profile.Filters[filter]
		path, _ := filepath.Abs(".")
		err := filter.RunFilter(path)
		if err != nil {
			return wrapError(fmt.Sprintf("%s failed", filter.GetFriendlyName()), err)
		}
	}

	// Export files
	Logger.Info("Moving files to target directory")
	start := time.Now()
	err = ExportProject(profile, project.Name)
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
