package regolith

import (
	"encoding/json"
	"path/filepath"
	"strconv"
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

func (f *ExeFilter) Run(context RunContext) error {
	return runExeFilter(*f, f.Settings, context.AbsoluteLocation)
}

func (f *ExeFilter) Watch(context RunContext) (bool, error) {
	if err := f.Run(context); err != nil {
		return false, err
	}
	return context.Config.IsInterrupted(), nil
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

func runExeFilter(
	filter ExeFilter, settings map[string]interface{},
	absoluteLocation string,
) error {
	var err error = nil
	if len(settings) == 0 {
		err = executeExeFile(filter.Id,
			filter.Definition.Exe,
			filter.Arguments, absoluteLocation,
			GetAbsoluteWorkingDirectory())
	} else {
		jsonSettings, _ := json.Marshal(settings)
		err = executeExeFile(filter.Id,
			filter.Definition.Exe,
			append([]string{string(jsonSettings)}, filter.Arguments...),
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
	for i, arg := range args {
		args[i] = strconv.Quote(arg)
	}
	exe = filepath.Join(filterDir, exe)
	Logger.Debugf("Running exe file %s:", exe)
	err := RunSubProcess(exe, args, filterDir, workingDir, id)
	if err != nil {
		return WrapError(err, "Failed to run exe file.")
	}
	return nil
}
