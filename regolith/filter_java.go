package regolith

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"

	"github.com/Bedrock-OSS/go-burrito/burrito"
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
	var path string
	pathObj, ok := obj["path"]
	if !ok {
		scriptObj, ok := obj["script"]
		if !ok {
			return nil, burrito.WrappedErrorf(jsonPropertyMissingError, "path")
		}
		Logger.Warnf("\"script\" property in %s definition is deprecated, use \"path\" instead.", FullFilterToNiceFilterName(filter.Id))
		path, ok = scriptObj.(string)
		if !ok {
			return nil, burrito.WrappedErrorf(jsonPropertyTypeError, "script", "string")
		}

	} else {
		path, ok = pathObj.(string)
		if !ok {
			return nil, burrito.WrappedErrorf(jsonPropertyTypeError, "path", "string")
		}
	}
	filter.Script = path
	return filter, nil
}
func (f *JavaFilter) Run(context RunContext) (bool, error) {
	if err := f.run(context); err != nil {
		return false, burrito.PassError(err)
	}
	return context.IsInterrupted(), nil
}

func (f *JavaFilter) run(context RunContext) error {
	// Run the filter
	if len(f.Settings) == 0 {
		err := RunSubProcess(
			"java",
			append(
				[]string{
					"-jar", context.AbsoluteLocation + string(os.PathSeparator) +
						f.Definition.Script,
				},
				f.Arguments...,
			),
			context.AbsoluteLocation,
			GetAbsoluteWorkingDirectory(context.DotRegolithPath),
			ShortFilterName(f.Id),
		)
		if err != nil {
			return burrito.WrapError(err, "Failed to run Java filter")
		}
	} else {
		jsonSettings, _ := json.Marshal(f.Settings)
		err := RunSubProcess(
			"java",
			append(
				[]string{
					"-jar", context.AbsoluteLocation + string(os.PathSeparator) +
						f.Definition.Script, string(jsonSettings)},
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

func (f *JavaFilterDefinition) CreateFilterRunner(runConfiguration map[string]interface{}, id string) (FilterRunner, error) {
	basicFilter, err := filterFromObject(runConfiguration, id)
	if err != nil {
		return nil, burrito.WrapError(err, filterFromObjectError)
	}
	filter := &JavaFilter{
		Filter:     *basicFilter,
		Definition: *f,
	}
	return filter, nil
}

func (f *JavaFilterDefinition) InstallDependencies(*RemoteFilterDefinition, string) error {
	return nil
}

func (f *JavaFilterDefinition) Check(context RunContext) error {
	_, err := exec.LookPath("java")
	if err != nil {
		return burrito.WrapError(
			err,
			"Java not found, download and install it"+
				" from https://adoptopenjdk.net/")
	}
	cmd, err := exec.Command("java", "-version").Output()
	if err != nil {
		return burrito.WrapError(err, "Failed to check Java version")
	}
	a := strings.Split(strings.Trim(string(cmd), " \n\t"), " ")
	if len(a) > 1 {
		Logger.Debugf("Found Java %s version %s", a[0], a[1])
	} else {
		Logger.Debugf("Failed to parse Java version.\nVersion string: %s", a)
	}
	return nil
}

func (f *JavaFilter) Check(context RunContext) error {
	return f.Definition.Check(context)
}
