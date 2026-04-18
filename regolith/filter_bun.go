package regolith

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"

	"github.com/Bedrock-OSS/go-burrito/burrito"
)

type BunFilterDefinition struct {
	FilterDefinition
	Script string `json:"script,omitempty"`
}

type BunFilter struct {
	Filter
	Definition BunFilterDefinition `json:"-"`
}

func BunFilterDefinitionFromObject(id string, obj map[string]any) (*BunFilterDefinition, error) {
	filter := &BunFilterDefinition{FilterDefinition: *FilterDefinitionFromObject(id)}
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

func (f *BunFilter) run(context RunContext) error {
	absWorkingDir, err := GetAbsoluteWorkingDirectory(context.DotRegolithPath)
	if err != nil {
		return burrito.WrapError(err, getAbsoluteWorkingDirectoryError)
	}
	userConfig, err := getCombinedUserConfig()
	if err != nil {
		return burrito.WrapError(err, getUserConfigError)
	}
	bunRunner := *userConfig.BunRunner
	// Run filter
	if len(f.Settings) == 0 {
		err := RunSubProcess(
			bunRunner,
			append([]string{
				"run",
				context.AbsoluteLocation + string(os.PathSeparator) +
					f.Definition.Script},
				f.Arguments...,
			),
			context.AbsoluteLocation,
			absWorkingDir,
			ShortFilterName(f.Id),
		)
		if err != nil {
			return burrito.WrapError(err, runSubProcessError)
		}
	} else {
		jsonSettings, _ := json.Marshal(f.Settings)
		err := RunSubProcess(
			bunRunner,
			append([]string{
				"run",
				context.AbsoluteLocation + string(os.PathSeparator) +
					f.Definition.Script,
				string(jsonSettings)}, f.Arguments...),
			context.AbsoluteLocation,
			absWorkingDir,
			ShortFilterName(f.Id),
		)
		if err != nil {
			return burrito.WrapError(err, runSubProcessError)
		}
	}
	return nil
}

func (f *BunFilter) Run(context RunContext) (bool, error) {
	if err := f.run(context); err != nil {
		return false, burrito.PassError(err)
	}
	return context.IsInterrupted(), nil
}

func (f *BunFilterDefinition) CreateFilterRunner(runConfiguration map[string]any, id string) (FilterRunner, error) {
	basicFilter, err := filterFromObject(runConfiguration, id)
	if err != nil {
		return nil, burrito.WrapError(err, filterFromObjectError)
	}
	filter := &BunFilter{
		Filter:     *basicFilter,
		Definition: *f,
	}
	return filter, nil
}

func (f *BunFilterDefinition) Check(context RunContext) error {
	userConfig, err := getCombinedUserConfig()
	if err != nil {
		return burrito.WrapError(err, getUserConfigError)
	}
	bunRunner := *userConfig.BunRunner
	_, err = exec.LookPath(bunRunner)
	if err != nil {
		return burrito.WrapError(
			err, "Bun not found, download and install it from"+
				" https://bun.com/")
	}
	cmd, err := exec.Command(bunRunner, "--version").Output()
	if err != nil {
		return burrito.WrapError(err, "Failed to check Bun version")
	}
	a := strings.TrimPrefix(strings.Trim(string(cmd), " \n\t"), "v")
	Logger.Debugf("Found Bun version %s", a)
	return nil
}

func (f *BunFilterDefinition) InstallDependencies(parent *RemoteFilterDefinition, dotRegolithPath string) error {
	installLocation := ""
	// Install dependencies
	if parent != nil {
		installLocation = parent.GetDownloadPath(dotRegolithPath)
	}
	Logger.Infof("Downloading dependencies for %s...", f.Id)
	if hasPackageJson(installLocation) {
		userConfig, err := getCombinedUserConfig()
		if err != nil {
			return burrito.WrapError(err, getUserConfigError)
		}
		bunRunner := *userConfig.BunRunner
		Logger.Info("Installing bun dependencies...")
		err = RunSubProcess(bunRunner, []string{"install", "--silent"}, installLocation, installLocation, ShortFilterName(f.Id))
		if err != nil {
			return burrito.WrapErrorf(
				err, "Failed to run bun and install dependencies."+
					"\nFilter name: %s", f.Id)
		}
	}
	Logger.Infof("Dependencies for %s installed successfully", f.Id)
	return nil
}

func (f *BunFilter) Check(context RunContext) error {
	return f.Definition.Check(context)
}
