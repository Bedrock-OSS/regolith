package src

import (
	"encoding/json"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
)

const pythonFilterName = "python"

func RegisterPythonFilter(filters map[string]filterDefinition) {
	filters[pythonFilterName] = filterDefinition{
		filter:  runPythonFilter,
		install: installPythonFilter,
		check:   checkPythonRequirements,
	}
}

func runPythonFilter(filter Filter, settings map[string]interface{}, absoluteLocation string) {
	command := "python"
	dir := path.Dir(absoluteLocation)
	if needsVenv(dir) {
		suffix := ""
		if runtime.GOOS == "windows" {
			suffix = ".exe"
		}
		command = dir + "/venv/Scripts/python" + suffix
	}
	if len(settings) == 0 {
		RunSubProcess(command, append([]string{"-u", absoluteLocation}, filter.Arguments...), GetAbsoluteWorkingDirectory())
	} else {
		jsonSettings, _ := json.Marshal(settings)
		RunSubProcess(command, append([]string{"-u", absoluteLocation, string(jsonSettings)}, filter.Arguments...), GetAbsoluteWorkingDirectory())
	}
}

func installPythonFilter(filter Filter, filterPath string) {
	if needsVenv(filterPath) {
		Logger.Info("Creating venv...")
		RunSubProcess("python", []string{"-m", "venv", "venv"}, filterPath)
		suffix := ""
		if runtime.GOOS == "windows" {
			suffix = ".exe"
		}
		Logger.Info("Installing pip dependencies...")
		RunSubProcess("venv/Scripts/pip"+suffix, []string{"install", "-r", "requirements.txt"}, filterPath)
	}
}

func needsVenv(filterPath string) bool {
	_, err := os.Stat(path.Join(filterPath, "requirements.txt"))
	return err == nil
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
