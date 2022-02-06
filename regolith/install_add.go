// Functions used for the "regolith install --add <filters...>" command
package regolith

import (
	"encoding/json"
	"io/ioutil"
	"os/exec"
	"strings"

	"golang.org/x/mod/semver"
)

// AddFilters handles the "regolith install --add" command. It adds the
// specified filters to the config.json file and installs them.
func AddFilters(args []string, force bool) error {
	for _, filter := range args {
		if err := addFilter(filter, force); err != nil {
			return WrapErrorf(err, "unable to add filter %q", filter)
		}
	}
	return nil
}

// TODO - proper error handling (propagate error)
// addFilter downloads a filter and adds it to the filter definitions list in
// config and installs it.
func addFilter(filter string, force bool) error {
	// Load the config file as a map. Loading as Config object could break some
	// of the custom data that could potentially be in the config file.
	// Open the filter definitions map.
	config, err := LoadConfigAsMap()
	if err != nil {
		return WrapError(err, "unable to load config file")
	}
	var regolithProject map[string]interface{}
	if _, ok := config["regolith"]; !ok {
		regolithProject = make(map[string]interface{})
		config["regolith"] = regolithProject
	} else {
		regolithProject, ok = config["regolith"].(map[string]interface{})
		if !ok {
			return WrapErrorf(
				nil,
				"'regolith' property of the config file is a %T not a map",
				config["regolith"])
		}
	}
	var filterDefinitions map[string]interface{}
	if _, ok := regolithProject["filterDefinitions"]; !ok {
		filterDefinitions = make(map[string]interface{})
		regolithProject["filterDefinitions"] = filterDefinitions
	} else {
		filterDefinitions, ok = regolithProject["filterDefinitions"].(map[string]interface{})
		if !ok {
			return WrapErrorf(
				nil,
				"'filterDefinitions' property of the config file is a %T not a map",
				regolithProject["filterDefinitions"])
		}
	}
	filterUrl, filterName, version, err := parseInstallFilterArg(filter)
	if err != nil {
		return WrapErrorf(
			err, "unable to parse filter name and version from %q", filter)
	}
	// Check if the filter is already installed
	if _, ok := filterDefinitions[filterName]; ok && !force {
		return WrapErrorf(
			nil,
			"The filter %q is already on the filter definitions list.\n"+
				"Please remove it first before installing it again or use "+
				"the --force option.", filterName)
	}
	// Add the filter info to filter definitions
	filterDefinition, err := FilterDefinitionFromTheInternet(
		filterUrl, filterName, version)
	if err != nil {
		return WrapErrorf(
			err, "unable to get filter definition %q", filter)
	}
	err = filterDefinition.Download(force)
	if err != nil {
		return WrapErrorf(
			err, "unable to download filter definition %q", filter)
	}
	// The default URL don't need to be added to the config file
	if filterDefinition.Url == StandardLibraryUrl {
		filterDefinition.Url = ""
	}
	// The "HEAD" and "latest" keywords should be the same in the config file
	// don't lock them to the actual versions
	if version == "HEAD" || version == "latest" {
		filterDefinition.Version = version
	}
	filterDefinitions[filterName] = filterDefinition
	// Save the config file
	jsonBytes, _ := json.MarshalIndent(config, "", "  ")
	err = ioutil.WriteFile(ManifestName, jsonBytes, 0666)
	if err != nil {
		return WrapError(err, "unable to save the config file")
	}
	// Install the dependencies of the filter
	err = filterDefinition.InstallDependencies(nil)
	if err != nil {
		return WrapErrorf(
			err, "unable to instsall dependencies of filter %q",
			filterDefinition.Id)
	}
	return nil
}

