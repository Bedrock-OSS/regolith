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
	arguments, _ := obj["arguments"].([]string)
	filter.Arguments = arguments
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
	Run(absoluteLocation string) error
	Check() error
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
