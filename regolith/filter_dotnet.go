package regolith

import (
	"encoding/json"
	"os"
	"os/exec"
)

type DotNetFilterDefinition struct {
	FilterDefinition
	Path string `json:"path,omitempty"`
}

type DotNetFilter struct {
	Filter
	Definition DotNetFilterDefinition `json:"-"`
}

func DotNetFilterDefinitionFromObject(id string, obj map[string]interface{}) (*DotNetFilterDefinition, error) {
	filter := &DotNetFilterDefinition{FilterDefinition: *FilterDefinitionFromObject(id)}
	path, ok := obj["path"].(string)
	if !ok {
		return nil, WrappedErrorf(
			"Missing \"path\" property in %s definition.",
			FullFilterToNiceFilterName(filter.Id))
	}
	filter.Path = path
	return filter, nil
}
func (f *DotNetFilter) Run(context RunContext) (bool, error) {
	if err := f.run(context); err != nil {
		return false, PassError(err)
	}
	return context.IsInterrupted(), nil
}

func (f *DotNetFilter) run(context RunContext) error {
	// Run the filter
	if len(f.Settings) == 0 {
		err := RunSubProcess(
			"dotnet",
			append(
				[]string{
					context.AbsoluteLocation + string(os.PathSeparator) +
						f.Definition.Path,
				},
				f.Arguments...,
			),
			context.AbsoluteLocation,
			GetAbsoluteWorkingDirectory(context.DotRegolithPath),
			ShortFilterName(f.Id),
		)
		if err != nil {
			return WrapError(err, "Failed to run .Net filter")
		}
	} else {
		jsonSettings, _ := json.Marshal(f.Settings)
		err := RunSubProcess(
			"dotnet",
			append(
				[]string{
					context.AbsoluteLocation + string(os.PathSeparator) +
						f.Definition.Path, string(jsonSettings)},
				f.Arguments...,
			),
			context.AbsoluteLocation,
			GetAbsoluteWorkingDirectory(context.DotRegolithPath),
			ShortFilterName(f.Id),
		)
		if err != nil {
			return WrapError(err, "Failed to run .Net filter")
		}
	}
	return nil
}

func (f *DotNetFilterDefinition) CreateFilterRunner(runConfiguration map[string]interface{}) (FilterRunner, error) {
	basicFilter, err := FilterFromObject(runConfiguration)
	if err != nil {
		return nil, WrapError(err, "Failed to create .Net filter")
	}
	filter := &DotNetFilter{
		Filter:     *basicFilter,
		Definition: *f,
	}
	return filter, nil
}

func (f *DotNetFilterDefinition) InstallDependencies(*RemoteFilterDefinition, string) error {
	return nil
}

func (f *DotNetFilterDefinition) Check(context RunContext) error {
	_, err := exec.LookPath("dotnet")
	if err != nil {
		return WrapError(
			err,
			".Net not found, download and install it"+
				" from https://dotnet.microsoft.com/download")
	}
	cmd, err := exec.Command("dotnet", "--version").Output()
	if err != nil {
		return WrapError(err, "Failed to check .Net version")
	}
	cmdStr := string(cmd)
	if len(cmdStr) > 1 {
		Logger.Debugf("Found .Net version %s", cmdStr)
	} else {
		Logger.Debugf("Failed to parse .Net version")
	}
	return nil
}

func (f *DotNetFilter) Check(context RunContext) error {
	return f.Definition.Check(context)
}
