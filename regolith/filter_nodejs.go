package regolith

import (
	"encoding/json"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"time"
)

type NodeJSFilterDefinition struct {
	FilterDefinition
	Script string `json:"script,omitempty"`
}

type NodeJSFilter struct {
	Filter
	Definition NodeJSFilterDefinition `json:"-"`
}

func NodeJSFilterDefinitionFromObject(id string, obj map[string]interface{}) (*NodeJSFilterDefinition, error) {
	filter := &NodeJSFilterDefinition{FilterDefinition: *FilterDefinitionFromObject(id)}
	script, ok := obj["script"].(string)
	if !ok {
		return nil, WrappedErrorf(
			"Missing \"script\" property in filter definition %q.", filter.Id)
	}
	filter.Script = script
	return filter, nil
}

func (f *NodeJSFilter) Run(absoluteLocation string) error {
	// Disabled filters are skipped
	if f.Disabled {
		Logger.Infof("Filter \"%s\" is disabled, skipping.", f.Id)
		return nil
	}
	Logger.Infof("Running filter %s", f.Id)
	start := time.Now()
	defer Logger.Debugf("Executed in %s", time.Since(start))
	// Run filter
	if len(f.Settings) == 0 {
		err := RunSubProcess(
			"node",
			append([]string{
				absoluteLocation + string(os.PathSeparator) +
					f.Definition.Script},
				f.Arguments...),
			absoluteLocation,
			GetAbsoluteWorkingDirectory(),
		)
		if err != nil {
			return WrapError(err, "failed to run NodeJS script")
		}
	} else {
		jsonSettings, _ := json.Marshal(f.Settings)
		err := RunSubProcess(
			"node",
			append([]string{
				absoluteLocation + string(os.PathSeparator) +
					f.Definition.Script,
				string(jsonSettings)}, f.Arguments...),
			absoluteLocation,
			GetAbsoluteWorkingDirectory(),
		)
		if err != nil {
			return WrapError(err, "failed to run NodeJS script")
		}
	}
	return nil
}

func (f *NodeJSFilterDefinition) CreateFilterRunner(runConfiguration map[string]interface{}) (FilterRunner, error) {
	basicFilter, err := FilterFromObject(runConfiguration)
	if err != nil {
		return nil, WrapError(err, "failed to create Java filter")
	}
	filter := &NodeJSFilter{
		Filter:     *basicFilter,
		Definition: *f,
	}
	return filter, nil
}

func (f *NodeJSFilterDefinition) InstallDependencies(parent *RemoteFilterDefinition) error {
	installLocation := ""
	// Install dependencies
	if parent != nil {
		installLocation = parent.GetDownloadPath()
	}
	Logger.Infof("Downloading dependencies for %s...", f.Id)
	scriptPath, err := filepath.Abs(filepath.Join(installLocation, f.Script))
	if err != nil {
		return WrapErrorf(err, "unable to resolve path of %s script", f.Id)
	}

	filterPath := filepath.Dir(scriptPath)
	if hasPackageJson(filterPath) {
		Logger.Info("Installing npm dependencies...")
		err := RunSubProcess("npm", []string{"i", "--no-fund", "--no-audit"}, filterPath, filterPath)
		if err != nil {
			return WrapErrorf(
				err, "failed to run npm and install dependencies of %s", f.Id)
		}
	}
	Logger.Infof("Dependencies for %s installed successfully", f.Id)
	return nil
}

func (f *NodeJSFilterDefinition) Check() error {
	_, err := exec.LookPath("node")
	if err != nil {
		return WrapError(
			err, "NodeJS not found, download and install it from"+
				" https://nodejs.org/en/")
	}
	cmd, err := exec.Command("node", "--version").Output()
	if err != nil {
		return WrapError(err, "failed to check NodeJS version")
	}
	a := strings.TrimPrefix(strings.Trim(string(cmd), " \n\t"), "v")
	Logger.Debugf("Found NodeJS version %s", a)
	return nil
}

func (f *NodeJSFilter) Check() error {
	return f.Definition.Check()
}

func (f *NodeJSFilter) CopyArguments(parent *RemoteFilter) {
	f.Arguments = parent.Arguments
	f.Settings = parent.Settings
}

func hasPackageJson(filterPath string) bool {
	_, err := os.Stat(path.Join(filterPath, "package.json"))
	return err == nil
}
