// Functions used for the "regolith install --add <filters...>" command
package regolith

import (
	"os/exec"
	"strings"

	"golang.org/x/mod/semver"
)

type parsedInstallFilterArg struct {
	// raw is the raw value of the filter argument before processing.
	raw string
	// url is the URL to the repository with the filter
	url string
	// name is the name of the filter (name of the subfoder in th repository)
	name string
	// version is the version string of the filter ("HEAD", semver, etc.)
	version string
}

// installFilters installs the filters from the list and their dependencies,
// and copies their data to the data path. If the filter is already installed,
// it returns an error unless the force flag is set.
func installFilters(
	filterDefinitions map[string]FilterInstaller, force bool,
	dataPath string,
) error {
	err := CreateDirectoryIfNotExists(".regolith/cache/filters", true)
	if err != nil {
		return PassError(err)
	}
	err = CreateDirectoryIfNotExists(".regolith/cache/venvs", true)
	if err != nil {
		return PassError(err)
	}

	// Download all of the remote filters
	resolverUpdated := false
	for name, filterDefinition := range filterDefinitions {
		Logger.Infof("Downloading %q filter...", name)
		if remoteFilter, ok := filterDefinition.(*RemoteFilterDefinition); ok {
			// Download resolver once if remote filter is found
			if !resolverUpdated {
				err = DownloadResolverMap()
				if err != nil {
					Logger.Warn("Failed to download resolver map.")
				}
				resolverUpdated = true
			}
			// Download the remote filter
			err := remoteFilter.Download(force)
			if err != nil {
				return WrapErrorf(
					err, "Could not download %q!", name)
			}
			// Copy the data of the remote filter to the data path
			remoteFilter.CopyFilterData(dataPath)
		}
		// Install the dependencies of the filter
		Logger.Infof("Installing %q filter dependencies...", name)
		err = filterDefinition.InstallDependencies(nil)
		if err != nil {
			return WrapErrorf(
				err, "Failed to install dependencies for %q filter.", name)
		}
	}
	return nil
}

// updateFilters updates the filters from the list.
func updateFilters(
	remoteFilterDefinitions map[string]FilterInstaller,
) error {
	err := CreateDirectoryIfNotExists(".regolith/cache/filters", true)
	if err != nil {
		return PassError(err)
	}
	err = CreateDirectoryIfNotExists(".regolith/cache/venvs", true)
	if err != nil {
		return PassError(err)
	}
	resolverUpdated := false
	// Download all of the remote filters
	for name, filterDefinition := range remoteFilterDefinitions {
		Logger.Infof("Updating %q filter...", name)
		if remoteFilter, ok := filterDefinition.(*RemoteFilterDefinition); ok {
			// Download resolver once if remote filter is found
			if !resolverUpdated {
				err = DownloadResolverMap()
				if err != nil {
					Logger.Warn("Failed to download resolver map.")
				}
				resolverUpdated = true
			}
			// Update the filter
			err := remoteFilter.Update()
			if err != nil {
				return WrapErrorf(
					err, "Could not update %q!", name)
			}
		}
	}
	return nil
}

// parseInstallFilterArgs parses a list of arguments of the
// "regolith install" command and returns a list of download tasks.
func parseInstallFilterArgs(
	filters []string,
) ([]*parsedInstallFilterArg, error) {
	result := []*parsedInstallFilterArg{}
	if len(filters) == 0 {
		return nil, WrappedError("No filters specified.")
	}

	// Parse the filter argument
	var url, name, version string
	var err error
	updatedResolver := false
	// resolvedArgs is used for finding duplicates (duplicate is a filter with
	// the same name and url)
	parsedArgs := make(map[[2]string]struct{})

	for _, arg := range filters {
		if strings.Contains(arg, "==") {
			splitStr := strings.Split(arg, "==")
			if len(splitStr) != 2 {
				return nil, WrappedErrorf(
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
			// Example inputs: "name_ninja==HEAD", "name_ninja"
			if !updatedResolver {
				err := DownloadResolverMap()
				if err != nil {
					Logger.Warn("Failed to download resolver map.")
				}
				updatedResolver = true
			}
			name = url
			url, err = ResolveUrl(url)
			if err != nil {
				return nil, WrapErrorf(
					err, "Unable to resolve URL of %q.", url)
			}
		}
		key := [2]string{url, name}
		if _, ok := parsedArgs[key]; ok {
			return nil, WrapErrorf(
				err, "Duplicate filter:\n URL: %s\n name: %s",
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
// from the "regolith install" command arguments.
func GetRemoteFilterDownloadRef(url, name, version string) (string, error) {
	// The custom type and a function is just to reduce the amount of code by
	// changing the function signature. In order to pass it in the 'vg' list.
	type vg []func(string, string) (string, error)
	var versionGetters vg
	if version == "" {
		versionGetters = vg{GetLatestRemoteFilterTag, GetHeadSha}
	} else if version == "latest" {
		versionGetters = vg{GetLatestRemoteFilterTag}
	} else if version == "HEAD" {
		versionGetters = vg{GetHeadSha}
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
	return "", WrappedErrorf("No valid version found for %q filter.", name)
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
		return "", WrappedErrorf("No tags found for %q filter.", name)
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
			err, "Unable to list tags for %q filter.", name)
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
			err, "Unable to get head SHA for %q filter.", name)
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
