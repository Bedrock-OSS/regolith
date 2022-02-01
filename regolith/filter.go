package regolith

type FilterDefinition struct {
	Id string `json:"-"`
}

type Filter struct {
	Id        string                 `json:"filter,omitempty"`
	Name      string                 `json:"name,omitempty"`
	Disabled  bool                   `json:"disabled,omitempty"`
	Arguments []string               `json:"arguments,omitempty"`
	Settings  map[string]interface{} `json:"settings,omitempty"`
}

func FilterDefinitionFromObject(id string) *FilterDefinition {
	return &FilterDefinition{Id: id}
}

func FilterFromObject(obj map[string]interface{}) *Filter {
	filter := &Filter{}
	// Name
	name, _ := obj["name"].(string)
	filter.Name = name
	// Disabled
	disabled, _ := obj["disabled"].(bool)
	filter.Disabled = disabled
	// Arguments
	arguments, _ := obj["arguments"].([]string)
	filter.Arguments = arguments
	// Settings
	settings, _ := obj["settings"].(map[string]interface{})
	filter.Settings = settings
	return filter
}

type FilterInstaller interface {
	InstallDependencies(parent *RemoteFilterDefinition) error
	Check() error
	CreateFilterRunner(runConfiguration map[string]interface{}) FilterRunner
}

type FilterRunner interface {
	CopyArguments(parent *RemoteFilter)
	Run(absoluteLocation string) error
	Check() error
	GetFriendlyName() string
}

func FilterInstallerFromObject(id string, obj map[string]interface{}) FilterInstaller {
	runWith, _ := obj["runWith"].(string)
	switch runWith {
	case "java":
		return JavaFilterDefinitionFromObject(id, obj)
	case "nim":
		return NimFilterDefinitionFromObject(id, obj)
	case "nodejs":
		return NodeJSFilterDefinitionFromObject(id, obj)
	case "python":
		return PythonFilterDefinitionFromObject(id, obj)
	case "shell":
		return ShellFilterDefinitionFromObject(id, obj)
	case "":
		filter, err := BuildInFilterDefinitionFromObject(id, obj)
		if err == nil {
			return filter
		}
		return RemoteFilterDefinitionFromObject(id, obj)
	}
	Logger.Fatalf("Unknown runWith %q in filter definition %q", runWith, id)
	return nil
}

func FilterRunnerFromObjectAndDefinitions(
	obj map[string]interface{}, filterDefinitions map[string]FilterInstaller,
) FilterRunner {
	filter, _ := obj["filter"].(string)
	if filterDefinition, ok := filterDefinitions[filter]; ok {
		return filterDefinition.CreateFilterRunner(obj)
	} else {
		Logger.Fatalf("unable to find %q filter in filter definitions", filter)
	}
	return nil
}
