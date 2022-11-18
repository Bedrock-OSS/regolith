package regolith

import (
	"encoding/json"
	"path/filepath"

	"github.com/Bedrock-OSS/go-burrito/burrito"
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
	exeObj, ok := obj["exe"]
	if !ok {
		return nil, burrito.WrappedErrorf(jsonPropertyMissingError, "exe")
	}
	exe, ok := exeObj.(string)
	if !ok {
		return nil, burrito.WrappedErrorf(
			jsonPropertyTypeError, "exe", "string")
	}

	filter.Exe = exe
	return filter, nil
}

func (f *ExeFilter) Run(context RunContext) (bool, error) {
	if err := f.run(f.Settings, context); err != nil {
		return false, burrito.PassError(err)
	}
	return context.IsInterrupted(), nil
}

func (f *ExeFilterDefinition) CreateFilterRunner(
	runConfiguration map[string]interface{},
) (FilterRunner, error) {
	basicFilter, err := filterFromObject(runConfiguration)
	if err != nil {
		return nil, burrito.WrapError(err, filterFromObjectError)
	}
	filter := &ExeFilter{
		Filter:     *basicFilter,
		Definition: *f,
	}
	return filter, nil
}

func (f *ExeFilterDefinition) InstallDependencies(
	*RemoteFilterDefinition, string,
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
	context RunContext,
) error {
	var err error = nil
	if len(settings) == 0 {
		err = executeExeFile(f.Id,
			f.Definition.Exe,
			f.Arguments, context.AbsoluteLocation,
			GetAbsoluteWorkingDirectory(context.DotRegolithPath))
	} else {
		jsonSettings, _ := json.Marshal(settings)
		err = executeExeFile(f.Id,
			f.Definition.Exe,
			append([]string{string(jsonSettings)}, f.Arguments...),
			context.AbsoluteLocation, GetAbsoluteWorkingDirectory(
				context.DotRegolithPath))
	}
	if err != nil {
		return burrito.WrapErrorf(
			err, "Failed to run exe file.\nPath: %s", f.Definition.Exe)
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
		return burrito.WrapErrorf(err, runSubProcessError)
	}
	return nil
}
