package regolith

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const nimFilterName = "nim"

func RegisterNimFilter(filters map[string]filterDefinition) {
	filters[nimFilterName] = filterDefinition{
		filter:              runNimFilter,
		installDependencies: installNimFilter,
		check:               checkNimRequirements,
	}
}

func runNimFilter(filter Filter, settings map[string]interface{}, absoluteLocation string) error {
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

func installNimFilter(filter Filter, filterPath string) error {
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
