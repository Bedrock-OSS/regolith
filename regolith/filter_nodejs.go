package regolith

import (
	"encoding/json"
	"fmt"
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

func NodeJSFilterDefinitionFromObject(id string, obj map[string]interface{}) *NodeJSFilterDefinition {
	filter := &NodeJSFilterDefinition{FilterDefinition: *FilterDefinitionFromObject(id)}
	script, ok := obj["script"].(string)
	if !ok {
		Logger.Fatalf("Could find script in filter defnition %q", filter.Id)
	}
	filter.Script = script
	return filter
}

func NodeJSFilterFromObject(obj map[string]interface{}, definition NodeJSFilterDefinition) *NodeJSFilter {
	filter := &NodeJSFilter{
		Filter:     *FilterFromObject(obj),
		Definition: definition,
	}
	return filter
}

func (f *NodeJSFilter) Run(absoluteLocation string) error {
	// Disabled filters are skipped
	if f.Disabled {
		Logger.Infof("Filter '%s' is disabled, skipping.", f.GetFriendlyName())
		return nil
	}
	Logger.Infof("Running filter %s", f.GetFriendlyName())
	start := time.Now()
	defer Logger.Debugf("Executed in %s", time.Since(start))
	// Run filter
	if len(f.Settings) == 0 {
		err := RunSubProcess(
			"node",
			append([]string{
				absoluteLocation + string(os.PathSeparator) + f.Script},
				f.Arguments...),
			absoluteLocation,
			GetAbsoluteWorkingDirectory(),
		)
		if err != nil {
			return wrapError("Failed to run NodeJS script", err)
		}
	} else {
		jsonSettings, _ := json.Marshal(f.Settings)
		err := RunSubProcess(
			"node",
			append([]string{
				absoluteLocation + string(os.PathSeparator) + f.Script,
				string(jsonSettings)}, f.Arguments...),
			absoluteLocation,
			GetAbsoluteWorkingDirectory(),
		)
		if err != nil {
			return wrapError("Failed to run NodeJS script", err)
		}
	}
	return nil
}

func (f *NodeJSFilterDefinition) CreateFilterRunner(runConfiguration map[string]interface{}) FilterRunner {
	return NodeJSFilterFromObject(runConfiguration, *f)
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
		return wrapError(fmt.Sprintf(
			"Unable to resolve path of %s script",
			f.Id), err)
	}

	filterPath := filepath.Dir(scriptPath)
	if hasPackageJson(filterPath) {
		Logger.Info("Installing npm dependencies...")
		err := RunSubProcess("npm", []string{"i", "--no-fund", "--no-audit"}, filterPath, filterPath)
		if err != nil {
			return wrapError(
				fmt.Sprintf(
					"Failed to run npm and install dependencies of %s",
					f.Id),
				err,
			)
		}
	}
	Logger.Infof("Dependencies for %s installed successfully", f.Id)
	return nil
}

func (f *NodeJSFilterDefinition) Check() error {
	_, err := exec.LookPath("node")
	if err != nil {
		Logger.Fatal("NodeJS not found. Download and install it from https://nodejs.org/en/")
	}
	cmd, err := exec.Command("node", "--version").Output()
	if err != nil {
		return wrapError("Failed to check NodeJS version", err)
	}
	a := strings.TrimPrefix(strings.Trim(string(cmd), " \n\t"), "v")
	Logger.Debugf("Found NodeJS version %s", a)
	return nil
}

func (f *NodeJSFilter) CopyArguments(parent *RemoteFilter) {
	f.Arguments = parent.Arguments
	f.Settings = parent.Settings
}

func (f *NodeJSFilter) GetFriendlyName() string {
	if f.Name != "" {
		return f.Name
	}
	return "Unnamed NodeJS filter"
}

func hasPackageJson(filterPath string) bool {
	_, err := os.Stat(path.Join(filterPath, "package.json"))
	return err == nil
}
