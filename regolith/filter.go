package regolith

import (
	"slices"

	"github.com/Bedrock-OSS/go-burrito/burrito"
)

type FilterDefinition struct {
	Id string `json:"-"`
}

type Filter struct {
	Id          string         `json:"filter,omitempty"`
	Description string         `json:"name,omitempty"`
	Disabled    bool           `json:"disabled,omitempty"`
	Arguments   []string       `json:"arguments,omitempty"`
	Settings    map[string]any `json:"settings,omitempty"`
	When        string         `json:"when,omitempty"`
}

type RunContext struct {
	Initial          bool
	AbsoluteLocation string
	Config           *Config
	Profile          string
	Parent           *RunContext
	DotRegolithPath  string
	uwpDevelopment   bool
	Settings         map[string]any

	// interruption is a channel used to receive notifications about changes
	// in the source files, in order to trigger a restart of the program in
	// the watch mode. The string sent to the channel is the name of the source
	// of the change ("rp", "bp" or "data"), which may be used to handle
	// some interruptions differently.
	interruption chan string

	// fileWatchingError is used to pass any errors that may occur during
	// file watching.
	fileWatchingError chan error

	fileWatchingStage chan string
}

// GetProfile returns the Profile structure from the context.
func (c *RunContext) GetProfile() (Profile, error) {
	profile, ok := c.Config.Profiles[c.Profile]
	if !ok {
		return Profile{}, burrito.WrappedErrorf("Profile with specified name doesn't exist.\n"+
			"Profile name: %s", c.Profile)
	}
	return profile, nil
}

// IsInWatchMode returns a value that shows whether the context is in the
// watch mode.
func (c *RunContext) IsInWatchMode() bool {
	return c.interruption != nil
}

// StartWatchingSourceFiles causes the Context to start goroutines that watch
// for changes in the source files and report that to the
func (c *RunContext) StartWatchingSourceFiles() error {
	c.interruption = make(chan string)
	c.fileWatchingError = make(chan error)
	c.fileWatchingStage = make(chan string)
	err := NewDirWatcher(c.Config, c.interruption, c.fileWatchingError, c.fileWatchingStage)
	if err != nil {
		return err
	}
	return nil
}

// IsInterrupted returns true if there is a message on the interruptionChannel
// unless the source of the interruption is on the list of ignored sources.
// This function does not block.
func (c *RunContext) IsInterrupted(ignoredSource ...string) bool {
	if c.interruption == nil {
		return false
	}
	select {
	case source := <-c.interruption:
		return !slices.Contains(ignoredSource, source)
	default:
		return false
	}
}

func FilterDefinitionFromObject(id string) *FilterDefinition {
	return &FilterDefinition{Id: id}
}

func filterFromObject(obj map[string]any, id string) (*Filter, error) {
	filter := &Filter{}
	// Name
	description, _ := obj["description"].(string)
	filter.Description = description
	// Disabled
	disabled, _ := obj["disabled"].(bool)
	filter.Disabled = disabled
	// Arguments
	arguments, ok := obj["arguments"]
	if ok {
		// Try to parse arguments as []interface{} and as []string
		// one format is used when parsed from JSON, and the other format is
		// used by the ApplyFilter() function.
		switch arguments := arguments.(type) {
		case []any:
			s := make([]string, len(arguments))
			for i, v := range arguments {
				s[i] = v.(string)
			}
			filter.Arguments = s
		case []string:
			filter.Arguments = arguments
		default:
			filter.Arguments = []string{}
		}
	} else {
		filter.Arguments = []string{}
	}
	// Settings
	settings, _ := obj["settings"].(map[string]any)
	filter.Settings = settings
	// When
	when, ok := obj["when"]
	if !ok {
		when = ""
	} else {
		when, ok = when.(string)
		if !ok {
			when = ""
		}
	}
	filter.When = when.(string)

	// Id
	if id == "" {
		idObj, ok := obj["filter"]
		if !ok {
			return nil, burrito.WrappedErrorf(jsonPropertyMissingError, "filter")
		}
		parsedId, ok := idObj.(string)
		if !ok {
			return nil, burrito.WrappedErrorf(jsonPropertyTypeError, "filter", "string")
		}
		id = parsedId
	}
	filter.Id = id
	return filter, nil
}

type FilterInstaller interface {
	InstallDependencies(parent *RemoteFilterDefinition, dotRegolithPath string) error
	Check(context RunContext) error
	CreateFilterRunner(runConfiguration map[string]any, id string) (FilterRunner, error)
}

type FilterRunner interface {
	// CopyArguments copies the arguments from the parent filter to this
	// filter. It's used  for the remote filters.
	CopyArguments(parent *RemoteFilter)

	// Run runs the filter. If the context is in the watch mode, it also
	// checks whether there were any interruptions.
	// It returns true if the filter was interrupted. If the watch mode is
	// disabled it always returns false.
	Run(context RunContext) (bool, error)

	// IsDisabled returns whether the filter is disabled.
	IsDisabled(ctx RunContext) (bool, error)

	// GetId returns the id of the filter.
	GetId() string

	// GetSettings returns the settings of the filter.
	GetSettings() map[string]any

	// Check checks whether the requirements of the filter are met. For
	// example, a Python filter requires Python to be installed.
	Check(context RunContext) error

	// IsUsingDataExport returns whether the filter wants its data to be
	// exported back to the data folder after running the profile.
	IsUsingDataExport(dotRegolithPath string, ctx RunContext) (bool, error)
}

