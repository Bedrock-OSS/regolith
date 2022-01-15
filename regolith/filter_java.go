package regolith

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
)

const javaFilterName = "java"

func RegisterJavaFilter(filters map[string]filterDefinition) {
	filters[javaFilterName] = filterDefinition{
		filter: runJavaFilter,
		installDependencies: func(filter Filter, path string) error {
			return nil
		},
		check: checkJavaRequirements,
	}
}

func runJavaFilter(filter Filter, settings map[string]interface{}, absoluteLocation string) error {
	if len(settings) == 0 {
		err := RunSubProcess("java", append([]string{"-jar", absoluteLocation + string(os.PathSeparator) + filter.Script}, filter.Arguments...), absoluteLocation, GetAbsoluteWorkingDirectory())
		if err != nil {
			return wrapError("Failed to run Java filter", err)
		}
	} else {
		jsonSettings, _ := json.Marshal(settings)
		err := RunSubProcess("java", append([]string{"-jar", absoluteLocation + string(os.PathSeparator) + filter.Script, string(jsonSettings)}, filter.Arguments...), absoluteLocation, GetAbsoluteWorkingDirectory())
		if err != nil {
			return wrapError("Failed to run Java filter", err)
		}
	}
	return nil
}

func checkJavaRequirements() error {
	_, err := exec.LookPath("java")
	if err != nil {
		Logger.Fatal("Java not found. Download and install it from https://adoptopenjdk.net/")
	}
	cmd, err := exec.Command("java", "--version").Output()
	if err != nil {
		return wrapError("Failed to check Java version", err)
	}
	a := strings.Split(strings.Trim(string(cmd), " \n\t"), " ")
	if len(a) > 1 {
		Logger.Debugf("Found Java %s version %s", a[0], a[1])
	} else {
		Logger.Debugf("Failed to parse Java version")
	}
	return nil
}
