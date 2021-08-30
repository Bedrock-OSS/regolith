package src

import (
	"encoding/json"
	"os"
	"os/exec"
	"path"
	"path/filepath"
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

func runPythonFilter(filter Filter, settings map[string]interface{}, absoluteLocation string, profile Profile) {
	command := "python"
	if needsVenv(absoluteLocation) {
		venvPath := resolveVenvPath(profile, absoluteLocation)
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

func installPythonFilter(filter Filter, filterPath string, profile Profile) {
	if needsVenv(filterPath) {
		venvPath := resolveVenvPath(profile, filterPath)
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

func resolveVenvPath(profile Profile, filterPath string) string {
	var resolvedPath string
	var err error
	if profile.VenvPath == "" {
		// No path defined use filterPath
		resolvedPath, err = filepath.Abs(path.Join(filterPath, "venv"))
		if err != nil {
			Logger.Fatal("Unable to resolve filter path: ", filterPath)
		}
		Logger.Debug("Using default venv_path: ", resolvedPath)
	} else if filepath.IsAbs(profile.VenvPath) {
		// path is absolute (don't change)
		Logger.Debug("Using absolute venv_path: ", profile.VenvPath)
		return profile.VenvPath
	} else {
		// non-absolute path put it into .regolith/venvs
		resolvedPath, err = filepath.Abs(
			path.Join(".regolith/venvs", profile.VenvPath))
		if err != nil {
			Logger.Fatal("Unable to resolve venv_path: ", profile.VenvPath)
		}
		Logger.Debug("Using resolved venv_path: ", profile.VenvPath)
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
