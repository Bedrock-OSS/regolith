package src

import (
	"encoding/json"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

const shellFilterName = "shell"

var shells = [][]string{{"powershell", "-command"}, {"cmd", "/k"}, {"bash", "-c"}, {"sh", "-c"}}

func RegisterShellFilter(filters map[string]filterDefinition) {
	filters[shellFilterName] = filterDefinition{
		filter:  runShellFilter,
		install: func(filter Filter, path string) {},
		check:   checkShellRequirements,
	}
}

func runShellFilter(filter Filter, settings map[string]interface{}, absoluteLocation string) {
	if len(settings) == 0 {
		executeCommand(filter.Command, filter.Arguments, absoluteLocation, GetAbsoluteWorkingDirectory())
	} else {
		jsonSettings, _ := json.Marshal(settings)
		executeCommand(filter.Command, append([]string{string(jsonSettings)}, filter.Arguments...), absoluteLocation, GetAbsoluteWorkingDirectory())
	}
}

func executeCommand(command string, args []string, absoluteLocation string, workingDir string) {
	for i, arg := range args {
		args[i] = strconv.Quote(arg)
	}
	joined := strings.Join(append([]string{command}, args...), " ")
	Logger.Debug(joined)
	shell, arg := findShell()
	cmd := exec.Command(shell, arg, joined)
	cmd.Dir = workingDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "FILTER_DIR="+absoluteLocation)

	err := cmd.Run()

	if err != nil {
		Logger.Fatal(zap.Error(err))
	}
}

func findShell() (string, string) {
	for _, shell := range shells {
		_, err := exec.LookPath(shell[0])
		if err == nil {
			return shell[0], shell[1]
		}
	}
	Logger.Fatal("Unable to find a valid shell")
	return "", ""
}

func checkShellRequirements() {
	shell, _ := findShell()
	Logger.Debugf("Using shell: %s", shell)
}
