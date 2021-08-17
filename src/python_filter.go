package src

import (
	"os/exec"
	"strings"
)

const pythonFilterName = "python"

func RegisterPythonFilter(filters map[string]filterDefinition) {
	filters[pythonFilterName] = filterDefinition{
		filter: runPythonFilter,
		check:  checkPythonRequirements,
	}
}

func runPythonFilter(filter Filter, absoluteLocation string) {
	RunSubProcess("python", append([]string{"-u", absoluteLocation}, filter.Arguments...))
}

func checkPythonRequirements() {
	_, err := exec.LookPath("python")
	if err != nil {
		Logger.Fatal("Python not found")
	}
	cmd, _ := exec.Command("python", "--version").Output()
	a := strings.TrimPrefix(strings.Trim(string(cmd), " \n\t"), "Python ")
	Logger.Debugf("Found Python version %s", a)
}
