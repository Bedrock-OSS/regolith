package regolith

import (
	"encoding/json"
	"path/filepath"
	"runtime"
)

type ExeFilterDefinition struct {
	FilterDefinition
	Exe      string `json:"exe,omitempty"`
	ExeWin   string `json:"exeWindows,omitempty"`
	ExeLinux string `json:"exeLinux,omitempty"`
	ExeMac   string `json:"exeMac,omitempty"`
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
		return nil, WrappedErrorf(jsonPropertyMissingError, "exe")
	}
	exe, ok := exeObj.(string)
	if !ok {
		return nil, WrappedErrorf(
			jsonPropertyTypeError, "exe", "string")
	}

	filter.Exe = exe
	if exeObj, ok = obj["exeWindows"]; ok {
		if exe, ok = exeObj.(string); ok {
			filter.ExeWin = exe
		} else {
			return nil, WrappedErrorf(
				jsonPropertyTypeError, "exeWindows", "string")
		}
	}
	if exeObj, ok = obj["exeLinux"]; ok {
		if exe, ok = exeObj.(string); ok {
			filter.ExeLinux = exe
		} else {
			return nil, WrappedErrorf(
				jsonPropertyTypeError, "exeLinux", "string")
		}
	}
	if exeObj, ok = obj["exeMac"]; ok {
		if exe, ok = exeObj.(string); ok {
			filter.ExeMac = exe
		} else {
			return nil, WrappedErrorf(
				jsonPropertyTypeError, "exeMac", "string")
		}
	}
	return filter, nil
}

func (f *ExeFilter) Run(context RunContext) (bool, error) {
	if err := f.run(f.Settings, context); err != nil {
		return false, PassError(err)
	}
	return context.IsInterrupted(), nil
}

func (f *ExeFilterDefinition) CreateFilterRunner(
	runConfiguration map[string]interface{},
) (FilterRunner, error) {
	basicFilter, err := filterFromObject(runConfiguration)
	if err != nil {
		return nil, WrapError(err, filterFromObjectError)
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
	exe := f.Definition.Exe
	if runtime.GOOS == "windows" && f.Definition.ExeWin != "" {
		exe = f.Definition.ExeWin
	}
	if runtime.GOOS == "linux" && f.Definition.ExeLinux != "" {
		exe = f.Definition.ExeLinux
	}
	if runtime.GOOS == "darwin" && f.Definition.ExeMac != "" {
		exe = f.Definition.ExeMac
	}
	if len(settings) == 0 {
		err = executeExeFile(f.Id,
			exe,
			f.Arguments, context.AbsoluteLocation,
			GetAbsoluteWorkingDirectory(context.DotRegolithPath))
	} else {
		jsonSettings, _ := json.Marshal(settings)
		err = executeExeFile(f.Id,
			exe,
			append([]string{string(jsonSettings)}, f.Arguments...),
			context.AbsoluteLocation, GetAbsoluteWorkingDirectory(
				context.DotRegolithPath))
	}
	if err != nil {
		return WrapErrorf(
			err, "Failed to run exe file.\nPath: %s", exe)
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
		return WrapErrorf(err, runSubProcessError)
	}
	return nil
}
