package regolith

type Filter struct {
	Name      string                 `json:"name,omitempty"`
	Disabled  bool                   `json:"disabled,omitempty"`
	Arguments []string               `json:"arguments,omitempty"`
	Settings  map[string]interface{} `json:"settings,omitempty"`
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

type FilterRunner interface {
	Run(absoluteLocation string) error
	InstallDependencies(parent *RemoteFilter) error
	Check() error

	CopyArguments(parent *RemoteFilter)
	GetFriendlyName() string
}

func RunnableFilterFromObject(
	obj map[string]interface{}, filterDefinitions map[string]FilterDefinition,
) FilterRunner {
	runWith, _ := obj["runWith"].(string)
	switch runWith {
	case "java":
		return JavaFilterFromObject(obj)
	case "nim":
		return NimFilterFromObject(obj)
	case "nodejs":
		return NodeJSFilterFromObject(obj)
	case "python":
		return PythonFilterFromObject(obj)
	case "shell":
		return ShellFilterFromObject(obj)
	case "":
		filter, err := BuildInFilterFromObject(obj)
		if err == nil {
			return filter
		}
		return RemoteFilterFromObject(obj, filterDefinitions)
	}
	Logger.Fatalf("Unknown runWith '%s'", runWith)
	return nil
}

func LocalFilterFromObject(obj map[string]interface{}) FilterRunner {
	runWith, _ := obj["runWith"].(string)
	switch runWith {
	case "java":
		return JavaFilterFromObject(obj)
	case "nim":
		return NimFilterFromObject(obj)
	case "nodejs":
		return NodeJSFilterFromObject(obj)
	case "python":
		return PythonFilterFromObject(obj)
	case "shell":
		return ShellFilterFromObject(obj)
	case "":
		filter, err := BuildInFilterFromObject(obj)
		if err != nil {
			Logger.Fatal(err.Error())
		}
		return filter
	}
	Logger.Fatalf("Unknown runWith '%s'", runWith)
	return nil
}