func (f *Filter) CopyArguments(parent *RemoteFilter) {
	f.Arguments = append(f.Arguments, parent.Arguments...)
	f.Settings = parent.Settings
	if f.When == "" {
		f.When = parent.When
	}
}

func (f *Filter) Check() error {
	return NotImplementedError("Check")
}

func (f *Filter) Run(context RunContext) (bool, error) {
	return false, NotImplementedError("Run")
}

func (f *Filter) GetId() string {
	return f.Id
}

func (f *Filter) GetSettings() map[string]any {
	return f.Settings
}

func (f *Filter) IsDisabled(ctx RunContext) (bool, error) {
	if f.Disabled {
		return true, nil
	}
	if f.When != "" {
		condition, err := EvalCondition(f.When, ctx)
		if err != nil {
			return false, burrito.WrapError(err, "Could not evaluate condition.")
		}
		return !condition, nil
	}
	return false, nil
}

func (f *Filter) IsUsingDataExport(_ string, _ RunContext) (bool, error) {
	return false, nil
}

type filterInstallerFactory struct {
	constructor func(string, map[string]any) (FilterInstaller, error)
	name        string
}

var filterInstallerFactories = map[string]filterInstallerFactory{
	"java": {
		constructor: func(id string, obj map[string]any) (FilterInstaller, error) {
			return JavaFilterDefinitionFromObject(id, obj)
		},
		name: "Java",
	},
	"dotnet": {
		constructor: func(id string, obj map[string]any) (FilterInstaller, error) {
			return DotNetFilterDefinitionFromObject(id, obj)
		},
		name: ".Net",
	},
	"nim": {
		constructor: func(id string, obj map[string]any) (FilterInstaller, error) {
			return NimFilterDefinitionFromObject(id, obj)
		},
		name: "Nim",
	},
	"deno": {
		constructor: func(id string, obj map[string]any) (FilterInstaller, error) {
			return DenoFilterDefinitionFromObject(id, obj)
		},
		name: "Deno",
	},
	"nodejs": {
		constructor: func(id string, obj map[string]any) (FilterInstaller, error) {
			return NodeJSFilterDefinitionFromObject(id, obj)
		},
		name: "NodeJs",
	},
	"python": {
		constructor: func(id string, obj map[string]any) (FilterInstaller, error) {
			return PythonFilterDefinitionFromObject(id, obj)
		},
		name: "Python",
	},
	"shell": {
		constructor: func(id string, obj map[string]any) (FilterInstaller, error) {
			return ShellFilterDefinitionFromObject(id, obj)
		},
		name: "shell",
	},
	"exe": {
		constructor: func(id string, obj map[string]any) (FilterInstaller, error) {
			return ExeFilterDefinitionFromObject(id, obj)
		},
		name: "exe",
	},
	"": {
		constructor: func(id string, obj map[string]any) (FilterInstaller, error) {
			return RemoteFilterDefinitionFromObject(id, obj)
		},
		name: "remote",
	},
}

func FilterInstallerFromObject(id string, obj map[string]any) (FilterInstaller, error) {
	runWith, _ := obj["runWith"].(string)
	if factory, ok := filterInstallerFactories[runWith]; ok {
		filter, err := factory.constructor(id, obj)
		if err != nil {
			return nil, burrito.WrapErrorf(
				err,
				"Unable to create %s filter from %q filter definition.",
				factory.name, id)
		}
		return filter, nil
	}
	return nil, burrito.WrappedErrorf(
		"Invalid runWith value filter definition.\n"+
			"Filter: %s\n"+
			"Value: %s\n"+
			"Valid values: java, dotnet, nim, deno, nodejs, python, shell, exe",
		runWith, id)
}

func FilterRunnerFromObjectAndDefinitions(
	obj map[string]any, filterDefinitions map[string]FilterInstaller,
) (FilterRunner, error) {
	profile, ok := obj["profile"].(string)
	if ok {
		return &ProfileFilter{Profile: profile}, nil
	}
	filterObj, ok := obj["filter"]
	if !ok {
		return nil, burrito.WrappedErrorf(jsonPropertyMissingError, "filter")
	}
	filter, ok := filterObj.(string)
	if !ok {
		return nil, burrito.WrappedErrorf(jsonPropertyTypeError, "filter", "string")
	}
	if filterDefinition, ok := filterDefinitions[filter]; ok {
		filterRunner, err := filterDefinition.CreateFilterRunner(obj, filter)
		if err != nil {
			return nil, burrito.WrapErrorf(err, createFilterRunnerError, filter)
		}
		return filterRunner, nil
	}
	return nil, burrito.WrappedErrorf(
		"Unable to find filter in filter definitions.\nFilter name: %s",
		filter)
}
