package regolith

import (
	"encoding/json"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
)

type NodeJSFilterDefinition struct {
	FilterDefinition
	Script string `json:"script,omitempty"`

	// Requirements is an optional path to the folder with the package.json file.
	// If not specified the parent of thhe script path is used instead.
	Requirements string `json:"requirements,omitempty"`
}

type NodeJSFilter struct {
	Filter
	Definition NodeJSFilterDefinition `json:"-"`
}

func NodeJSFilterDefinitionFromObject(id string, obj map[string]interface{}) (*NodeJSFilterDefinition, error) {
	filter := &NodeJSFilterDefinition{FilterDefinition: *FilterDefinitionFromObject(id)}
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

func (f *NodeJSFilter) run(context RunContext) error {
	// Run filter
	if len(f.Settings) == 0 {
		err := RunSubProcess(
			"node",
			append([]string{
				context.AbsoluteLocation + string(os.PathSeparator) +
					f.Definition.Script},
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
			"node",
			append([]string{
				context.AbsoluteLocation + string(os.PathSeparator) +
					f.Definition.Script,
				string(jsonSettings)}, f.Arguments...),
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

func (f *NodeJSFilter) Run(context RunContext) (bool, error) {
	if err := f.run(context); err != nil {
		return false, PassError(err)
	}
	return context.IsInterrupted(), nil
}

func (f *NodeJSFilterDefinition) CreateFilterRunner(runConfiguration map[string]interface{}) (FilterRunner, error) {
	basicFilter, err := filterFromObject(runConfiguration)
	if err != nil {
		return nil, WrapError(err, filterFromObjectError)
	}
	filter := &NodeJSFilter{
		Filter:     *basicFilter,
		Definition: *f,
	}
	return filter, nil
}

func (f *NodeJSFilterDefinition) InstallDependencies(parent *RemoteFilterDefinition, dotRegolithPath string) error {
	installLocation := ""
	// Install dependencies
	if parent != nil {
		installLocation = parent.GetDownloadPath(dotRegolithPath)
	}
	Logger.Infof("Downloading dependencies for %s...", f.Id)
	var requirementsPath string
	if f.Requirements == "" {
		// Deduce the path from the script path
		joinedPath := filepath.Join(installLocation, f.Script)
		scriptPath, err := filepath.Abs(joinedPath)
		if err != nil {
			return WrapErrorf(err, filepathAbsError, joinedPath)
		}
		requirementsPath = filepath.Dir(scriptPath)
	} else {
		joinedPath := filepath.Join(installLocation, f.Requirements)
		installPath, err := filepath.Abs(joinedPath)
		if err != nil {
			return WrapErrorf(err, filepathAbsError, joinedPath)
		}
		requirementsPath = installPath
	}
	if hasPackageJson(requirementsPath) {
		Logger.Info("Installing npm dependencies...")
		err := RunSubProcess("npm", []string{"i", "--no-fund", "--no-audit"}, requirementsPath, requirementsPath, ShortFilterName(f.Id))
		if err != nil {
			return WrapErrorf(
				err, "Failed to run npm and install dependencies."+
					"\nFilter name: %s", f.Id)
		}
	}
	Logger.Infof("Dependencies for %s installed successfully", f.Id)
	return nil
}

func (f *NodeJSFilterDefinition) Check(context RunContext) error {
	_, err := exec.LookPath("node")
	if err != nil {
		return WrapError(
			err, "NodeJS not found, download and install it from"+
				" https://nodejs.org/en/")
	}
	cmd, err := exec.Command("node", "--version").Output()
	if err != nil {
		return WrapError(err, "Failed to check NodeJS version")
	}
	a := strings.TrimPrefix(strings.Trim(string(cmd), " \n\t"), "v")
	Logger.Debugf("Found NodeJS version %s", a)
	return nil
}

func (f *NodeJSFilter) Check(context RunContext) error {
	return f.Definition.Check(context)
}

func hasPackageJson(filterPath string) bool {
	_, err := os.Stat(path.Join(filterPath, "package.json"))
	return err == nil
}
