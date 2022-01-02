package regolith

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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

func runPythonFilter(
	filter Filter, settings map[string]interface{}, absoluteLocation string,
) error {
	// command is a list of strings that can possibly run python (it's python3
	// on some OSs)
	command := []string{"python", "python3"}
	scriptPath := filepath.Join(absoluteLocation, filter.Script)

	if needsVenv(filepath.Dir(scriptPath)) {
		venvPath, err := resolveVenvPath(filter)
		if err != nil {
			return wrapError("Failed to resolve venv path", err)
		}
		Logger.Debug("Running Python filter using venv: ", venvPath)
		command = []string{
			filepath.Join(venvPath, venvScriptsPath, "python"+exeSuffix)}
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
	var err error
	for _, c := range command {
		err = RunSubProcess(
			c, args, absoluteLocation, GetAbsoluteWorkingDirectory())
		if err == nil {
			return nil
		}
	}
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
		// it's sometimes python3 on some OSs
		for _, c := range []string{"python", "python3"} {
			err = RunSubProcess(
				c, []string{"-m", "venv", venvPath}, filterPath, "")
			if err == nil {
				break
			}
		}
		if err != nil {
			return err
		}
		Logger.Info("Installing pip dependencies...")
		err = RunSubProcess(
			filepath.Join(venvPath, venvScriptsPath, "pip"+exeSuffix),
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
	python := ""
	var err error
	for _, c := range []string{"python", "python3"} {
		_, err = exec.LookPath(c)
		if err == nil {
			python = c
			break
		}
	}
	if err != nil {
		return errors.New("Python not found. Download and install it from https://www.python.org/downloads/")
	}
	cmd, err := exec.Command(python, "--version").Output()
	if err != nil {
		return wrapError("Python version check failed", err)
	}
	a := strings.TrimPrefix(strings.Trim(string(cmd), " \n\t"), "Python ")
	Logger.Debugf("Found Python version %s", a)
	return nil
}
