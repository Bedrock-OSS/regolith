package regolith

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Bedrock-OSS/go-burrito/burrito"
)

type PythonFilterDefinition struct {
	FilterDefinition
	Script   string `json:"script,omitempty"`
	VenvSlot int    `json:"venvSlot,omitempty"`

	// Requirements is an optional path to the file with the requirements
	// (usually requirements.txt). If not specified, the parent path of the
	// script is used.
	Requirements string `json:"requirements,omitempty"`
}

type PythonFilter struct {
	Filter
	Definition PythonFilterDefinition `json:"-"`
}

func PythonFilterDefinitionFromObject(id string, obj map[string]interface{}) (*PythonFilterDefinition, error) {
	filter := &PythonFilterDefinition{FilterDefinition: *FilterDefinitionFromObject(id)}
	scripObj, ok := obj["script"]
	if !ok {
		return nil, burrito.WrappedErrorf(jsonPropertyMissingError, "script")
	}
	script, ok := scripObj.(string)
	if !ok {
		return nil, burrito.WrappedErrorf(jsonPropertyTypeError, "script", "string")
	}
	filter.Script = script
	filter.VenvSlot, _ = obj["venvSlot"].(int) // default venvSlot is 0

	requirementsObj, ok := obj["requirements"]
	if ok {
		requirements, ok := requirementsObj.(string)
		if !ok {
			return nil, burrito.WrappedErrorf(
				jsonPropertyTypeError, "requirements", "string")
		}
		filter.Requirements = requirements
	}
	return filter, nil
}

func (f *PythonFilter) run(context RunContext) error {
	// Run filter
	pythonCommand, err := findPython()
	if err != nil {
		return burrito.PassError(err)
	}
	scriptPath := filepath.Join(context.AbsoluteLocation, f.Definition.Script)
	filterPath := filepath.Dir(scriptPath)
	var requirementsFile string
	if f.Definition.Requirements == "" {
		requirementsFile = filepath.Join(filterPath, "requirements.txt")
	} else {
		requirementsFile = filepath.Join(
			context.AbsoluteLocation, f.Definition.Requirements)
		requirementsFile, err = filepath.Abs(requirementsFile)
		if err != nil {
			return burrito.WrapErrorf(err, filepathAbsError, requirementsFile)
		}
	}

	if needsVenv(requirementsFile) {
		venvPath, err := f.Definition.resolveVenvPath(context.DotRegolithPath)
		if err != nil {
			return burrito.WrapError(err, "Failed to resolve venv path.")
		}
		Logger.Debug("Running Python filter using venv: ", venvPath)
		pythonCommand = filepath.Join(
			venvPath, venvScriptsPath, "python"+exeSuffix)
	}
	var args []string
	if len(f.Settings) == 0 {
		args = append([]string{"-u", scriptPath}, f.Arguments...)
	} else {
		jsonSettings, _ := json.Marshal(f.Settings)
		args = append(
			[]string{"-u", scriptPath, string(jsonSettings)},
			f.Arguments...,
		)
	}
	err = RunSubProcess(
		pythonCommand, args, context.AbsoluteLocation,
		GetAbsoluteWorkingDirectory(context.DotRegolithPath),
		ShortFilterName(f.Id))
	if err != nil {
		return burrito.WrapError(err, "Failed to run Python script.")
	}
	return nil
}

func (f *PythonFilter) Run(context RunContext) (bool, error) {
	if err := f.run(context); err != nil {
		return false, burrito.PassError(err)
	}
	return context.IsInterrupted(), nil
}

func (f *PythonFilterDefinition) CreateFilterRunner(runConfiguration map[string]interface{}) (FilterRunner, error) {
	basicFilter, err := filterFromObject(runConfiguration)
	if err != nil {
		return nil, burrito.WrapError(err, filterFromObjectError)
	}
	filter := &PythonFilter{
		Filter:     *basicFilter,
		Definition: *f,
	}
	return filter, nil
}

