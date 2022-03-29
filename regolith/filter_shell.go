package regolith

import (
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type ShellFilterDefinition struct {
	FilterDefinition
	Command string `json:"command,omitempty"`
}

type ShellFilter struct {
	Filter
	Definition ShellFilterDefinition `json:"definition,omitempty"`
}

func ShellFilterDefinitionFromObject(
	id string, obj map[string]interface{},
) (*ShellFilterDefinition, error) {
	filter := &ShellFilterDefinition{
		FilterDefinition: *FilterDefinitionFromObject(id)}
	command, ok := obj["command"].(string)
	if !ok {
		return nil, WrapErrorf(
			nil,
			"Missing \"command\" property in filter definition %q.", filter.Id)
	}
	filter.Command = command
	return filter, nil
}

func (f *ShellFilter) Run(absoluteLocation string) error {
	// Disabled filters are skipped
	if f.Disabled {
		Logger.Infof("Filter \"%s\" is disabled, skipping.", f.Id)
		return nil
	}
	Logger.Infof("Running filter %s.", f.Id)
	start := time.Now()
	defer Logger.Debugf("Executed in %s.", time.Since(start))
	return runShellFilter(*f, f.Settings, absoluteLocation)
}

func (f *ShellFilterDefinition) CreateFilterRunner(
	runConfiguration map[string]interface{},
) (FilterRunner, error) {
	basicFilter, err := FilterFromObject(runConfiguration)
	if err != nil {
		return nil, WrapError(err, "Failed to create shell filter.")
	}
	filter := &ShellFilter{
		Filter:     *basicFilter,
		Definition: *f,
	}
	return filter, nil
}

func (f *ShellFilterDefinition) InstallDependencies(
	parent *RemoteFilterDefinition,
) error {
	return nil
}

func (f *ShellFilterDefinition) Check() error {
	shell, _, err := findShell()
	if err != nil {
		return WrapError(err, "Shell requirements check failed")
	}
	Logger.Debugf("Using shell: %s", shell)
	return nil
}

func (f *ShellFilter) Check() error {
	return f.Definition.Check()
}

var shells = [][]string{
	{"powershell", "-command"}, {"cmd", "/k"}, {"bash", "-c"}, {"sh", "-c"}}

func runShellFilter(
	filter ShellFilter, settings map[string]interface{},
	absoluteLocation string,
) error {
	var err error = nil
	if len(settings) == 0 {
		err = executeCommand(
			filter.Definition.Command,
			filter.Arguments, absoluteLocation,
			GetAbsoluteWorkingDirectory())
	} else {
		jsonSettings, _ := json.Marshal(settings)
		err = executeCommand(
			filter.Definition.Command,
			append([]string{string(jsonSettings)}, filter.Arguments...),
			absoluteLocation, GetAbsoluteWorkingDirectory())
	}
	if err != nil {
		return WrapError(err, "Failed to run shell filter.")
	}
	return nil
}

func executeCommand(
	command string, args []string, filterDir string, workingDir string,
) error {
	for i, arg := range args {
		args[i] = strconv.Quote(arg)
	}
	joined := strings.Join(append([]string{command}, args...), " ")
	Logger.Debugf("Executing command: %s", joined)
	shell, arg, err := findShell()
	if err != nil {
		return WrapError(err, "Unable to find a valid shell.")
	}
	cmd := exec.Command(shell, arg, joined)
	cmd.Dir = workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	env, err1 := CreateEnvironmentVariables(filterDir)
	if err1 != nil {
		return WrapErrorf(err1, "Failed to create environment variables.")
	}
	cmd.Env = env

	err = cmd.Run()

	if err != nil {
		return WrapError(err, "Failed to run shell script.")
	}
	return nil
}

func findShell() (string, string, error) {
	for _, shell := range shells {
		_, err := exec.LookPath(shell[0])
		if err == nil {
			return shell[0], shell[1], nil
		}
	}
	return "", "", WrappedError("Unable to find a valid shell.")
}
