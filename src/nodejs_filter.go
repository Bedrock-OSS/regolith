package src

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
		filter:  runNodeJSFilter,
		install: installNodeJSFilter,
		check:   checkNodeJSRequirements,
	}
}

func runNodeJSFilter(filter Filter, settings map[string]interface{}, absoluteLocation string) {
	if len(settings) == 0 {
		RunSubProcess("node", append([]string{absoluteLocation + string(os.PathSeparator) + filter.Location}, filter.Arguments...), GetAbsoluteWorkingDirectory())
	} else {
		jsonSettings, _ := json.Marshal(settings)
		RunSubProcess("node", append([]string{absoluteLocation + string(os.PathSeparator) + filter.Location, string(jsonSettings)}, filter.Arguments...), GetAbsoluteWorkingDirectory())
	}
}

func installNodeJSFilter(filter Filter, filterPath string) {
	if hasPackageJson(filterPath) {
		Logger.Info("Installing npm dependencies...")
		RunSubProcess("npm", []string{"i", "--no-fund", "--no-audit"}, filterPath)
	}
}

func hasPackageJson(filterPath string) bool {
	_, err := os.Stat(path.Join(filterPath, "package.json"))
	return err == nil
}

func checkNodeJSRequirements() {
	_, err := exec.LookPath("node")
	if err != nil {
		Logger.Fatal("NodeJS not found. Download and install it from https://nodejs.org/en/")
	}
	cmd, _ := exec.Command("node", "--version").Output()
	a := strings.TrimPrefix(strings.Trim(string(cmd), " \n\t"), "v")
	Logger.Debugf("Found NodeJS version %s", a)
}
