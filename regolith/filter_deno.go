package regolith

import (
	"encoding/json"
	"github.com/Bedrock-OSS/go-burrito/burrito"
	"os"
	"os/exec"
	"strings"
)

type DenoFilterDefinition struct {
	FilterDefinition
	Script string `json:"script,omitempty"`
}

type DenoFilter struct {
	Filter
	Definition DenoFilterDefinition `json:"-"`
}

func DenoFilterDefinitionFromObject(id string, obj map[string]interface{}) (*DenoFilterDefinition, error) {
	filter := &DenoFilterDefinition{FilterDefinition: *FilterDefinitionFromObject(id)}
	scriptObj, ok := obj["script"]
	if !ok {
		return nil, burrito.WrappedErrorf(jsonPropertyMissingError, "script")
	}
	script, ok := scriptObj.(string)
	if !ok {
		return nil, burrito.WrappedErrorf(
			jsonPropertyTypeError, "script", "string")
	}
	filter.Script = script
	return filter, nil
}

func (f *DenoFilter) run(context RunContext) error {
	// Run filter
	if len(f.Settings) == 0 {
		err := RunSubProcess(
			"deno",
			append([]string{
				"run", "--allow-all",
				context.AbsoluteLocation + string(os.PathSeparator) +
					f.Definition.Script},
				f.Arguments...,
			),
			context.AbsoluteLocation,
			GetAbsoluteWorkingDirectory(context.DotRegolithPath),
			ShortFilterName(f.Id),
		)
		if err != nil {
			return burrito.WrapError(err, runSubProcessError)
		}
	} else {
		jsonSettings, _ := json.Marshal(f.Settings)
		err := RunSubProcess(
			"deno",
			append([]string{
				"run",
				context.AbsoluteLocation + string(os.PathSeparator) +
					f.Definition.Script,
				string(jsonSettings)}, f.Arguments...),
			context.AbsoluteLocation,
			GetAbsoluteWorkingDirectory(context.DotRegolithPath),
			ShortFilterName(f.Id),
		)
		if err != nil {
			return burrito.WrapError(err, runSubProcessError)
		}
	}
	return nil
}

func (f *DenoFilter) Run(context RunContext) (bool, error) {
	if err := f.run(context); err != nil {
		return false, burrito.PassError(err)
	}
	return context.IsInterrupted(), nil
}

func (f *DenoFilterDefinition) CreateFilterRunner(runConfiguration map[string]interface{}) (FilterRunner, error) {
	basicFilter, err := filterFromObject(runConfiguration)
	if err != nil {
		return nil, burrito.WrapError(err, filterFromObjectError)
	}
	filter := &DenoFilter{
		Filter:     *basicFilter,
		Definition: *f,
	}
	return filter, nil
}

func (f *DenoFilterDefinition) Check(context RunContext) error {
	_, err := exec.LookPath("deno")
	if err != nil {
		return burrito.WrapError(
			err, "Deno not found, download and install it from"+
				" https://deno.land/")
	}
	cmd, err := exec.Command("deno", "--version").Output()
	if err != nil {
		return burrito.WrapError(err, "Failed to check Deno version")
	}
	a := strings.TrimPrefix(strings.Trim(string(cmd), " \n\t"), "v")
	Logger.Debugf("Found Deno version %s", a)
	return nil
}

func (f *DenoFilterDefinition) InstallDependencies(
	parent *RemoteFilterDefinition, dotRegolithPath string,
) error {
	return nil
}

func (f *DenoFilter) Check(context RunContext) error {
	return f.Definition.Check(context)
}