func (f *PythonFilterDefinition) InstallDependencies(
	parent *RemoteFilterDefinition, dotRegolithPath string,
) error {
	installLocation := ""
	// Install dependencies
	if parent != nil {
		installLocation = parent.GetDownloadPath(dotRegolithPath)
	}
	Logger.Infof("Downloading dependencies for %s...", f.Id)
	joinedPath := filepath.Join(installLocation, f.Script)
	scriptPath, err := filepath.Abs(joinedPath)
	if err != nil {
		return burrito.WrapErrorf(err, filepathAbsError, joinedPath)
	}

	// Install the filter dependencies
	filterPath := filepath.Dir(scriptPath)
	var requirementsFile string
	if f.Requirements == "" {
		requirementsFile = filepath.Join(filterPath, "requirements.txt")
	} else {
		requirementsFile = filepath.Join(
			installLocation, f.Requirements)
		requirementsFile, err = filepath.Abs(requirementsFile)
		if err != nil {
			return burrito.WrapErrorf(err, filepathAbsError, requirementsFile)
		}
	}
	if needsVenv(requirementsFile) {
		venvPath, err := f.resolveVenvPath(dotRegolithPath)
		if err != nil {
			return burrito.WrapError(err, "Failed to resolve venv path.")
		}
		Logger.Info("Creating venv...")
		pythonCommand, err := findPython()
		if err != nil {
			return burrito.PassError(err)
		}
		// Create the "venv"
		err = RunSubProcess(
			pythonCommand, []string{"-m", "venv", venvPath}, filterPath, "", ShortFilterName(f.Id))
		if err != nil {
			return burrito.WrapError(err, "Failed to create venv.")
		}
		// Update pip of the venv
		venvPythonCommand := filepath.Join(
			venvPath, venvScriptsPath, "python"+exeSuffix)
		err = RunSubProcess(
			venvPythonCommand,
			[]string{"-m", "pip", "install", "--upgrade", "pip"},
			filterPath, "", ShortFilterName(f.Id))
		if err != nil {
			Logger.Warn("Failed to upgrade pip in venv.")
		}
		// Install the dependencies
		Logger.Info("Installing pip dependencies...")
		requirementsFolder := filepath.Dir(requirementsFile)
		err = RunSubProcess(
			filepath.Join(venvPath, venvScriptsPath, "pip"+exeSuffix),
			[]string{"install", "-r", filepath.Base(requirementsFile)}, requirementsFolder,
			requirementsFolder, ShortFilterName(f.Id))
		if err != nil {
			return burrito.WrapErrorf(
				err, "Couldn't run Pip to install dependencies of %s",
				f.Id,
			)
		}
	}
	Logger.Infof("Dependencies for %s installed successfully.", f.Id)
	return nil
}

func (f *PythonFilterDefinition) InstallRuntime() error {
	return burrito.WrappedErrorf(
		"Python filter type does not support installing runtimes.")
}

func (f *PythonFilterDefinition) Check(context RunContext) error {
	pythonCommand, err := findPython()
	if err != nil {
		return burrito.PassError(err)
	}
	cmd, err := exec.Command(pythonCommand, "--version").Output()
	if err != nil {
		return burrito.WrapError(err, "Python version check failed.")
	}
	a := strings.TrimPrefix(strings.Trim(string(cmd), " \n\t"), "Python ")
	Logger.Debugf("Found Python version %s", a)
	return nil
}

func (f *PythonFilter) Check(context RunContext) error {
	return f.Definition.Check(context)
}

func (f *PythonFilter) CopyArguments(parent *RemoteFilter) {
	f.Arguments = append(f.Arguments, parent.Arguments...)
	f.Settings = parent.Settings
	if f.When == "" {
		f.When = parent.When
	}
	f.Definition.VenvSlot = parent.Definition.VenvSlot
}

func (f *PythonFilterDefinition) resolveVenvPath(dotRegolithPath string) (string, error) {
	resolvedPath, err := filepath.Abs(
		filepath.Join(filepath.Join(dotRegolithPath, "cache/venvs"), strconv.Itoa(f.VenvSlot)))
	if err != nil {
		return "", burrito.WrapErrorf(
			err, "Unable to create venv for VenvSlot %v.", f.VenvSlot)
	}
	return resolvedPath, nil
}

func needsVenv(requirementsFilePath string) bool {
	stats, err := os.Stat(requirementsFilePath)
	if err == nil {
		return !stats.IsDir()
	}
	return false
}

func findPython() (string, error) {
	var err error
	for _, c := range pythonExeNames {
		_, err = exec.LookPath(c)
		if err == nil {
			return c, nil
		}
	}
	return "", burrito.WrappedError(
		"Python not found, download and install it from " +
			"https://www.python.org/downloads/")
}
