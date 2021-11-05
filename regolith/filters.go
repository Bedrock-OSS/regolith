package regolith

import (
	"errors"
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
	RegisterJavaFilter(FilterTypes)
	RegisterNimFilter(FilterTypes)
}

// RunFilter determine whether the filter is remote, standard (from standard
// library) or local and executes it using the proper function. The
// absoluteLocation is an absolute path to the root folder of the filter.
// In case of local filters it's a root path of the project.
func (filter *Filter) RunFilter(absoluteLocation string) error {
	Logger.Infof("Running filter %s", filter.GetFriendlyName())
	start := time.Now()

	// Disabled filters are skipped
	if filter.Disabled == true {
		Logger.Infof("Filter '%s' is disabled, skipping.", filter.GetFriendlyName())
		return nil
	}

	// Standard Filter is only filter that doesn't require authentication.
	if filter.Filter != "" {
		err := RunStandardFilter(*filter)
		if err != nil {
			return err
		}
	} else {

		// All other filters require safe mode to be turned off
		if !IsUnlocked() {
			return errors.New("Safe mode is on. Please turn it off using 'regolith unlock'.")
		}

		if filter.Url != "" {
			err := RunRemoteFilter(filter.Url, *filter)
			if err != nil {
				return err
			}
		} else {
			if f, ok := FilterTypes[filter.RunWith]; ok {
				if filter.Script == "" {
					return errors.New("Missing 'script' field in filter definition")
				}
				err := f.filter(*filter, filter.Settings, absoluteLocation)
				if err != nil {
					return err
				}
			} else {
				Logger.Warnf("Filter type '%s' not supported", filter.RunWith)
			}
			Logger.Debugf("Executed in %s", time.Since(start))
		}
	}
	return nil
}

// RunStandardFilter runs a filter from standard Bedrock-OSS library. The
// function doesn't test if the filter passed on input is standard.
func RunStandardFilter(filter Filter) error {
	Logger.Debugf("RunStandardFilter '%s'", filter.Filter)
	return RunRemoteFilter(FilterNameToUrl(StandardLibraryUrl, filter.Filter), filter)
}

// RunRemoteFilter runs loads and runs the content of filter.json from in
// regolith cache. The url is the URL of the filter from which the filter
// was downloaded (used to specify its path in the cache). The parentFilter
// is a filter that caused the downloading. Some properties of
// parentFilter are propagated to its children.
func RunRemoteFilter(url string, parentFilter Filter) error {
	settings := parentFilter.Settings
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
		filter.Arguments = append(filter.Arguments, parentFilter.Arguments...)
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
