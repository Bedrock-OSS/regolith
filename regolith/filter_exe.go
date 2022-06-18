package regolith

import (
	"encoding/json"
	"path/filepath"
)

type ExeFilterDefinition struct {
	FilterDefinition
	Exe string `json:"exe,omitempty"`
}

type ExeFilter struct {
	Filter
	Definition ExeFilterDefinition `json:"definition,omitempty"`
}

func ExeFilterDefinitionFromObject(
	id string, obj map[string]interface{},
) (*ExeFilterDefinition, error) {
	filter := &ExeFilterDefinition{
		FilterDefinition: *FilterDefinitionFromObject(id)}
	exe, ok := obj["exe"].(string)
	if !ok {
		return nil, WrapErrorf(
			nil,
			"Missing \"exe\" property in filter definition %q.", filter.Id)
	}
	filter.Exe = exe
	return filter, nil
}

func (f *ExeFilter) Run(context RunContext) (bool, error) {
	if err := f.run(f.Settings, context.AbsoluteLocation); err != nil {
		return false, PassError(err)
	}
	return context.IsInterrupted(), nil
}

func (f *ExeFilterDefinition) CreateFilterRunner(
	runConfiguration map[string]interface{},
) (FilterRunner, error) {
	basicFilter, err := FilterFromObject(runConfiguration)
	if err != nil {
		return nil, WrapError(err, "Failed to create exe filter.")
	}
	filter := &ExeFilter{
		Filter:     *basicFilter,
		Definition: *f,
	}
	return filter, nil
}

func (f *ExeFilterDefinition) InstallDependencies(
	parent *RemoteFilterDefinition,
) error {
	return nil
}

func (f *ExeFilterDefinition) Check(context RunContext) error {
	return nil
}

func (f *ExeFilter) Check(context RunContext) error {
	return f.Definition.Check(context)
}

func (f *ExeFilter) run(
	settings map[string]interface{},
	absoluteLocation string,
) error {
	var err error = nil
	if len(settings) == 0 {
		err = executeExeFile(f.Id,
			f.Definition.Exe,
			f.Arguments, absoluteLocation,
			GetAbsoluteWorkingDirectory())
	} else {
		jsonSettings, _ := json.Marshal(settings)
		err = executeExeFile(f.Id,
			f.Definition.Exe,
			append([]string{string(jsonSettings)}, f.Arguments...),
			absoluteLocation, GetAbsoluteWorkingDirectory())
	}
	if err != nil {
		return WrapError(err, "Failed to run shell filter.")
	}
	return nil
}

func executeExeFile(id string,
	exe string, args []string, filterDir string, workingDir string,
) error {
	exe = filepath.Join(filterDir, exe)
	Logger.Debugf("Running exe file %s:", exe)
	err := RunSubProcess(exe, args, filterDir, workingDir, id)
	if err != nil {
		return WrapError(err, "Failed to run exe file.")
	}
	return nil
}
