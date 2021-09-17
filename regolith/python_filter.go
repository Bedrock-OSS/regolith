package regolith

import (
	"encoding/json"
	"errors"
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

func runPythonFilter(filter Filter, settings map[string]interface{}, absoluteLocation string) error {
	command := "python"
	if needsVenv(absoluteLocation) {
		venvPath, err := resolveVenvPath(filter, absoluteLocation)
		if err != nil {
			return wrapError("Failed to resolve venv path", err)
		}
		Logger.Debug("Running Python filter using venv: ", venvPath)
		suffix := ""
		if runtime.GOOS == "windows" {
			suffix = ".exe"
		}
		command = path.Join(venvPath, "Scripts/python"+suffix)
	}
	if len(settings) == 0 {
		err := RunSubProcess(command, append([]string{"-u", absoluteLocation + string(os.PathSeparator) + filter.Location}, filter.Arguments...), GetAbsoluteWorkingDirectory())
		if err != nil {
			return wrapError("Failed to run Python script", err)
		}
	} else {
		jsonSettings, _ := json.Marshal(settings)
		err := RunSubProcess(command, append([]string{"-u", absoluteLocation + string(os.PathSeparator) + filter.Location, string(jsonSettings)}, filter.Arguments...), GetAbsoluteWorkingDirectory())
		if err != nil {
			return wrapError("Failed to run Python script", err)
		}
	}
	return nil
}

func installPythonFilter(filter Filter, filterPath string) error {
	if needsVenv(filterPath) {
		venvPath, err := resolveVenvPath(filter, filterPath)
		if err != nil {
			return wrapError("Failed to resolve venv path", err)
		}
		Logger.Info("Creating venv...")
		err = RunSubProcess("python", []string{"-m", "venv", venvPath}, "")
		if err != nil {
			return err
		}
		suffix := ""
		if runtime.GOOS == "windows" {
			suffix = ".exe"
		}
		Logger.Info("Installing pip dependencies...")
		err = RunSubProcess(
			path.Join(venvPath, "Scripts/pip"+suffix),
			[]string{"install", "-r", "requirements.txt"}, filterPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func needsVenv(filterPath string) bool {
	stats, err := os.Stat(path.Join(filterPath, "requirements.txt"))
	if err == nil {
		return !stats.IsDir()
	}
	return false
}

func resolveVenvPath(filter Filter, filterPath string) (string, error) {
	resolvedPath, err := filepath.Abs(
		path.Join(".regolith/cache/venvs", strconv.Itoa(filter.VenvSlot)))
	if err != nil {
		return "", wrapError(fmt.Sprintf("VenvSlot %v: Unable to create venv", filter.VenvSlot), err)
	}
	return resolvedPath, nil
}

func checkPythonRequirements() error {
	_, err := exec.LookPath("python")
	if err != nil {
		return errors.New("Python not found. Download and install it from https://www.python.org/downloads/")
	}
	cmd, err := exec.Command("python", "--version").Output()
	if err != nil {
		return wrapError("Python version check failed", err)
	}
	a := strings.TrimPrefix(strings.Trim(string(cmd), " \n\t"), "Python ")
	Logger.Debugf("Found Python version %s", a)
	return nil
}
