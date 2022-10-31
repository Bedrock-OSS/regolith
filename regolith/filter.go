package regolith

type FilterDefinition struct {
	Id string `json:"-"`
}

type Filter struct {
	Id          string                 `json:"filter,omitempty"`
	Description string                 `json:"name,omitempty"`
	Disabled    bool                   `json:"disabled,omitempty"`
	Arguments   []string               `json:"arguments,omitempty"`
	Settings    map[string]interface{} `json:"settings,omitempty"`
	When        string                 `json:"when,omitempty"`
}

type RunContext struct {
	AbsoluteLocation string
	Config           *Config
	Profile          string
	Parent           *RunContext
	DotRegolithPath  string

	// interruptionChannel is a channel that is used to notify about changes
	// in the sourec files, in order to trigger a restart of the program in
	// the watch mode. The string send to the channel is the name of the source
	// of the change ("rp", "bp" or "data"), which may be used to handle
	// some interuptions differently.
	interruptionChannel chan string
}

// GetProfile returns the Profile structure from the context.
func (c *RunContext) GetProfile() (Profile, error) {
	profile, ok := c.Config.Profiles[c.Profile]
	if !ok {
		return Profile{}, WrappedErrorf("Profile with specified name doesn't exist.\n"+
			"Profile name: %s", c.Profile)
	}
	return profile, nil
}

// IsWatchMode returns a value that shows whether the context is in the
// watch mode.
func (c *RunContext) IsInWatchMode() bool {
	return c.interruptionChannel == nil
}

// StartWatchingSourceFiles causes the Context to start goroutines that watch
// for changes in the source files and report that to the
func (c *RunContext) StartWatchingSourceFiles() error {
	// TODO - if you want to be able to restart the watcher, you need to handle
	// closing the channels somewhere. Currently the watching goroutines yield
	// their messages until the end of the program. Sending to a closed channel
	// would cause panic.
	if c.interruptionChannel != nil {
		return WrappedError("Files are already being watched.")
	}
	rpWatcher, err := NewDirWatcher(c.Config.ResourceFolder)
	if err != nil {
		return WrapError(err, "Could not create resource pack watcher.")
	}
	bpWatcher, err := NewDirWatcher(c.Config.BehaviorFolder)
	if err != nil {
		return WrapError(err, "Could not create behavior pack watcher.")
	}
	dataWatcher, err := NewDirWatcher(c.Config.DataPath)
	if err != nil {
		return WrapError(err, "Could not create data watcher.")
	}
	c.interruptionChannel = make(chan string)
	yieldChanges := func(
		watcher *DirWatcher, sourceName string,
	) {
		for {
			err := watcher.WaitForChangeGroup(
				100, c.interruptionChannel, sourceName)
			if err != nil {
				return
			}
		}
	}
	go yieldChanges(rpWatcher, "rp")
	go yieldChanges(bpWatcher, "bp")
	go yieldChanges(dataWatcher, "data")
	return nil
}

// AwaitInterruption locks the goroutine with the interruption channel until
// the Config is interrupted and returns the interruption message.
func (c *RunContext) AwaitInterruption() string {
	return <-c.interruptionChannel
}

// IsInterrupted returns true if there is a message on the interruptionChannel
// unless the source of the interruption is on the list of ignored sources.
// This function does not block.
func (c *RunContext) IsInterrupted(ignoredSourece ...string) bool {
	if c.interruptionChannel == nil {
		return false
	}
	select {
	case source := <-c.interruptionChannel:
		for _, ignored := range ignoredSourece {
			if ignored == source {
				return false
			}
		}
		return true
	default:
		return false
	}
}

func FilterDefinitionFromObject(id string) *FilterDefinition {
	return &FilterDefinition{Id: id}
}

