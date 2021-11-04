package regolith

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	getter "github.com/hashicorp/go-getter"
	"github.com/otiai10/copy"
)

const StandardLibraryUrl = "github.com/Bedrock-OSS/regolith-filters"

// UrlToPath returns regolith cache path for given URL.
func UrlToPath(url string) string {
	return ".regolith/cache/filters/" + url
}

// FilterNameToUrl returns the URL of a standard filter based on its name.
func FilterNameToUrl(libraryUrl string, name string) string {
	return fmt.Sprintf("%s//%s", libraryUrl, name)
}

func ValidateUrl(url string) error {
	if !strings.HasPrefix(url, "http") {
		return fmt.Errorf("Invalid URL: %s", url)
	}
	return nil
}

// IsRemoteFilterCached checks whether the filter of given URL is already saved
// in cache.
func IsRemoteFilterCached(url string) bool {
	_, err := os.Stat(UrlToPath(url))
	return err == nil
}

// Recursively install dependencies for the entire config.
//  - Force mode will overwrite existing dependencies.
//  - Non-force mode will only install dependencies that are not already installed.
func InstallDependencies(isForced bool) error {
	Logger.Infof("Installing dependencies...")

	project, err := LoadConfig()
	if err != nil {
		return wrapError("Failed to load project config", err)
	}

	err = os.MkdirAll(".regolith/cache/filters", 0666)
	if err != nil {
		return wrapError("Could not create .regolith/cache/filters", err)
	}

	// Special path for virtual environments for python
	err = os.MkdirAll(".regolith/cache/venvs", 0666)
	if err != nil {
		return wrapError("Could not create .regolith/cache/venvs", err)
	}

	for _, profile := range project.Profiles {
		err := profile.Install(isForced)
		if err != nil {
			return wrapError("Could not install dependency", err)
		}
	}

	Logger.Infof("Dependencies installed.")
	return nil
}

// LoadFilterJsonProfile loads a profile from path to filter.json file of
// a remote filter and propagates the properties of the parent filter (the
// filter in config.json or other remote filter that caused creation of this
// profile).
func LoadFilterJsonProfile(filterJsonPath string, parentFilter Filter) (*Profile, error) {
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
	return &remoteProfile, nil
}

// Install installs all of the dependencies of the profile
func (p *Profile) Install(isForced bool) error {
	for filter := range p.Filters {
		filter := &p.Filters[filter] // Using pointer is faster than creating copies in the loop and gives more options
		downloadPath, err := filter.Download(isForced)
		// TODO - we could use type switch to handle different kinds of errors
		// here. Download can fail on downloading or on cleaning the download
		// path. It can also fail when isForced is false and the path already
		// exists.
		if err != nil {
			return wrapError("Could not download filter", err) // TODO - I don't think this should break the entire install
		} else if downloadPath == "" {
			continue // Not a remote filter, skip
		}

		// Move filters 'data' folder contents into 'data'
		filterName := filter.GetIdName()
		localDataPath := path.Join(p.DataPath, filterName)
		remoteDataPath := path.Join(downloadPath, "data")

		// If the filterDataPath already exists, we must not overwrite
		// Additionally, if the remote data path doesn't exist, we don't need
		// to do anything
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

		// Create profile from filter.json file
		remoteProfile, err := LoadFilterJsonProfile(
			filepath.Join(downloadPath, "filter.json"), *filter)
		if err != nil {
			return err // TODO - I don't think this should break the entire install. Just remove the files and continue.
		}

		// Install dependencies of remote filters. Recursion ends when there
		// is no more nested remote dependencies.
		err = remoteProfile.Install(isForced)
		if err != nil {
			return err
		}

		// Install filter dependencies (python/nodejs modules etc.)
		// TODO - can we move this out of the loop? This way the filters from
		// config.json would also run their install function and we could
		// modify the code a little to enable dependencies for them.
		for _, filter := range remoteProfile.Filters {
			if filter.RunWith != "" {
				if f, ok := FilterTypes[filter.RunWith]; ok {
					err := f.install(filter, downloadPath)
					if err != nil {
						return wrapError(fmt.Sprintf(
							"Couldn't install filter %s", filter.Name), err)
					}
				} else {
					Logger.Warnf(
						"Filter type '%s' not supported", filter.RunWith)
				}
			}
		}
	}
	return nil
}

// GetFilterPath returns URL for downloading the filter or empty string
// if the filter is not remote
func (f *Filter) GetDownloadUrl() string {
	if f.Url != "" {
		return FilterNameToUrl(f.Url, f.Filter)
	} else if f.Filter != "" {
		return FilterNameToUrl(StandardLibraryUrl, f.Filter)
	}
	return ""
}

// GetIdName returns the name that identifies the filter. This name is used to
// create and access the data folder for the filter. This property only makes
// sense for remote filters. Non-remote filters return empty string.
func (f *Filter) GetIdName() string {
	if f.Filter != "" {
		return f.Filter
	} else if f.Url != "" {
		splitUrl := strings.Split(f.Url, "/")
		return splitUrl[len(splitUrl)-1]
	}
	return ""
}

// Download downloads the filter and returns the download path. If the filter
// is not remote, it returns empty string.
func (f *Filter) Download(isForced bool) (string, error) {
	url := f.GetDownloadUrl()
	if url == "" {
		return "", nil // No dependency to install
	}

	// Download the filter into the cache folder
	downloadPath := UrlToPath(url)

	// If downloadPath already exists, we don't need to download it again.
	// Force mode allows overwriting.
	if _, err := os.Stat(downloadPath); err == nil {
		if !isForced {
			Logger.Warnf("Dependency %s already installed, skipping. Run "+
				"with '-f' to force.", url)
			return "", nil
		} else {
			Logger.Warnf("Dependency %s already installed, but forcing "+
				"installation.", url)
			err := os.RemoveAll(downloadPath)
			if err != nil {
				return "", wrapError("Could not remove installed filter", err)
			}
		}
	}

	Logger.Infof("Installing dependency %s...", url)

	// Download the filter fresh
	ok, err := DownloadGitHubUrl(url, downloadPath)
	if err != nil {
		Logger.Debug(err)
	}
	if !ok {
		Logger.Debug(
			"Failed to download filter " + f.Filter + " without git")
		err := getter.Get(downloadPath, url)
		if err != nil {
			return "", err
		}
	}

	// Remove 'test' folder, which may be installed via git-getter library
	// This is a workaround for cases where our own getter library is not
	// able to download the filter.
	testFolder := path.Join(downloadPath, "test")
	if _, err := os.Stat(testFolder); err == nil {
		os.RemoveAll(testFolder)
		if err != nil {
			Logger.Debug("Could not remove test folder", err)
		}
	}
	return downloadPath, nil
}
