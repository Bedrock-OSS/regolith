package regolith

import "fmt"

type FilterBuildIn struct {
	Filter
	Id string `json:"filter,omitempty"`
}

func BuildInFilterFromObject(obj map[string]interface{}) (*FilterBuildIn, error) {
	id, _ := obj["filter"].(string) // filter property is optional
	if id != "hello_world" {
		return nil, fmt.Errorf("invalid filter id: %s", id)
	}
	filter := &FilterBuildIn{Filter: *FilterFromObject(obj)}
	filter.Id = id
	return filter, nil
}

func (f *FilterBuildIn) Run(absoluteLocation string) error {
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

func (f *FilterBuildIn) InstallDependencies(parent *RemoteFilter) error {
	return nil
}

func (f *FilterBuildIn) Check() error {
	return nil
}

func (f *FilterBuildIn) CopyArguments(parent *RemoteFilter) {
	f.Arguments = parent.Arguments
	f.Settings = parent.Settings
}

func (f *FilterBuildIn) GetFriendlyName() string {
	return f.Name
}
