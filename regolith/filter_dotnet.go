package regolith

import (
	"encoding/json"
	"os"
	"os/exec"

	"github.com/Bedrock-OSS/go-burrito/burrito"
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
	pathObj, ok := obj["path"]
	if !ok {
		return nil, burrito.WrappedErrorf(jsonPropertyMissingError, "path")
	}
	path, ok := pathObj.(string)
	if !ok {
		return nil, burrito.WrappedErrorf(jsonPropertyTypeError, "path", "string")
	}
	filter.Path = path
	return filter, nil
}
func (f *DotNetFilter) Run(context RunContext) (bool, error) {
	if err := f.run(context); err != nil {
		return false, burrito.PassError(err)
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
			return burrito.WrapError(err, "Failed to run .Net filter")
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
			return burrito.PassError(err)
		}
	}
	return nil
}

func (f *DotNetFilterDefinition) CreateFilterRunner(runConfiguration map[string]interface{}) (FilterRunner, error) {
	basicFilter, err := filterFromObject(runConfiguration)
	if err != nil {
		return nil, burrito.WrapError(err, filterFromObjectError)
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
		return burrito.WrapError(
			err,
			".Net not found, download and install it"+
				" from https://dotnet.microsoft.com/download")
	}
	cmd, err := exec.Command("dotnet", "--version").Output()
	if err != nil {
		return burrito.WrapError(err, "Failed to check .Net version")
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
