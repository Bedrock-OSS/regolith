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

	// Run filter
	if len(f.Settings) == 0 {
		err := RunSubProcess(
			"nim",
			append([]string{
				"-r", "c", "--hints:off", "--warnings:off",
				absoluteLocation + string(os.PathSeparator) + f.Script},
				f.Arguments...,
			),
			absoluteLocation,
			GetAbsoluteWorkingDirectory(),
		)
		if err != nil {
			return wrapError("Failed to run Nim script", err)
		}
	} else {
		jsonSettings, _ := json.Marshal(f.Settings)
		err := RunSubProcess(
			"nim",
			append([]string{
				"-r", "c", "--hints:off", "--warnings:off",
				absoluteLocation + string(os.PathSeparator) + f.Script,
				string(jsonSettings)},
				f.Arguments...),
			absoluteLocation,
			GetAbsoluteWorkingDirectory(),
		)
		if err != nil {
			return wrapError("Failed to run Nim script", err)
		}
	}
	return nil
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
	filterPath := filepath.Dir(scriptPath)
	if hasNimble(filterPath) {
		Logger.Info("Installing nim dependencies...")
		err := RunSubProcess(
			"nimble", []string{"install"}, filterPath, filterPath)
		if err != nil {
			return wrapError(
				fmt.Sprintf(
					"Failed to run nimble to install dependencies of %s",
					f.GetFriendlyName(),
				),
				err,
			)
		}
	}
	Logger.Infof("Dependencies for %s installed successfully", f.GetFriendlyName())
	return nil
}

func (f *NimFilter) Check() error {
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