// parseInstallFilterArg parses a single argument of the
// "regolith install --add" command and returns the name, the url and
// the version of the filter.
func parseInstallFilterArg(arg string) (url, name, version string, err error) {
	// Parse the filter argument
	if strings.Contains(arg, "==") {
		splitStr := strings.Split(arg, "==")
		if len(splitStr) != 2 {
			err = WrapErrorf(
				nil,
				"Unable to parse argument %q as filter data. "+
					"The argument should contain an URL and optionally a "+
					"version number separated by '=='.",
				arg)
			return "", "", "", err
		}
		url, version = splitStr[0], splitStr[1]
	} else {
		url = arg
	}
	// Check if identifier is an URL. The last part of the URL is the name
	// of the filter
	if strings.Contains(url, "/") {
		splitStr := strings.Split(url, "/")
		name = splitStr[len(splitStr)-1]
		url = strings.Join(splitStr[:len(splitStr)-1], "/")
	} else {
		// Example inputs: "name_ninja==HEAD", "name_ninja"
		name = url
		url = StandardLibraryUrl
	}
	return
}

func GetRemoteFilterDownloadRef(
	url, name, version string, filterNamePrefix bool,
) (string, error) {
	// The custom type and a function is just to reduce the amount of code by
	// changing the function signature. In order to pass it in the 'vg' list.
	type vg []func(string, string) (string, error)
	var versionGetters vg
	getLatestRemoteFilterTag := func(url, name string) (string, error) {
		return GetLatestRemoteFilterTag(url, name, filterNamePrefix)
	}
	if version == "" {
		versionGetters = vg{getLatestRemoteFilterTag, GetHeadSha}
	} else if version == "latest" {
		versionGetters = vg{getLatestRemoteFilterTag}
	} else if version == "HEAD" {
		versionGetters = vg{GetHeadSha}
	} else {
		if semver.IsValid("v"+version) && filterNamePrefix {
			version = name + "-" + version
		}
		return version, nil
	}
	for _, versionGetter := range versionGetters {
		version, err := versionGetter(url, name)
		if err == nil {
			return version, nil
		}
	}
	return "", WrapErrorf(
		nil, "no valid version found for filter %q", name)
}

// GetLatestRemoteFilterTag returns the most up-to-date tag of the remote filter
// specified by the filter name and URL.
func GetLatestRemoteFilterTag(
	url, name string, filterNamePrefix bool,
) (string, error) {
	tags, err := ListRemoteFilterTags(url, name)
	if err == nil {
		if len(tags) > 0 {
			lastTag := tags[len(tags)-1]
			if filterNamePrefix {
				lastTag = name + "-" + lastTag
			}
			return lastTag, nil
		}
		return "", WrapErrorf(nil, "no tags found for filter %q", name)
	}
	return "", err
}

// ListRemoteFilterTags returns the list tags of the remote filter specified by the
// filter name and URL.
func ListRemoteFilterTags(url, name string) ([]string, error) {
	output, err := exec.Command(
		"git", "ls-remote", "--tags", "https://"+url,
	).Output()
	if err != nil {
		return nil, WrapErrorf(
			err, "unable to list tags for filter %q: ", name)
	}
	// Go line by line though the output
	var tags []string
	for _, line := range strings.Split(string(output), "\n") {
		// The command returns SHA and the tag name. We only want the tag name.
		if strings.Contains(line, "refs/tags/") {
			tag := strings.Split(line, "refs/tags/")[1]
			if !strings.HasPrefix(tag, name+"-") {
				continue
			}
			tag = tag[len(name)+1:]
			if semver.IsValid("v" + tag) {
				tags = append(tags, tag)
			}
		}
	}
	semver.Sort(tags)
	return tags, nil
}

// GetHeadSha returns the SHA of the HEAD of the repository specified by the
// filter URL. This function does not check whether the filter actually exists
// in the repository.
func GetHeadSha(url, name string) (string, error) {
	output, err := exec.Command(
		"git", "ls-remote", "--symref", "https://"+url, "HEAD",
	).Output()
	if err != nil {
		return "", WrapErrorf(
			err, "Unable to get head SHA for filter %q: ", name)
	}
	// The result is on the second line.
	lines := strings.Split(string(output), "\n")
	sha := strings.Split(lines[1], "\t")[0]
	return sha, nil
}
