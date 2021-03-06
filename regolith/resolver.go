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

// GetRegolithConfigPath returns path to the resolver.json file
func GetRegolithConfigPath() (string, error) {
	path, err := os.UserCacheDir()
	if err != nil {
		return "", WrappedError("Unable to get user cache dir")
	}
	return filepath.Join(path, regolithConfigPath), nil
}

// DownloadResolverMap downloads the resolver.json file
func DownloadResolverMap() error {
	Logger.Info("Downloading resolver.json")
	path, err := GetRegolithConfigPath()
	if err != nil {
		return PassError(err)
	}
	// Download to tmp path first and then move it to the real path,
	// overwritting the old file is possible only if download is successful
	tmpPath := filepath.Join(path, ".resolver-tmp.json")
	targetPath := filepath.Join(path, "resolver.json")
	err = getter.GetFile(tmpPath, resolverUrl)
	if err != nil {
		os.Remove(tmpPath) // I don't think errors matter here
		return WrapError(err, "Unable to download filter resolver map file.")
	}
	os.Remove(targetPath)
	err = os.Rename(tmpPath, targetPath)
	if err != nil {
		return WrapErrorf(
			err, "Unable to move the temporary download file \"%s\" to "+
				"the target location \"%s\"", tmpPath, targetPath)
	}
	return nil
}

func LoadResolverAsMap() (map[string]interface{}, error) {
	resolverPath, err := GetRegolithConfigPath()
	if err != nil {
		return nil, WrapError(
			err, "Unable to get the resolver.json path")
	}
	resolverPath = filepath.Join(resolverPath, "resolver.json")
	file, err := ioutil.ReadFile(resolverPath)
	if err != nil {
		return nil, WrapError(
			err, "Unable to open the resolver.json file")
	}
	var resolverJson map[string]interface{}
	err = jsonc.Unmarshal(file, &resolverJson)
	if err != nil {
		return nil, WrapError(
			err, "Could not load resolver.json as a JSON file.")
	}
	return resolverJson, nil
}

func ResolverFromObject(obj map[string]interface{}) (ResolverJson, error) {
	result := ResolverJson{}
	// FormatVersion
	formatVersionObj, ok := obj["formatVersion"]
	if !ok {
		return result, WrappedError(
			"The \"formatVersion\" property is missing.")
	}
	formatVersion, ok := formatVersionObj.(string)
	if !ok {
		return result, WrappedError(
			"The \"formatVersion\" property is not a string.")
	}
	result.FormatVersion = formatVersion
	// Filters
	filtersObj, ok := obj["filters"]
	if !ok {
		return result, WrappedError(
			"The \"filters\" property is missing.")
	}
	filters, ok := filtersObj.(map[string]interface{})
	if !ok {
		return result, WrappedError(
			"The \"filters\" property is not a map.")
	}
	result.Filters = make(map[string]ResolverMap)
	for shortName, filterObj := range filters {
		filter, ok := filterObj.(map[string]interface{})
		if !ok {
			return result, WrappedError(
				"The \"filters\" property is not a map.")
		}
		filterMap, err := ResolverMapFromObject(filter)
		if err != nil {
			return result, WrapError(
				err, "Could not load filter map from JSON.")
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
		return result, WrappedError(
			"The \"url\" property is missing.")
	}
	url, ok := urlObj.(string)
	if !ok {
		return result, WrappedError(
			"The \"url\" property is not a string.")
	}
	result.Url = url
	return result, nil
}

// ResolveUrl tries to resolve the URL to a filter based on a shortName. If
// it fails it updates the resolver.json file and tries again
func ResolveUrl(shortName string) (string, error) {
	resolverObj, err := LoadResolverAsMap()
	if err != nil {
		return "", WrapError(err, "Unable to load resolver.json")
	}
	resolver, err := ResolverFromObject(resolverObj)
	if err != nil {
		return "", WrapError(err, "Unable to load resolver.json")
	}
	filterMap, ok := resolver.Filters[shortName]
	if !ok {
		return "", WrappedErrorf(
			"The filter \"%s\" is not mapped in the resolver.json file.",
			shortName)
	}
	return filterMap.Url, nil
}