func filterFromObject(obj map[string]interface{}) (*Filter, error) {
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
		// used by the Tool() function.
		switch arguments := arguments.(type) {
		case []interface{}:
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
	settings, _ := obj["settings"].(map[string]interface{})
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
	idObj, ok := obj["filter"]
	if !ok {
		return nil, WrappedErrorf(jsonPropertyMissingError, "filter")
	}
	id, ok := idObj.(string)
	if !ok {
		return nil, WrappedErrorf(jsonPropertyTypeError, "filter", "string")
	}
	filter.Id = id
	return filter, nil
}

type FilterInstaller interface {
	InstallDependencies(parent *RemoteFilterDefinition, dotRegolithPath string) error
	Check(context RunContext) error
	CreateFilterRunner(runConfiguration map[string]interface{}) (FilterRunner, error)
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
	IsDisabled() (bool, error)

	// GetId returns the id of the filter.
	GetId() string

	// Check checks whether the requirements of the filter are met. For
	// example, a Python filter requires Python to be installed.
	Check(context RunContext) error
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

func (f *Filter) IsDisabled() (bool, error) {
	if f.Disabled {
		return true, nil
	}
	if f.When != "" {
		condition, err := EvalCondition(f.When)
		if err != nil {
			return false, WrapError(err, "Could not evaluate condition.")
		}
		return !condition, nil
	}
	return false, nil
}

func FilterInstallerFromObject(id string, obj map[string]interface{}) (FilterInstaller, error) {
	runWith, _ := obj["runWith"].(string)
	switch runWith {
	case "java":
		filter, err := JavaFilterDefinitionFromObject(id, obj)
		if err != nil {
			return nil, WrapErrorf(
				err,
				"Unable to create Java filter from %q filter definition.", id)
		}
		return filter, nil
	case "dotnet":
		filter, err := DotNetFilterDefinitionFromObject(id, obj)
		if err != nil {
			return nil, WrapErrorf(
				err,
				"Unable to create .Net filter from %q filter definition.", id)
		}
		return filter, nil
	case "nim":
		filter, err := NimFilterDefinitionFromObject(id, obj)
		if err != nil {
			return nil, WrapErrorf(
				err,
				"Unable to create Nim filter from %q filter definition.", id)
		}
		return filter, nil
	case "deno":
		filter, err := DenoFilterDefinitionFromObject(id, obj)
		if err != nil {
			return nil, WrapErrorf(
				err,
				"Unable to create Deno filter from %q filter definition.", id)
		}
		return filter, nil
	case "nodejs":
		filter, err := NodeJSFilterDefinitionFromObject(id, obj)
		if err != nil {
			return nil, WrapErrorf(
				err,
				"Unable to create NodeJs filter from %q filter definition.",
				id)
		}
		return filter, nil
	case "python":
		filter, err := PythonFilterDefinitionFromObject(id, obj)
		if err != nil {
			return nil, WrapErrorf(
				err,
				"Unable to create Python filter from %q filter definition.",
				id)
		}
		return filter, nil
	case "shell":
		filter, err := ShellFilterDefinitionFromObject(id, obj)
		if err != nil {
			return nil, WrapErrorf(
				err,
				"Unable to create shell filter from %q filter definition.", id)
		}
		return filter, nil
	case "exe":
		filter, err := ExeFilterDefinitionFromObject(id, obj)
		if err != nil {
			return nil, WrapErrorf(
				err,
				"Unable to create exe filter from %q filter definition.", id)
		}
		return filter, nil
	case "":
		filter, err := RemoteFilterDefinitionFromObject(id, obj)
		if err != nil {
			return nil, WrapErrorf(
				err,
				"Unable to create remote filter from %q filter definition.",
				id)
		}
		return filter, nil
	}
	return nil, WrappedErrorf(
		"Invalid runWith value filter definition.\n"+
			"Filter: %s\n"+
			"Value: %s\n"+
			"Valid values: java, dotnet, nim, deno, nodejs, python, shell, exe",
		runWith, id)
}

func FilterRunnerFromObjectAndDefinitions(
	obj map[string]interface{}, filterDefinitions map[string]FilterInstaller,
) (FilterRunner, error) {
	profile, ok := obj["profile"].(string)
	if ok {
		return &ProfileFilter{Profile: profile}, nil
	}
	filterObj, ok := obj["filter"]
	if !ok {
		return nil, WrappedErrorf(jsonPropertyMissingError, "filter")
	}
	filter, ok := filterObj.(string)
	if !ok {
		return nil, WrappedErrorf(jsonPropertyTypeError, "filter", "string")
	}
	if filterDefinition, ok := filterDefinitions[filter]; ok {
		filterRunner, err := filterDefinition.CreateFilterRunner(obj)
		if err != nil {
			return nil, WrapErrorf(err, createFilterRunnerError, filter)
		}
		return filterRunner, nil
	}
	return nil, WrappedErrorf(
		"Unable to find filter in filter definitions.\nFilter name: %s",
		filter)
}
