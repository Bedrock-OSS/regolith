package regolith

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"time"
)

type JavaFilterDefinition struct {
	FilterDefinition
	Script string `json:"script,omitempty"`
}

type JavaFilter struct {
	Filter
	Definition JavaFilterDefinition `json:"-"`
}

func JavaFilterDefinitionFromObject(id string, obj map[string]interface{}) (*JavaFilterDefinition, error) {
	filter := &JavaFilterDefinition{FilterDefinition: *FilterDefinitionFromObject(id)}
	script, ok := obj["script"].(string)
	if !ok {
		return nil, WrappedErrorf(
			"Missing \"script\" property in filter definition %q.",
			filter.Id)
	}
	filter.Script = script
	return filter, nil
}

func (f *JavaFilter) Run(absoluteLocation string) error {
	// Disabled filters are skipped
	if f.Disabled {
		Logger.Infof("Filter \"%s\" is disabled, skipping.", f.Id)
		return nil
	}
	Logger.Infof("Running filter %s", f.Id)
	start := time.Now()
	defer Logger.Debugf("Executed in %s", time.Since(start))

	// Run the filter
	if len(f.Settings) == 0 {
		err := RunSubProcess(
			"java",
			append(
				[]string{
					"-jar", absoluteLocation + string(os.PathSeparator) +
						f.Definition.Script,
				},
				f.Arguments...,
			),
			absoluteLocation,
			GetAbsoluteWorkingDirectory(),
		)
		if err != nil {
			return WrapError(err, "Failed to run Java filter")
		}
	} else {
		jsonSettings, _ := json.Marshal(f.Settings)
		err := RunSubProcess(
			"java",
			append(
				[]string{
					"-jar", absoluteLocation + string(os.PathSeparator) +
						f.Definition.Script, string(jsonSettings)},
				f.Arguments...,
			),
			absoluteLocation,
			GetAbsoluteWorkingDirectory(),
		)
		if err != nil {
			return WrapError(err, "Failed to run Java filter")
		}
	}
	return nil
}

func (f *JavaFilterDefinition) CreateFilterRunner(runConfiguration map[string]interface{}) (FilterRunner, error) {
	basicFilter, err := FilterFromObject(runConfiguration)
	if err != nil {
		return nil, WrapError(err, "Failed to create Java filter")
	}
	filter := &JavaFilter{
		Filter:     *basicFilter,
		Definition: *f,
	}
	return filter, nil
}

func (f *JavaFilterDefinition) InstallDependencies(parent *RemoteFilterDefinition) error {
	return nil
}

func (f *JavaFilterDefinition) Check() error {
	_, err := exec.LookPath("java")
	if err != nil {
		return WrapError(
			err,
			"Java not found, download and install it"+
				" from https://adoptopenjdk.net/")
	}
	cmd, err := exec.Command("java", "--version").Output()
	if err != nil {
		return WrapError(err, "Failed to check Java version")
	}
	a := strings.Split(strings.Trim(string(cmd), " \n\t"), " ")
	if len(a) > 1 {
		Logger.Debugf("Found Java %s version %s", a[0], a[1])
	} else {
		Logger.Debugf("Failed to parse Java version")
	}
	return nil
}

func (f *JavaFilter) Check() error {
	return f.Definition.Check()
}
