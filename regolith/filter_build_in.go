package regolith

import "fmt"

type BuildInFilterDefinition struct {
	FilterDefinition
}

type BuildInFilter struct {
	Filter
	Definition BuildInFilterDefinition `json:"-"`
}

func BuildInFilterDefinitionFromObject(id string, obj map[string]interface{}) (*BuildInFilterDefinition, error) {
	result := &BuildInFilterDefinition{FilterDefinition: *FilterDefinitionFromObject(id)}
	if id != "hello_world" {
		return nil, fmt.Errorf("invalid filter id: %s", id)
	}
	return result, nil
}

func BuildInFilterFromObject(obj map[string]interface{}, definition BuildInFilterDefinition) *BuildInFilter {
	id, _ := obj["filter"].(string) // filter property is optional
	filter := &BuildInFilter{
		Filter:     *FilterFromObject(obj),
		Definition: definition,
	}
	filter.Id = id
	return filter
}

func (f *BuildInFilter) Run(absoluteLocation string) error {
	switch f.Id {
	case "hello_world":
		Logger.Info(
			"Hello world!\n" +
				"===========================================================\n" +
				" Welcome to Regolith!\n" +
				"\n" +
				" This message is generated from the 'hello_world' filter.\n" +
				" You can delete this filter when you're ready, and replace it with" +
				" Something more useful!\n" +
				"===========================================================\n",
		)
	default:
		return fmt.Errorf("unknown filter: %s", f.Name)
	}
	return nil
}

func (f *BuildInFilterDefinition) InstallDependencies(parent *RemoteFilterDefinition) error {
	return nil
}

func (f *BuildInFilterDefinition) Check() error {
	return nil
}

func (f *BuildInFilterDefinition) CreateFilterRunner(runConfiguration map[string]interface{}) FilterRunner {
	return BuildInFilterFromObject(runConfiguration, f)
}

func (f *BuildInFilter) CopyArguments(parent *RemoteFilter) {
	f.Arguments = parent.Arguments
	f.Settings = parent.Settings
}

func (f *BuildInFilter) GetFriendlyName() string {
	return f.Name
}
