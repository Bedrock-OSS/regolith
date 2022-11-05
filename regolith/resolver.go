package regolith

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-getter"
	"muzzammil.xyz/jsonc"
)

const (
	// regolithConfigPath is a path to the regolith config relative to
	// UserCacheDir()
	regolithConfigPath = "regolith"
	// resolverUrl is  the default URL to the resolver.json file
	resolverUrl = "github.com/Bedrock-OSS/regolith-filter-resolver/resolver.json"
)

type ResolverMap struct {
	Url string `json:"url"`
}

type ResolverJson struct {
	FormatVersion string                 `json:"formatVersion"`
	Filters       map[string]ResolverMap `json:"filters"`
}

// GetRegolithConfigPath returns path to the regolith filesi in user app data
func GetRegolithConfigPath() (string, error) {
	path, err := os.UserCacheDir()
	if err != nil {
		return "", WrappedError(osUserCacheDirError)
	}
	return filepath.Join(path, regolithConfigPath), nil
}

// resolveResolverUrl resolves the resolver URL from the short name to a full
// URL that can be used by go-getter
func resolveResolverUrl(url string) (string, error) {
	urlParts := strings.Split(url, "/")
	if len(urlParts) < 4 {
		return "", WrappedErrorf(
			"Incorrect URL format.\n" +
				"Expected format:" +
				"github.com/<user-name>/<repo-name>/<path-to-the-resolver-file>")
	}
	repoUrl := strings.Join(urlParts[0:3], "/")
	path := strings.Join(urlParts[3:], "/")
	sha, err := GetHeadSha(repoUrl)
	if err != nil {
		return "", WrapError(
			err,
			"Failed to get the HEAD of the repository with the resolver.json file")
	}
	return fmt.Sprintf("https:/%s//%s?ref=%s", repoUrl, path, sha), nil
}

// DownloadResolverMap downloads the resolver.json file
func DownloadResolverMap() error {
	Logger.Info("Downloading resolver.json")
	path, err := GetRegolithConfigPath()
	if err != nil {
		return WrapError(err, getRegolithConfigPathError)
	}
	// Download to tmp path first and then move it to the real path,
	// overwritting the old file is possible only if download is successful
	tmpPath := filepath.Join(path, ".resolver-tmp.json")
	targetPath := filepath.Join(path, "resolver.json")
	userConfig, err := getUserConfig()
	if err != nil {
		return WrapError(err, getUserConfigError)
	}
	url, err := resolveResolverUrl(userConfig.Resolvers[0])
	if err != nil {
		return WrapError(err,
			"Failed to resolve the URL of the resolver.json file into a full"+
				" URL to download the file")
	}
	err = getter.GetFile(tmpPath, url)
	if err != nil {
		os.Remove(tmpPath) // I don't think errors matter here
		return WrapErrorf(
			err,
			"Unable to download filter resolver map file."+
				"Download URL: %s"+
				"Download path (for saving file): %s",
			userConfig.Resolvers[0], tmpPath)
	}
	os.Remove(targetPath)
	err = os.Rename(tmpPath, targetPath)
	if err != nil {
		return WrapErrorf(err, osRenameError, tmpPath, targetPath)
	}
	return nil
}

func LoadResolverAsMap() (map[string]interface{}, error) {
	resolverPath, err := GetRegolithConfigPath()
	if err != nil {
		return nil, WrapError(err, getRegolithConfigPathError)
	}
	resolverPath = filepath.Join(resolverPath, "resolver.json")
	file, err := ioutil.ReadFile(resolverPath)
	if err != nil {
		return nil, WrapErrorf(
			err, fileReadError, resolverPath)
	}
	var resolverJson map[string]interface{}
	err = jsonc.Unmarshal(file, &resolverJson)
	if err != nil {
		return nil, WrapErrorf(err, jsonUnmarshalError, resolverPath)
	}
	return resolverJson, nil
}

func ResolverFromObject(obj map[string]interface{}) (ResolverJson, error) {
	result := ResolverJson{}
	// FormatVersion
	formatVersionObj, ok := obj["formatVersion"]
	if !ok {
		return result, WrappedErrorf(
			jsonPathMissingError, "formatVersion")
	}
	formatVersion, ok := formatVersionObj.(string)
	if !ok {
		return result, WrappedErrorf(
			jsonPathTypeError, "formatVersion", "string")
	}
	result.FormatVersion = formatVersion
	// Filters
	filtersObj, ok := obj["filters"]
	if !ok {
		return result, WrappedErrorf(jsonPathMissingError, "filters")
	}
	filters, ok := filtersObj.(map[string]interface{})
	if !ok {
		return result, WrappedErrorf(jsonPathParseError, "filters", "object")
	}
	result.Filters = make(map[string]ResolverMap)
	for shortName, filterObj := range filters {
		filter, ok := filterObj.(map[string]interface{})
		if !ok {
			return result, WrappedErrorf(
				jsonPathTypeError,
				"filters->"+shortName, "object")
		}
		filterMap, err := ResolverMapFromObject(filter)
		if err != nil {
			return result, WrapErrorf(
				err, jsonPathParseError, "filters->"+shortName)
		}
		result.Filters[shortName] = filterMap
	}
	return result, nil
}

func ResolverMapFromObject(obj map[string]interface{}) (ResolverMap, error) {
	result := ResolverMap{}
	// Url
	urlObj, ok := obj["url"]
	if !ok {
		return result, WrappedErrorf(jsonPropertyMissingError, "url")
	}
	url, ok := urlObj.(string)
	if !ok {
		return result, WrappedErrorf(jsonPropertyTypeError, "url", "string")
	}
	result.Url = url
	return result, nil
}

// ResolveUrl tries to resolve the URL to a filter based on a shortName. If
// it fails it updates the resolver.json file and tries again
func ResolveUrl(shortName string) (string, error) {
	const resolverLoadErrror = "Unable to load the name to URL resolver map."
	resolverObj, err := LoadResolverAsMap()
	if err != nil {
		return "", WrapError(err, resolverLoadErrror)
	}
	resolver, err := ResolverFromObject(resolverObj)
	if err != nil {
		return "", WrapError(err, resolverLoadErrror)
	}
	filterMap, ok := resolver.Filters[shortName]
	userConfig, err := getUserConfig()
	if err != nil {
		return "", WrapError(err, getUserConfigError)
	}
	if !ok {
		return "", WrappedErrorf(
			"The filter doesn't have known mapping to URL in the URL "+
				"resolver.\n"+
				"Filter name: %s\n"+
				"Resolver URL: %s",
			shortName, userConfig.Resolvers[0])
	}
	return filterMap.Url, nil
}
