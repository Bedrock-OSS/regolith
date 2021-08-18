package src

import (
	"encoding/json"
	"os/exec"
	"strings"
)

const nodeJSFilterName = "nodejs"

func RegisterNodeJSFilter(filters map[string]filterDefinition) {
	filters[nodeJSFilterName] = filterDefinition{
		filter: runNodeJSFilter,
		check:  checkNodeJSRequirements,
	}
}

func runNodeJSFilter(filter Filter, settings map[string]interface{}, absoluteLocation string) {
	if len(settings) == 0 {
		RunSubProcess("node", append([]string{absoluteLocation}, filter.Arguments...))
	} else {
		jsonSettings, _ := json.Marshal(settings)
		RunSubProcess("node", append([]string{absoluteLocation, string(jsonSettings)}, filter.Arguments...))
	}
}

func checkNodeJSRequirements() {
	_, err := exec.LookPath("node")
	if err != nil {
		Logger.Fatal("NodeJS not found")
	}
	cmd, _ := exec.Command("node", "--version").Output()
	a := strings.TrimPrefix(strings.Trim(string(cmd), " \n\t"), "v")
	Logger.Debugf("Found NodeJS version %s", a)
}
