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

const nodeJSFilterName = "nodejs"

type NodeJSFilter struct {
	Filter

	Script string `json:"script,omitempty"`
}

func NodeJSFilterFromObject(obj map[string]interface{}) *NodeJSFilter {
	filter := &NodeJSFilter{Filter: *FilterFromObject(obj)}

	script, ok := obj["script"].(string)
	if !ok {
		Logger.Fatalf("Could filter %q", filter.GetFriendlyName())
	}
	filter.Script = script
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
	return runNodeJSFilter(*f, f.Settings, absoluteLocation)
}

func (f *NodeJSFilter) InstallDependencies(parent *RemoteFilter) error {
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
	err = installNodeJSFilter(*f, filepath.Dir(scriptPath))
	if err != nil {
		return wrapError(fmt.Sprintf(
			"Couldn't install filter dependencies %s",
			f.GetFriendlyName()), err)
	}

	Logger.Infof("Dependencies for %s installed successfully", f.GetFriendlyName())
	return nil
}

func (f *NodeJSFilter) Check() error {
	return checkNodeJSRequirements()
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

func runNodeJSFilter(filter NodeJSFilter, settings map[string]interface{}, absoluteLocation string) error {
	if len(settings) == 0 {
		err := RunSubProcess("node", append([]string{absoluteLocation + string(os.PathSeparator) + filter.Script}, filter.Arguments...), absoluteLocation, GetAbsoluteWorkingDirectory())
		if err != nil {
			return wrapError("Failed to run NodeJS script", err)
		}
	} else {
		jsonSettings, _ := json.Marshal(settings)
		err := RunSubProcess("node", append([]string{absoluteLocation + string(os.PathSeparator) + filter.Script, string(jsonSettings)}, filter.Arguments...), absoluteLocation, GetAbsoluteWorkingDirectory())
		if err != nil {
			return wrapError("Failed to run NodeJS script", err)
		}
	}
	return nil
}

func installNodeJSFilter(filter NodeJSFilter, filterPath string) error {
	if hasPackageJson(filterPath) {
		Logger.Info("Installing npm dependencies...")
		err := RunSubProcess("npm", []string{"i", "--no-fund", "--no-audit"}, filterPath, filterPath)
		if err != nil {
			return wrapError("Failed to run npm", err)
		}
	}
	return nil
}

func hasPackageJson(filterPath string) bool {
	_, err := os.Stat(path.Join(filterPath, "package.json"))
	return err == nil
}

func checkNodeJSRequirements() error {
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
