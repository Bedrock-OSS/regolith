package regolith

import (
	"encoding/json"
	"errors"
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

func ShellFilterDefinitionFromObject(id string, obj map[string]interface{}) *ShellFilterDefinition {
	filter := &ShellFilterDefinition{FilterDefinition: *FilterDefinitionFromObject(id)}
	command, ok := obj["command"].(string)
	if !ok {
		Logger.Fatalf("Could find command in filter defnition %q", filter.Id)
	}
	filter.Command = command
	return filter
}

func ShellFilterFromObject(obj map[string]interface{}, definition ShellFilterDefinition) *ShellFilter {
	filter := &ShellFilter{
		Filter:     *FilterFromObject(obj),
		Definition: definition,
	}
	return filter
}

func (f *ShellFilter) Run(absoluteLocation string) error {
	// Disabled filters are skipped
	if f.Disabled {
		Logger.Infof("Filter '%s' is disabled, skipping.", f.GetFriendlyName())
		return nil
	}
	Logger.Infof("Running filter %s", f.GetFriendlyName())
	start := time.Now()
	defer Logger.Debugf("Executed in %s", time.Since(start))
	return runShellFilter(*f, f.Settings, absoluteLocation)
}

func (f *ShellFilterDefinition) CreateFilterRunner(runConfiguration map[string]interface{}) FilterRunner {
	return ShellFilterFromObject(runConfiguration, *f)
}

func (f *ShellFilterDefinition) InstallDependencies(parent *RemoteFilterDefinition) error {
	return nil
}

func (f *ShellFilterDefinition) Check() error {
	return checkShellRequirements()
}

func (f *ShellFilter) Check() error {
	return f.Definition.Check()
}

func (f *ShellFilter) CopyArguments(parent *RemoteFilter) {
	f.Arguments = parent.Arguments
	f.Settings = parent.Settings
}

func (f *ShellFilter) GetFriendlyName() string {
	if f.Name != "" {
		return f.Name
	}
	return "Unnamed Shell filter"
}

var shells = [][]string{{"powershell", "-command"}, {"cmd", "/k"}, {"bash", "-c"}, {"sh", "-c"}}

func runShellFilter(filter ShellFilter, settings map[string]interface{}, absoluteLocation string) error {
	var err error = nil
	if len(settings) == 0 {
		err = executeCommand(
			filter.Definition.Command, filter.Arguments, absoluteLocation,
			GetAbsoluteWorkingDirectory())
	} else {
		jsonSettings, _ := json.Marshal(settings)
		err = executeCommand(
			filter.Definition.Command,
			append([]string{string(jsonSettings)}, filter.Arguments...),
			absoluteLocation, GetAbsoluteWorkingDirectory())
	}
	return err
}

func executeCommand(command string, args []string, absoluteLocation string, workingDir string) error {
	for i, arg := range args {
		args[i] = strconv.Quote(arg)
	}
	joined := strings.Join(append([]string{command}, args...), " ")
	Logger.Debugf("Executing command: %s", joined)
	shell, arg, err := findShell()
	if err != nil {
		return err
	}
	cmd := exec.Command(shell, arg, joined)
	cmd.Dir = workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "FILTER_DIR="+absoluteLocation)

	err = cmd.Run()

	if err != nil {
		return wrapError("Failed to run shell script", err)
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
	return "", "", errors.New("unable to find a valid shell")
}

func checkShellRequirements() error {
	shell, _, err := findShell()
	if err == nil {
		Logger.Debugf("Using shell: %s", shell)
	}
	return err
}
