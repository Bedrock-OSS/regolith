package regolith

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Bedrock-OSS/go-burrito/burrito"

	"golang.org/x/mod/semver"
)

// Functions used for the "regolith install --add <filters...>" command

type parsedInstallFilterArg struct {
	// raw is the raw value of the filter argument before processing.
	raw string
	// url is the URL to the repository with the filter
	url string
	// name is the name of the filter (name of the subfolder in th repository)
	name string
	// version is the version string of the filter ("HEAD", semver, etc.)
	version string
}

// installFilters installs the filters from the list and their dependencies,
// and copies their data to the data path. If the filter is already installed,
// it returns an error unless the force flag is set.
func installFilters(
	filterDefinitions map[string]FilterInstaller, force bool,
	dataPath, dotRegolithPath string, refreshFilters bool,
) error {
	joinedPath := filepath.Join(dotRegolithPath, "cache/filters")
	err := os.MkdirAll(joinedPath, 0755)
	if err != nil {
		return burrito.WrapErrorf(err, osMkdirError, "cache/filters")
	}
	joinedPath = filepath.Join(dotRegolithPath, "cache/venvs")
	err = os.MkdirAll(joinedPath, 0755)
	if err != nil {
		return burrito.WrapErrorf(err, osMkdirError, "cache/venvs")
	}

	// Download all the remote filters
	for name, filterDefinition := range filterDefinitions {
		Logger.Infof("Downloading %q filter...", name)
		if remoteFilter, ok := filterDefinition.(*RemoteFilterDefinition); ok {
			// Download the remote filter, and its dependencies
			err := remoteFilter.Update(force, dotRegolithPath, refreshFilters)
			if err != nil {
				return burrito.WrapErrorf(err, remoteFilterDownloadError, name)
			}
			// Copy the data of the remote filter to the data path
			remoteFilter.CopyFilterData(dataPath, dotRegolithPath)
		} else {
			// Non-remote filters must always update their dependencies.
			// TODO - add option to track if the filter already installed
			// its dependencies.
			Logger.Infof("Installing %q filter dependencies...", name)
			err = filterDefinition.InstallDependencies(nil, dotRegolithPath)
			if err != nil {
				return burrito.WrapErrorf(
					err,
					"Failed to install dependencies of the filter.\nFilter: %s.",
					name)
			}
		}
	}
	return nil
}

// parseInstallFilterArgs parses a list of arguments of the
// "regolith install" command and returns a list of download tasks.
func parseInstallFilterArgs(
	filters []string, refreshResolvers bool,
) ([]*parsedInstallFilterArg, error) {
	var result []*parsedInstallFilterArg
	if len(filters) == 0 {
		return nil, burrito.WrappedError(
			"No filters specified.\n" +
				"Please specify at least one filter to install.")
	}

	// Parse the filter argument
	var url, name, version string
	var err error
	// resolvedArgs is used for finding duplicates (duplicate is a filter with
	// the same name and url)
	parsedArgs := make(map[[2]string]struct{})

	for _, arg := range filters {
		if strings.Contains(arg, "==") {
			splitStr := strings.Split(arg, "==")
			if len(splitStr) != 2 {
				return nil, burrito.WrappedErrorf(
					"Unable to parse argument.\n"+
						"Argument: %s\n"+
						"The argument should contain an URL and optionally a "+
						"version number separated by \"==\".",
					arg)
			}
			url, version = splitStr[0], splitStr[1]
		} else {
			url = arg
		}
		// Check if identifier is a URL. The last part of the URL is the name
		// of the filter
		if strings.Contains(url, "/") {
			splitStr := strings.Split(url, "/")
			name = splitStr[len(splitStr)-1]
			url = strings.Join(splitStr[:len(splitStr)-1], "/")
			if strings.HasPrefix(strings.ToLower(url), "https://") || strings.HasPrefix(strings.ToLower(url), "http://") {
				return nil, burrito.WrappedErrorf(
					"Unable to parse argument.\n"+
						"Argument: %s\n"+
						"Filter URLs cannot start with \"https://\" or \"http://\".",
					arg)
			}
		} else {
			// Example inputs: "name_ninja==HEAD", "name_ninja"
			name = url
			url, err = ResolveUrl(name, refreshResolvers)
			if err != nil {
				return nil, burrito.WrapErrorf(
					err,
					"Unable to resolve filter name to URL.\n"+
						"Filter name: %s", name)
			}
		}
		key := [2]string{url, name}
		if _, ok := parsedArgs[key]; ok {
			return nil, burrito.WrapErrorf(
				err, "Duplicate filter:\nURL: %s\nFilter name: %s",
				url, name)
		}
		parsedArgs[key] = struct{}{}
		result = append(result, &parsedInstallFilterArg{
			url:     url,
			name:    name,
			version: version,
		})
	}

	return result, nil
}

