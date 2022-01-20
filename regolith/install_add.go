// Functions used for the "regolith install --add <filters...>" command
package regolith

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	"golang.org/x/mod/semver"
)

// AddFilters handles the "regolith install --add" command. It adds the
// specified filters to the config.json file and installs them.
func AddFilters(args []string, force bool) error {
	for _, filter := range args {
		addFilter(filter, force)
	}
	// TODO - how do we handle errors? With Logger.Fattal or by return statemets?
	return nil
}

// addFilter downloads a filter and adds it to the installations list in
// config and installs it.
func addFilter(filter string, force bool) {
	// Load the config file as a map. Loading as Config object could break some
	// of the custom data that could potentially be in the config file.
	// Open the installations map.
	config := LoadConfigAsMap()
	var regolithProject map[string]interface{}
	if _, ok := config["regolith"]; !ok {
		regolithProject = make(map[string]interface{})
		config["regolith"] = regolithProject
	} else {
		regolithProject, ok = config["regolith"].(map[string]interface{})
		if !ok {
			Logger.Fatal(
				"Unable to convert the 'regolith' property of the " +
					"config file to a map.")
		}
	}
	var installations map[string]interface{}
	if _, ok := regolithProject["installations"]; !ok {
		installations = make(map[string]interface{})
		regolithProject["installations"] = installations
	} else {
		installations, ok = regolithProject["installations"].(map[string]interface{})
		if !ok {
			Logger.Fatal(
				"Unable to convert the 'regolith->installations' property " +
					"of the config file to a map.")
		}
	}

	filterUrl, filterName, version := parseInstallFilterArg(filter)

	// Check if the filter is already installed
	if _, ok := installations[filterName]; ok && !force {
		Logger.Fatalf(
			"The filter %q is already on the installations list."+
				"Please remove it first before installing it again or use "+
				"the --force option.", filterName)
	}
	// Add the filter info to Installations
	installation, err := InstallationFromTheInternet(
		filterUrl, filterName, version)
	if err != nil {
		Logger.Fatal(err)
	}
	installations[filterName] = installation
	// Save the config file
	jsonBytes, _ := json.MarshalIndent(config, "", "  ")
	err = ioutil.WriteFile(ManifestName, jsonBytes, 0666)
	if err != nil {
		Logger.Fatal("Unable to save the config file: ", err)
	}
}

// parseInstallFilterArg parses a single argument of the
// "regolith install --add" command and returns the name, the url and
// the version of the filter.
func parseInstallFilterArg(arg string) (url, name, version string) {
	// Parse the filter argument
	if strings.Contains(arg, "==") {
		splitStr := strings.Split(arg, "==")
		if len(splitStr) != 2 {
			Logger.Fatalf(
				"Unable to parse argument %q as filter data. "+
					"The argument should contain an URL and optionally a "+
					"version number separated by '=='.",
				arg)
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
		Logger.Fatalf(
			"Unable to parse argument %q as filter data. "+
				"The argument should contain an URL and optionally a "+
				"version number separated by '=='.",
			arg)
	}
	return
}

// InstallationFromTheInternet downloads a filter from the internet and returns
// the installation data.
func InstallationFromTheInternet(url, name, version string) (Installation, error) {
	type vg []func(string, string) (string, error)
	var versionGetters vg
	if version == "" {
		versionGetters = vg{GetLastFilterTag, GetHeadSha}
	} else if version == "latest" {
		versionGetters = vg{GetLastFilterTag}
	} else if version == "HEAD" {
		versionGetters = vg{GetHeadSha}
	} else {
		return Installation{
			Filter:  name,
			Version: version,
			Url:     url,
		}, nil
	}
	for _, versionGetter := range versionGetters {
		version, err := versionGetter(url, name)
		if err == nil {
			return Installation{
				Filter:  name,
				Version: version,
				Url:     url,
			}, nil
		}
	}
	return Installation{}, fmt.Errorf(
		"no valid version found for filter %q", name)
}

// GetLastFilterTag returns the most up-to-date tag of the remote filter
// specified by the filter name and URL.
func GetLastFilterTag(url, name string) (string, error) {
	tags, err := ListFilterTags(url, name)
	if err == nil {
		if len(tags) > 0 {
			semver.Sort(tags)
			return tags[len(tags)-1], nil
		}
		return "", fmt.Errorf("no tags found for filter %q", name)
	}
	return "", err
}

// ListFilterTags returns the list tags of the remote filter specified by the
// filter name and URL.
func ListFilterTags(url, name string) ([]string, error) {
	output, err := exec.Command(
		"git", "ls-remote", "--sort=committerdate", "--tags",
		"https://"+url,
	).Output()
	if err != nil {
		return nil, wrapError(
			fmt.Sprintf("unable to list tags for filter %q: ", name),
			err)
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
			if semver.IsValid(tag) {
				tags = append(tags, tag)
			}
		}
	}
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
		return "", wrapError(
			fmt.Sprintf("Unable to get head SHA for filter %q: ", name),
			err)
	}
	// The result is on the second line.
	lines := strings.Split(string(output), "\n")
	sha := strings.Split(lines[1], "\t")[0]
	return sha, nil
}
