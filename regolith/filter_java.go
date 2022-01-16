package regolith

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"time"
)

type JavaFilter struct {
	Filter

	Script string `json:"script,omitempty"`
}

func JavaFilterFromObject(obj map[string]interface{}) *JavaFilter {
	filter := &JavaFilter{Filter: *FilterFromObject(obj)}

	script, ok := obj["script"].(string)
	if !ok {
		Logger.Fatalf("Could filter %q", filter.GetFriendlyName())
	}
	filter.Script = script
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
						f.Script},
				f.Arguments...,
			),
			absoluteLocation,
			GetAbsoluteWorkingDirectory(),
		)
		if err != nil {
			return wrapError("Failed to run Java filter", err)
		}
	} else {
		jsonSettings, _ := json.Marshal(f.Settings)
		err := RunSubProcess(
			"java",
			append(
				[]string{
					"-jar", absoluteLocation + string(os.PathSeparator) +
						f.Script, string(jsonSettings)},
				f.Arguments...,
			),
			absoluteLocation,
			GetAbsoluteWorkingDirectory(),
		)
		if err != nil {
			return wrapError("Failed to run Java filter", err)
		}
	}
	return nil
}

func (f *JavaFilter) InstallDependencies(parent *RemoteFilter) error {
	return nil
}

func (f *JavaFilter) Check() error {
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