// GetRemoteFilterDownloadRef returns a reference for go-getter to be used
// to download a filter, based on the url, name and version properties from
// the "regolith install" command arguments.
func GetRemoteFilterDownloadRef(url, name, version string) (string, error) {
	// The custom type and a function is just to reduce the amount of code by
	// changing the function signature. In order to pass it in the 'vg' list.
	type vg []func(string, string) (string, error)
	var versionGetters vg
	getHeadSha := func(url, _ string) (string, error) { return GetHeadSha(url) }
	if version == "" {
		versionGetters = vg{GetLatestRemoteFilterTag, getHeadSha}
	} else if version == "latest" {
		versionGetters = vg{GetLatestRemoteFilterTag}
	} else if version == "HEAD" {
		versionGetters = vg{getHeadSha}
	} else {
		if semver.IsValid("v" + version) {
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
	return "", burrito.WrappedError(
		"Unable to find version of the filter that satisfies the " +
			"specified constraints.")
}

// GetLatestRemoteFilterTag returns the most up-to-date tag of the remote filter
// specified by the filter name and URL.
func GetLatestRemoteFilterTag(url, name string) (string, error) {
	tags, err := ListRemoteFilterTags(url, name)
	if err == nil {
		if len(tags) > 0 {
			lastTag := tags[len(tags)-1]
			return lastTag, nil
		}
		return "", burrito.WrappedError(
			"No version tags found for the filter on its repository.")
	}
	return "", err
}

// ListRemoteFilterTags returns the list tags of the remote filter specified by the
// filter name and URL.
func ListRemoteFilterTags(url, name string) ([]string, error) {
	commandArgs := []string{"ls-remote", "--tags", "https://" + url}
	output, err := exec.Command("git", commandArgs...).Output()
	if err != nil {
		command := "git " + strings.Join(commandArgs, " ")
		return nil, burrito.WrapErrorf(err, execCommandError, command)
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
			strippedTag := tag[len(name)+1:]
			if semver.IsValid("v" + strippedTag) {
				tags = append(tags, "v"+strippedTag)
			}
		}
	}
	semver.Sort(tags)
	for i, tag := range tags {
		tags[i] = name + "-" + tag[1:]
	}
	return tags, nil
}

// GetHeadSha returns the SHA of the HEAD of the repository specified by the
// filter URL. This function does not check whether the filter actually exists
// in the repository.
func GetHeadSha(url string) (string, error) {
	commandArgs := []string{
		"ls-remote", "--symref", "https://" + url, "HEAD"}
	output, err := exec.Command("git", commandArgs...).Output()
	if err != nil {
		commandText := "git " + strings.Join(commandArgs, " ")
		return "", burrito.WrapErrorf(err, execCommandError, commandText)
	}
	// The result is on the second line.
	lines := strings.Split(string(output), "\n")
	sha := strings.Split(lines[1], "\t")[0]
	return sha, nil
}

// trimFilterPrefix removes the prefix of the filter name from versionTag if
// versionTag follows the pattern <filterName>-<version>, otherwise it returns
// the same string.
func trimFilterPrefix(versionTag, prefix string) string {
	if strings.HasPrefix(versionTag, prefix+"-") {
		trimmedVersionTag := versionTag[len(prefix)+1:]
		if semver.IsValid("v" + trimmedVersionTag) {
			return trimmedVersionTag
		}
	}
	return versionTag
}
