package regolith

import (
	"encoding/json"
	"os"
	"os/exec"
	"path"
	"strings"
)

const nodeJSFilterName = "nodejs"

func RegisterNodeJSFilter(filters map[string]filterDefinition) {
	filters[nodeJSFilterName] = filterDefinition{
		filter:              runNodeJSFilter,
		installDependencies: installNodeJSFilter,
		check:               checkNodeJSRequirements,
	}
}

func runNodeJSFilter(filter Filter, settings map[string]interface{}, absoluteLocation string) error {
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

func installNodeJSFilter(filter Filter, filterPath string) error {
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
