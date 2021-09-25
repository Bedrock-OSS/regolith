package regolith

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"time"
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

// RunFilter determine whether the filter is remote, standard (from standard
// library) or local and executes it using the proper function. The
// absoluteLocation is an absolute path to the root folder of the filter.
// In case of local filters it's a root path of the project.
func (filter *Filter) RunFilter(absoluteLocation string) error {
	Logger.Infof("%s...", filter.GetName())
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
		Logger.Debugf("Executed in %s", time.Since(start))
	}
	Logger.Infof("%s done", filter.GetName())
	return nil
}

// RunStandardFilter runs a filter from standard Bedrock-OSS library. The
// function doesn't test if the filter passed on input is standard.
func RunStandardFilter(filter Filter) error {
	Logger.Debugf("RunStandardFilter '%s'", filter.Filter)
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
// was downloaded (used to specify its path in the cache). The parentFilter
// is a filter that caused the downloading. Some properties of
// parentFilter are propagated to its children.
func RunRemoteFilter(url string, parentFilter Filter) error {
	settings := parentFilter.Settings
	// TODO - I think this also should be used somehow:
	// arguments := parentFilter.Arguments
	Logger.Debugf("RunRemoteFilter '%s'", url)
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
