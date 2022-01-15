package regolith

import (
	"errors"
	"path/filepath"
	"time"
)

type filterDefinition struct {
	filter              func(filter Filter, settings map[string]interface{}, absoluteLocation string) error
	installDependencies func(filter Filter, path string) error
	check               func() error
	validateDefinition  func(filter Filter) error
}

var FilterTypes = map[string]filterDefinition{}

func RegisterFilters() {
	RegisterPythonFilter(FilterTypes)
	RegisterNodeJSFilter(FilterTypes)
	RegisterShellFilter(FilterTypes)
	RegisterJavaFilter(FilterTypes)
	RegisterNimFilter(FilterTypes)
}

func DefaultValidateDefinition(filter Filter) error {
	if filter.Script == "" {
		return errors.New("Missing 'script' field in filter definition")
	}
	return nil
}

/*
Run determine whether the filter is remote, standard (from standard
library) or local and executes it using the proper function.

absoluteLocation is an absolute path to the root folder of the filter.
In case of local filters it's a root path of the project.
*/
func (filter *Filter) Run(absoluteLocation string) error {
	// Disabled filters are skipped
	if filter.Disabled == true {
		Logger.Infof("Filter '%s' is disabled, skipping.", filter.GetFriendlyName())
		return nil
	}

	Logger.Infof("Running filter %s", filter.GetFriendlyName())
	start := time.Now()

	// Standard Filter is only filter that doesn't require authentication.
	if filter.Filter != "" {

		// Special handling for our hello world filter,
		// which welcomes new users to Regolith
		if filter.Filter == "hello_world" {
			return RunHelloWorldFilter(filter)
		}

		// Otherwise drop down to standard handling
		err := RunStandardFilter(*filter)
		if err != nil {
			return err
		}
	} else {

		// All other filters require safe mode to be turned off
		if !IsUnlocked() {
			return errors.New("Safe mode is on, which protects you from potentially unsafe code. \nYou may turn it off using 'regolith unlock'.")
		}

		if filter.Url != "" {
			err := RunRemoteFilter(filter.Url, *filter)
			if err != nil {
				return err
			}
		} else {
			if f, ok := FilterTypes[filter.RunWith]; ok {
				if f.validateDefinition != nil {
					err := f.validateDefinition(*filter)
					if err != nil {
						return err
					}
				} else {
					err := DefaultValidateDefinition(*filter)
					if err != nil {
						return err
					}
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
	return RunRemoteFilter(filter.GetDownloadUrl(), filter)
}

func RunHelloWorldFilter(filter *Filter) error {
	Logger.Info(
		"Hello world!\n" +
			"===========================================================\n" +
			" Welcome to Regolith!\n" +
			"\n" +
			" This message is generated from the 'hello_world' filter.\n" +
			" You can delete this filter when you're ready, and replace it with" +
			" Something more useful!\n" +
			"===========================================================\n",
	)

	return nil
}

// RunRemoteFilter loads and runs the content of filter.json from in
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
	filterCollection, err := FilterCollectionFromFilterJson(path)
	if err != nil {
		return err
	}
	for _, filter := range filterCollection.Filters {
		// Overwrite the venvSlot with the parent value
		filter.VenvSlot = parentFilter.VenvSlot
		filter.Arguments = append(filter.Arguments, parentFilter.Arguments...)
		// Join settings from local config to remote definition
		for k, v := range settings {
			filter.Settings[k] = v
		}
		err := filter.Run(absolutePath)
		if err != nil {
			return err
		}
	}
	return nil
}
