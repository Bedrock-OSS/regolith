package regolith

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const nimFilterName = "nim"

type NimFilter struct {
	Filter

	Script string `json:"script,omitempty"`
}

func NimFilterFromObject(obj map[string]interface{}) *NimFilter {
	filter := &NimFilter{Filter: *FilterFromObject(obj)}

	script, ok := obj["script"].(string)
	if !ok {
		Logger.Fatalf("Could filter %q", filter.GetFriendlyName())
	}
	filter.Script = script
	return filter
}

func (f *NimFilter) Run(absoluteLocation string) error {
	// Disabled filters are skipped
	if f.Disabled {
		Logger.Infof("Filter '%s' is disabled, skipping.", f.GetFriendlyName())
		return nil
	}
	Logger.Infof("Running filter %s", f.GetFriendlyName())
	start := time.Now()
	defer Logger.Debugf("Executed in %s", time.Since(start))
	return runNimFilter(*f, f.Settings, absoluteLocation)
}

func (f *NimFilter) InstallDependencies(parent *RemoteFilter) error {
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
	err = installNimFilter(*f, filepath.Dir(scriptPath))
	if err != nil {
		return wrapError(fmt.Sprintf(
			"Couldn't install filter dependencies %s",
			f.GetFriendlyName()), err)
	}

	Logger.Infof("Dependencies for %s installed successfully", f.GetFriendlyName())
	return nil
}

func (f *NimFilter) Check() error {
	return checkNimRequirements()
}

func (f *NimFilter) CopyArguments(parent *RemoteFilter) {
	f.Arguments = parent.Arguments
	f.Settings = parent.Settings
}

func (f *NimFilter) GetFriendlyName() string {
	if f.Name != "" {
		return f.Name
	}
	return "Unnamed Nim filter"
}

func runNimFilter(filter NimFilter, settings map[string]interface{}, absoluteLocation string) error {
	if len(settings) == 0 {
		err := RunSubProcess("nim", append([]string{"-r", "c", "--hints:off", "--warnings:off", absoluteLocation + string(os.PathSeparator) + filter.Script}, filter.Arguments...), absoluteLocation, GetAbsoluteWorkingDirectory())
		if err != nil {
			return wrapError("Failed to run Nim script", err)
		}
	} else {
		jsonSettings, _ := json.Marshal(settings)
		err := RunSubProcess("nim", append([]string{"-r", "c", "--hints:off", "--warnings:off", absoluteLocation + string(os.PathSeparator) + filter.Script, string(jsonSettings)}, filter.Arguments...), absoluteLocation, GetAbsoluteWorkingDirectory())
		if err != nil {
			return wrapError("Failed to run Nim script", err)
		}
	}
	return nil
}

func installNimFilter(filter NimFilter, filterPath string) error {
	if hasNimble(filterPath) {
		Logger.Info("Installing nim dependencies...")
		err := RunSubProcess("nimble", []string{"install"}, filterPath, filterPath)
		if err != nil {
			return wrapError("Failed to run nimble", err)
		}
	}
	return nil
}

func hasNimble(filterPath string) bool {
	nimble := false
	filepath.Walk(filterPath, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) == ".nimble" {
			nimble = true
		}
		return nil
	})
	return nimble
}

func checkNimRequirements() error {
	_, err := exec.LookPath("nim")
	if err != nil {
		Logger.Fatal("Nim not found. Download and install it from https://nim-lang.org/")
	}
	cmd, err := exec.Command("nim", "--version").Output()
	if err != nil {
		return wrapError("Failed to check Nim version", err)
	}
	a := strings.TrimPrefix(strings.Trim(string(cmd), " \n\t"), "v")
	Logger.Debugf("Found Nim version %s", a)
	return nil
}
