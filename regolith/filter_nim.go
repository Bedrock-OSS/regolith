package regolith

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type NimFilterDefinition struct {
	FilterDefinition
	Script string `json:"script,omitempty"`

	// Requirements is an optional path to the folder with the nimble file. If not specified
	// the parent of the script path is used instead.
	Requirements string `json:"requirements,omitempty"`
}

type NimFilter struct {
	Filter
	Definition NimFilterDefinition `json:"-"`
}

func NimFilterDefinitionFromObject(
	id string, obj map[string]interface{},
) (*NimFilterDefinition, error) {
	filter := &NimFilterDefinition{FilterDefinition: *FilterDefinitionFromObject(id)}
	scriptObj, ok := obj["script"]
	if !ok {
		return nil, WrappedErrorf(jsonPropertyMissingError, "script")
	}
	script, ok := scriptObj.(string)
	if !ok {
		return nil, WrappedErrorf(
			jsonPropertyTypeError, "script", "string")
	}
	filter.Script = script

	requirementsObj, ok := obj["requirements"]
	if ok {
		requirements, ok := requirementsObj.(string)
		if !ok {
			return nil, WrappedErrorf(
				jsonPropertyTypeError, "requirements", "string")
		}
		filter.Requirements = requirements
	}
	return filter, nil
}

func (f *NimFilter) run(context RunContext) error {
	// Run filter
	if len(f.Settings) == 0 {
		err := RunSubProcess(
			"nim",
			append([]string{
				"-r", "c", "--hints:off", "--warnings:off", "--mm:orc",
				context.AbsoluteLocation + string(os.PathSeparator) + f.Definition.Script},
				f.Arguments...,
			),
			context.AbsoluteLocation,
			GetAbsoluteWorkingDirectory(context.DotRegolithPath),
			ShortFilterName(f.Id),
		)
		if err != nil {
			return PassError(err)
		}
	} else {
		jsonSettings, _ := json.Marshal(f.Settings)
		err := RunSubProcess(
			"nim",
			append([]string{
				"-r", "c", "--hints:off", "--warnings:off", "--mm:orc",
				context.AbsoluteLocation + string(os.PathSeparator) +
					f.Definition.Script,
				string(jsonSettings)},
				f.Arguments...),
			context.AbsoluteLocation,
			GetAbsoluteWorkingDirectory(context.DotRegolithPath),
			ShortFilterName(f.Id),
		)
		if err != nil {
			return PassError(err)
		}
	}
	return nil
}

func (f *NimFilter) Run(context RunContext) (bool, error) {
	if err := f.run(context); err != nil {
		return false, PassError(err)
	}
	return context.IsInterrupted(), nil
}

func (f *NimFilterDefinition) CreateFilterRunner(runConfiguration map[string]interface{}) (FilterRunner, error) {
	basicFilter, err := filterFromObject(runConfiguration)
	if err != nil {
		return nil, WrapError(err, filterFromObjectError)
	}
	filter := &NimFilter{
		Filter:     *basicFilter,
		Definition: *f,
	}
	return filter, nil
}

func (f *NimFilterDefinition) InstallDependencies(
	parent *RemoteFilterDefinition, dotRegolithPath string,
) error {
	installLocation := ""
	// Install dependencies
	if parent != nil {
		installLocation = parent.GetDownloadPath(dotRegolithPath)
	}
	Logger.Infof("Downloading dependencies for %s...", f.Id)
	var requirementsPath string
	if f.Requirements == "" {
		Logger.Debug("Unable to find the requirements property, using the parent folder instead.")
		// Deduce the path from the script path
		joinedPath := filepath.Join(installLocation, f.Script)
		scriptPath, err := filepath.Abs(joinedPath)
		if err != nil {
			return WrapErrorf(err, filepathAbsError, joinedPath)
		}
		requirementsPath = filepath.Dir(scriptPath)
	} else {
		joinedPath := filepath.Join(installLocation, f.Requirements)
		scriptPath, err := filepath.Abs(joinedPath)
		if err != nil {
			return WrapErrorf(err, filepathAbsError, joinedPath)
		}
		requirementsPath = scriptPath
	}
	Logger.Debugf("Installing dependencies using nimble in %s", requirementsPath)
	if hasNimble(requirementsPath) {
		Logger.Info("Installing nim dependencies...")
		err := RunSubProcess(
			"nimble", []string{"install", "-d", "-y"}, requirementsPath, requirementsPath, ShortFilterName(f.Id))
		if err != nil {
			return WrapErrorf(
				err, "Failed to run nimble to install dependencies of a filter.\n"+
					"Filter name: %s.",
				f.Id)
		}
	}
	Logger.Infof("Dependencies for %s installed successfully", f.Id)
	return nil
}

func (f *NimFilterDefinition) Check(context RunContext) error {
	_, err := exec.LookPath("nim")
	if err != nil {
		return WrapError(
			err,
			"Nim not found, download and install it from"+
				" https://nim-lang.org/")
	}
	cmd, err := exec.Command("nim", "--version").Output()
	if err != nil {
		return WrapError(err, "Failed to check Nim version.")
	}
	a := strings.TrimPrefix(strings.Trim(string(cmd), " \n\t"), "v")
	Logger.Debugf("Found Nim version %s.", a)
	return nil
}

func (f *NimFilter) Check(context RunContext) error {
	return f.Definition.Check(context)
}

func hasNimble(filterPath string) bool {
	nimble := false
	filepath.Walk(filterPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".nimble" {
			nimble = true
		}
		return nil
	})
	return nimble
}
