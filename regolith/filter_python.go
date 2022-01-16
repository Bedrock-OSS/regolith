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
	"time"
)

const pythonFilterName = "python"

type PythonFilter struct {
	Filter

	Script   string `json:"script,omitempty"`
	VenvSlot int    `json:"venvSlot,omitempty"`
}

func PythonFilterFromObject(obj map[string]interface{}) *PythonFilter {
	filter := &PythonFilter{Filter: *FilterFromObject(obj)}

	script, ok := obj["script"].(string)
	if !ok {
		Logger.Fatalf("Could filter %q", filter.GetFriendlyName())
	}
	filter.Script = script
	filter.VenvSlot, _ = obj["venvSlot"].(int) // default venvSlot is 0
	return filter
}

func (f *PythonFilter) Run(absoluteLocation string) error {
	// Disabled filters are skipped
	if f.Disabled {
		Logger.Infof("Filter '%s' is disabled, skipping.", f.GetFriendlyName())
		return nil
	}
	Logger.Infof("Running filter %s", f.GetFriendlyName())
	start := time.Now()
	defer Logger.Debugf("Executed in %s", time.Since(start))
	return runPythonFilter(*f, f.Settings, absoluteLocation)
}

func (f *PythonFilter) InstallDependencies(parent *RemoteFilter) error {
	installLocation := ""
	// Install dependencies
	if parent != nil {
		installLocation = parent.GetDownloadPath()
	}
	Logger.Infof("Downloading dependencies for %s...", f.GetFriendlyName())
	scriptPath, err := filepath.Abs(filepath.Join(installLocation, f.Script))
	if err != nil {
		return wrapError(fmt.Sprintf(
			"Unable to resolve path of %s script",
			f.GetFriendlyName()), err)
	}
	err = installPythonFilter(*f, filepath.Dir(scriptPath))
	if err != nil {
		return wrapError(fmt.Sprintf(
			"Couldn't install filter dependencies %s",
			f.GetFriendlyName()), err)
	}

	Logger.Infof("Dependencies for %s installed successfully", f.GetFriendlyName())
	return nil
}

func (f *PythonFilter) Check() error {
	return checkPythonRequirements()
}

func (f *PythonFilter) CopyArguments(parent *RemoteFilter) {
	f.Arguments = parent.Arguments
	f.Settings = parent.Settings
	f.VenvSlot = parent.VenvSlot
}

func (f *PythonFilter) GetFriendlyName() string {
	if f.Name != "" {
		return f.Name
	}
	return "Unnamed Python filter"
}

func runPythonFilter(
	filter PythonFilter, settings map[string]interface{}, absoluteLocation string,
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

func installPythonFilter(filter PythonFilter, filterPath string) error {
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
	Logger.Info(filepath.Join(filterPath, "requirements.txt"))
	stats, err := os.Stat(filepath.Join(filterPath, "requirements.txt"))
	if err == nil {
		return !stats.IsDir()
	}
	return false
}

func resolveVenvPath(filter PythonFilter) (string, error) {
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
