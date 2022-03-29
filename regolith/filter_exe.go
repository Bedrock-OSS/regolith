package regolith

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
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

func (f *ExeFilter) Run(absoluteLocation string) error {
	// Disabled filters are skipped
	if f.Disabled {
		Logger.Infof("Filter \"%s\" is disabled, skipping.", f.Id)
		return nil
	}
	Logger.Infof("Running filter %s.", f.Id)
	start := time.Now()
	defer Logger.Debugf("Executed in %s.", time.Since(start))
	return runExeFilter(*f, f.Settings, absoluteLocation)
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

func (f *ExeFilterDefinition) Check() error {
	return nil
}

func (f *ExeFilter) Check() error {
	return f.Definition.Check()
}

func runExeFilter(
	filter ExeFilter, settings map[string]interface{},
	absoluteLocation string,
) error {
	var err error = nil
	if len(settings) == 0 {
		err = executeExeFile(
			filter.Definition.Exe,
			filter.Arguments, absoluteLocation,
			GetAbsoluteWorkingDirectory())
	} else {
		jsonSettings, _ := json.Marshal(settings)
		err = executeExeFile(
			filter.Definition.Exe,
			append([]string{string(jsonSettings)}, filter.Arguments...),
			absoluteLocation, GetAbsoluteWorkingDirectory())
	}
	if err != nil {
		return WrapError(err, "Failed to run shell filter.")
	}
	return nil
}

func executeExeFile(
	exe string, args []string, filterDir string, workingDir string,
) error {
	for i, arg := range args {
		args[i] = strconv.Quote(arg)
	}
	exe = filepath.Join(filterDir, exe)
	cmd := exec.Command(exe, args...)
	cmd.Dir = workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	env, err1 := CreateEnvironmentVariables(filterDir)
	if err1 != nil {
		return WrapErrorf(err1, "Failed to create environment variables.")
	}
	cmd.Env = env
	Logger.Debugf("Running exe file %s:", exe)
	err := cmd.Run()
	if err != nil {
		return WrapError(err, "Failed to run exe file.")
	}
	return nil
}
