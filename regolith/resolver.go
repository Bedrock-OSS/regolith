package regolith

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-getter"
	"muzzammil.xyz/jsonc"
)

const (
	// regolithConfigPath is a path to the regolith config relative to
	// UserCacheDir()
	regolithConfigPath = "regolith"
	// resolverUrl is an URL to the resolver.json file
	resolverUrl = "https://raw.githubusercontent.com/Bedrock-OSS/regolith-filter-resolver/main/resolver.json"
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
	err = getter.GetFile(tmpPath, resolverUrl)
	if err != nil {
		os.Remove(tmpPath) // I don't think errors matter here
		return WrapErrorf(
			err,
			"Unable to download filter resolver map file."+
				"Download URL: %s"+
				"Download path (for saving file): %s",
			resolverUrl, tmpPath)
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
	if !ok {
		return "", WrappedErrorf(
			"The filter doesn't have known mapping to URL in the URL "+
				"resolver.\n"+
				"Filter name: %s\n"+
				"Resolver URL: %s",
			shortName, resolverUrl)
	}
	return filterMap.Url, nil
}
