package regolith

type Installation struct {
	Filter  string `json:"-"`
	Url     string `json:"url,omitempty"`
	Version string `json:"version,omitempty"`
}

func InstallationFromObject(name string, obj map[string]interface{}) Installation {
	result := Installation{}
	result.Filter = name
	url, ok := obj["url"].(string)
	if !ok {
		Logger.Fatal("could not find url in installation %s in config.json", name)
	}
	result.Url = url
	version, ok := obj["version"].(string)
	if !ok {
		Logger.Fatal("could not find version in installation %s in config.json", name)
	}
	result.Version = version
	return result
}
