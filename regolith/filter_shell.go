package regolith

import (
	"encoding/json"
	"github.com/Bedrock-OSS/go-burrito/burrito"
	"os/exec"
	"strings"
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
	commandObj, ok := obj["command"]
	if !ok {
		return nil, burrito.WrapErrorf(nil, jsonPropertyMissingError, "command")
	}
	command, ok := commandObj.(string)
	if !ok {
		return nil, burrito.WrappedErrorf(jsonPropertyTypeError, "command", "string")
	}
	filter.Command = command
	return filter, nil
}

func (f *ShellFilter) Run(context RunContext) (bool, error) {
	if err := f.run(f.Settings, context); err != nil {
		return false, burrito.PassError(err)
	}
	return context.IsInterrupted(), nil
}

func (f *ShellFilterDefinition) CreateFilterRunner(
	runConfiguration map[string]interface{},
) (FilterRunner, error) {
	basicFilter, err := filterFromObject(runConfiguration)
	if err != nil {
		return nil, burrito.WrapError(err, filterFromObjectError)
	}
	filter := &ShellFilter{
		Filter:     *basicFilter,
		Definition: *f,
	}
	return filter, nil
}

func (f *ShellFilterDefinition) InstallDependencies(*RemoteFilterDefinition, string) error {
	return nil
}

func (f *ShellFilterDefinition) Check(context RunContext) error {
	shell, _, err := findShell()
	if err != nil {
		return burrito.WrapError(err, "Shell requirements check failed")
	}
	Logger.Debugf("Using shell: %s", shell)
	return nil
}

func (f *ShellFilter) Check(context RunContext) error {
	return f.Definition.Check(context)
}

var shells = [][]string{
	{"powershell", "-command"}, {"cmd", "/k"}, {"bash", "-c"}, {"sh", "-c"}}

func (f *ShellFilter) run(
	settings map[string]interface{},
	context RunContext,
) error {
	var err error = nil
	if len(settings) == 0 {
		err = executeCommand(f.Id,
			f.Definition.Command,
			f.Arguments, context.AbsoluteLocation,
			GetAbsoluteWorkingDirectory(context.DotRegolithPath))
	} else {
		jsonSettings, _ := json.Marshal(settings)
		err = executeCommand(f.Id,
			f.Definition.Command,
			append([]string{string(jsonSettings)}, f.Arguments...),
			context.AbsoluteLocation,
			GetAbsoluteWorkingDirectory(context.DotRegolithPath))
	}
	if err != nil {
		return burrito.WrapError(err, "Failed to run shell command.")
	}
	return nil
}

func executeCommand(id string,
	command string, args []string, filterDir string, workingDir string,
) error {
	joined := strings.Join(append([]string{command}, args...), " ")
	Logger.Debugf("Executing command: %s", joined)
	shell, arg, err := findShell()
	if err != nil {
		return burrito.WrapError(err, "Unable to find a valid shell.")
	}
	err = RunSubProcess(shell, []string{arg, joined}, filterDir, workingDir, ShortFilterName(id))
	if err != nil {
		return burrito.WrapError(err, runSubProcessError)
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
	return "", "", burrito.WrappedError("Unable to find a valid shell.")
}
