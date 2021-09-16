package src

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const shellFilterName = "shell"

var shells = [][]string{{"powershell", "-command"}, {"cmd", "/k"}, {"bash", "-c"}, {"sh", "-c"}}

func RegisterShellFilter(filters map[string]filterDefinition) {
	filters[shellFilterName] = filterDefinition{
		filter:  runShellFilter,
		install: func(filter Filter, path string) error { return nil },
		check:   checkShellRequirements,
	}
}

func runShellFilter(filter Filter, settings map[string]interface{}, absoluteLocation string) error {
	var err error = nil
	if len(settings) == 0 {
		err = executeCommand(filter.Command, filter.Arguments, absoluteLocation, GetAbsoluteWorkingDirectory())
	} else {
		jsonSettings, _ := json.Marshal(settings)
		err = executeCommand(filter.Command, append([]string{string(jsonSettings)}, filter.Arguments...), absoluteLocation, GetAbsoluteWorkingDirectory())
	}
	return err
}

func executeCommand(command string, args []string, absoluteLocation string, workingDir string) error {
	for i, arg := range args {
		args[i] = strconv.Quote(arg)
	}
	joined := strings.Join(append([]string{command}, args...), " ")
	Logger.Debug(joined)
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
	return "", "", errors.New("Unable to find a valid shell")
}

func checkShellRequirements() error {
	shell, _, err := findShell()
	if err == nil {
		Logger.Debugf("Using shell: %s", shell)
	}
	return err
}
