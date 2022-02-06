package regolith

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"time"
)

type JavaFilterDefinition struct {
	FilterDefinition
	Script string `json:"script,omitempty"`
}

type JavaFilter struct {
	Filter
	Definition JavaFilterDefinition `json:"-"`
}

func JavaFilterDefinitionFromObject(id string, obj map[string]interface{}) *JavaFilterDefinition {
	filter := &JavaFilterDefinition{FilterDefinition: *FilterDefinitionFromObject(id)}
	script, ok := obj["script"].(string)
	if !ok {
		Logger.Fatalf("Could find script in filter defnition %q", filter.Id)
	}
	filter.Script = script
	return filter
}

func JavaFilterFromObject(obj map[string]interface{}, definition JavaFilterDefinition) *JavaFilter {
	filter := &JavaFilter{
		Filter:     *FilterFromObject(obj),
		Definition: definition,
	}
	return filter
}

func (f *JavaFilter) Run(absoluteLocation string) error {
	// Disabled filters are skipped
	if f.Disabled {
		Logger.Infof("Filter '%s' is disabled, skipping.", f.GetFriendlyName())
		return nil
	}
	Logger.Infof("Running filter %s", f.GetFriendlyName())
	start := time.Now()
	defer Logger.Debugf("Executed in %s", time.Since(start))

	// Run the filter
	if len(f.Settings) == 0 {
		err := RunSubProcess(
			"java",
			append(
				[]string{
					"-jar", absoluteLocation + string(os.PathSeparator) +
						f.Definition.Script,
				},
				f.Arguments...,
			),
			absoluteLocation,
			GetAbsoluteWorkingDirectory(),
		)
		if err != nil {
			return wrapError(err, "Failed to run Java filter")
		}
	} else {
		jsonSettings, _ := json.Marshal(f.Settings)
		err := RunSubProcess(
			"java",
			append(
				[]string{
					"-jar", absoluteLocation + string(os.PathSeparator) +
						f.Definition.Script, string(jsonSettings)},
				f.Arguments...,
			),
			absoluteLocation,
			GetAbsoluteWorkingDirectory(),
		)
		if err != nil {
			return wrapError(err, "Failed to run Java filter")
		}
	}
	return nil
}

func (f *JavaFilterDefinition) CreateFilterRunner(runConfiguration map[string]interface{}) FilterRunner {
	return JavaFilterFromObject(runConfiguration, *f)
}

func (f *JavaFilterDefinition) InstallDependencies(parent *RemoteFilterDefinition) error {
	return nil
}

func (f *JavaFilterDefinition) Check() error {
	_, err := exec.LookPath("java")
	if err != nil {
		Logger.Fatal("Java not found. Download and install it from https://adoptopenjdk.net/")
	}
	cmd, err := exec.Command("java", "--version").Output()
	if err != nil {
		return wrapError(err, "Failed to check Java version")
	}
	a := strings.Split(strings.Trim(string(cmd), " \n\t"), " ")
	if len(a) > 1 {
		Logger.Debugf("Found Java %s version %s", a[0], a[1])
	} else {
		Logger.Debugf("Failed to parse Java version")
	}
	return nil
}

func (f *JavaFilter) Check() error {
	return f.Definition.Check()
}

func (f *JavaFilter) CopyArguments(parent *RemoteFilter) {
	f.Arguments = parent.Arguments
	f.Settings = parent.Settings
}

func (f *JavaFilter) GetFriendlyName() string {
	if f.Name != "" {
		return f.Name
	}
	return "Unnamed Java filter"
}
