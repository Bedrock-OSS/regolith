package regolith

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

const pythonFilterName = "python"

func RegisterPythonFilter(filters map[string]filterDefinition) {
	filters[pythonFilterName] = filterDefinition{
		filter:              runPythonFilter,
		installDependencies: installPythonFilter,
		check:               checkPythonRequirements,
	}
}

func runPythonFilter(
	filter Filter, settings map[string]interface{}, absoluteLocation string,
) error {
	command := "python"
	scriptPath := filepath.Join(absoluteLocation, filter.Script)

	if needsVenv(filepath.Dir(scriptPath)) {
		venvPath, err := resolveVenvPath(filter)
		if err != nil {
			return wrapError("Failed to resolve venv path", err)
		}
		Logger.Debug("Running Python filter using venv: ", venvPath)
		suffix := ""
		if runtime.GOOS == "windows" {
			suffix = ".exe"
		}
		command = filepath.Join(venvPath, "Scripts/python"+suffix)
	}
	var args []string
	if len(settings) == 0 {
		args = append([]string{"-u", scriptPath}, filter.Arguments...)
	} else {
		jsonSettings, _ := json.Marshal(settings)
		args = append(
			[]string{"-u", scriptPath, string(jsonSettings)},
			filter.Arguments...)
	}
	err := RunSubProcess(
		command, args, absoluteLocation, GetAbsoluteWorkingDirectory())
	if err != nil {
		return wrapError("Failed to run Python script", err)
	}
	return nil
}

func installPythonFilter(filter Filter, filterPath string) error {
	if needsVenv(filterPath) {
		venvPath, err := resolveVenvPath(filter)
		if err != nil {
			return wrapError("Failed to resolve venv path", err)
		}
		Logger.Info("Creating venv...")
		err = RunSubProcess("python", []string{"-m", "venv", venvPath}, filterPath, "")
		if err != nil {
			return err
		}
		suffix := ""
		if runtime.GOOS == "windows" {
			suffix = ".exe"
		}
		Logger.Info("Installing pip dependencies...")
		err = RunSubProcess(
			filepath.Join(venvPath, "Scripts/pip"+suffix),
			[]string{"install", "-r", "requirements.txt"}, filterPath, filterPath)
		if err != nil {
			return err
		}
	}
	return nil
}

func needsVenv(filterPath string) bool {
	stats, err := os.Stat(filepath.Join(filterPath, "requirements.txt"))
	if err == nil {
		return !stats.IsDir()
	}
	return false
}

func resolveVenvPath(filter Filter) (string, error) {
	resolvedPath, err := filepath.Abs(
		filepath.Join(".regolith/cache/venvs", strconv.Itoa(filter.VenvSlot)))
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
