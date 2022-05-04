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
}

type RunContext struct {
	AbsoluteLocation string
	Config           Config
	Profile          string
	Parent           *RunContext
}

type ProfileFilter struct {
	Filter
	Profile string `json:"-"`
}

func (f *ProfileFilter) Run(context RunContext) error {
	profile, ok := context.Config.Profiles[f.Profile]
	if !ok {
		return WrappedErrorf("Profile %s not found", f.Profile)
	}
	parent := context.Parent
	for parent != nil {
		if parent.Profile == f.Profile {
			return WrappedErrorf("Profile %s is circularly defined", f.Profile)
		}
		parent = parent.Parent
	}
	Logger.Infof("Running %q nested profile...", f.Profile)
	return RunProfileImpl(profile, f.Profile, context.Config, &context)
}

func (f *ProfileFilter) Check() error {
	return nil
}

func FilterDefinitionFromObject(id string) *FilterDefinition {
	return &FilterDefinition{Id: id}
}

func FilterFromObject(obj map[string]interface{}) (*Filter, error) {
	filter := &Filter{}
	// Name
	description, _ := obj["description"].(string)
	filter.Description = description
	// Disabled
	disabled, _ := obj["disabled"].(bool)
	filter.Disabled = disabled
	// Arguments
	arguments, ok := obj["arguments"].([]interface{})
	if !ok {
		arguments = nil
	}
	s := make([]string, len(arguments))
	for i, v := range arguments {
		s[i] = v.(string)
	}
	filter.Arguments = s
	// Settings
	settings, _ := obj["settings"].(map[string]interface{})
	filter.Settings = settings

	// Id
	// TODO - this property is redundant. You can find it in Filter and
	// FilterDefinition. This could cause hard to find bugs. There should
	// be a mechanism that ensures that the two are consistent. The filters
	// defined in "filter.json" don't have an id but its required by the
	// other filters.
	id, ok := obj["filter"].(string)
	if !ok {
		return nil, WrappedError("Missing \"filter\" property in filter.")
	}
	filter.Id = id
	return filter, nil
}

type FilterInstaller interface {
	InstallDependencies(parent *RemoteFilterDefinition) error
	Check() error
	CreateFilterRunner(runConfiguration map[string]interface{}) (FilterRunner, error)
}

type FilterRunner interface {
	CopyArguments(parent *RemoteFilter)
	Run(context RunContext) error
	IsDisabled() bool
	GetId() string
	Check() error
}

func (f *Filter) CopyArguments(parent *RemoteFilter) {
	f.Arguments = append(f.Arguments, parent.Arguments...)
	f.Settings = parent.Settings
}

func (f *Filter) Check() error {
	return NotImplementedError("Check")
}

func (f *Filter) Run(absoluteLocation string) error {
	return NotImplementedError("Run")
}

func (f *Filter) GetId() string {
	return f.Id
}

func (f *Filter) IsDisabled() bool {
	return f.Disabled
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
	case "nim":
		filter, err := NimFilterDefinitionFromObject(id, obj)
		if err != nil {
			return nil, WrapErrorf(
				err,
				"Unable to create Nim filter from %q filter definition.", id)
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
		"Unknown runWith %q in filter definition %q", runWith, id)
}

func FilterRunnerFromObjectAndDefinitions(
	obj map[string]interface{}, filterDefinitions map[string]FilterInstaller,
) (FilterRunner, error) {
	profile, ok := obj["profile"].(string)
	if ok {
		return &ProfileFilter{Profile: profile}, nil
	}
	filter, ok := obj["filter"].(string)
	if !ok {
		return nil, WrappedError(
			"Missing \"filter\" property in filter runner.")
	}
	if filterDefinition, ok := filterDefinitions[filter]; ok {
		filterRunner, err := filterDefinition.CreateFilterRunner(obj)
		if err != nil {
			return nil, WrapErrorf(
				err,
				"Unable to create filter runner from %q filter definition.",
				filter)
		}
		return filterRunner, nil
	}
	return nil, WrappedErrorf(
		"Unable to find %q filter in filter definitions.", filter)
}
