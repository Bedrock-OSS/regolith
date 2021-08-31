package src

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
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
	if needsVenv(absoluteLocation) {
		venvPath := resolveVenvPath(filter, absoluteLocation)
		Logger.Debug("Running Python filter using venv: ", venvPath)
		suffix := ""
		if runtime.GOOS == "windows" {
			suffix = ".exe"
		}
		command = path.Join(venvPath, "Scripts/python"+suffix)
	}
	if len(settings) == 0 {
		RunSubProcess(command, append([]string{"-u", absoluteLocation + string(os.PathSeparator) + filter.Location}, filter.Arguments...), GetAbsoluteWorkingDirectory())
	} else {
		jsonSettings, _ := json.Marshal(settings)
		RunSubProcess(command, append([]string{"-u", absoluteLocation + string(os.PathSeparator) + filter.Location, string(jsonSettings)}, filter.Arguments...), GetAbsoluteWorkingDirectory())
	}
}

func installPythonFilter(filter Filter, filterPath string) {
	if needsVenv(filterPath) {
		venvPath := resolveVenvPath(filter, filterPath)
		Logger.Info("Creating venv...")
		RunSubProcess("python", []string{"-m", "venv", venvPath}, "")
		suffix := ""
		if runtime.GOOS == "windows" {
			suffix = ".exe"
		}
		Logger.Info("Installing pip dependencies...")
		RunSubProcess(
			path.Join(venvPath, "Scripts/pip"+suffix),
			[]string{"install", "-r", "requirements.txt"}, filterPath)
	}
}

func needsVenv(filterPath string) bool {
	stats, err := os.Stat(path.Join(filterPath, "requirements.txt"))
	if err == nil {
		return !stats.IsDir()
	}
	return false
}

func resolveVenvPath(filter Filter, filterPath string) string {
	resolvedPath, err := filepath.Abs(
		path.Join(".regolith/cache/venvs", strconv.Itoa(filter.VenvSlot)))
	if err != nil {
		Logger.Fatal(fmt.Sprintf("VenvSlot %v: Unable to create venv", filter.VenvSlot))
	}
	return resolvedPath
}

func checkPythonRequirements() {
	_, err := exec.LookPath("python")
	if err != nil {
		Logger.Fatal("Python not found. Download and install it from https://www.python.org/downloads/")
	}
	cmd, _ := exec.Command("python", "--version").Output()
	a := strings.TrimPrefix(strings.Trim(string(cmd), " \n\t"), "Python ")
	Logger.Debugf("Found Python version %s", a)
}
