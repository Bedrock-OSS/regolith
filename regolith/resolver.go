package regolith

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Bedrock-OSS/go-burrito/burrito"
	"github.com/paul-mannino/go-fuzzywuzzy"
)

const (
	// resolverUrl is  the default URL to the resolver.json file
	resolverUrl = "github.com/Bedrock-OSS/regolith-filter-resolver/resolver.json"
)

type ResolverMapItem struct {
	Url string `json:"url"`
}

// resolverMap is a lazy-loaded map with combined resolver.json files. This
// map should never be modified directly. Use getResolversAsMap() instead.
var resolverMap *map[string]ResolverMapItem

// resolveResolverUrl resolves the resolver URL from the short name to a full
// URL that can be used by go-getter
func resolveResolverUrl(url string) (string, string, error) {
	urlParts := strings.Split(url, "/")
	if len(urlParts) < 4 {
		return "", "", burrito.WrappedErrorf(
			"Incorrect URL format.\n" +
				"Expected format:" +
				"github.com/<user-name>/<repo-name>/<path-to-the-resolver-file>")
	}
	repoUrl := strings.Join(urlParts[0:3], "/")
	path := strings.Join(urlParts[3:], "/")
	return fmt.Sprintf("https://%s", repoUrl), path, nil
}

// DownloadResolverMaps downloads the resolver repositories and returns lists of urls and paths
func DownloadResolverMaps(forceUpdate bool) ([]string, []string, error) {
	// Download the global resolvers
	globalUserConfig, err := getGlobalUserConfig()
	globalUserConfig.fillDefaults() // The file must have the default resolver URL
	if err != nil {
		return nil, nil, burrito.WrapError(err, getUserConfigError)
	}
	if len(globalUserConfig.Resolvers) == 0 {
		return nil, nil, nil
	}
	config, err := getCombinedUserConfig()
	if err != nil {
		return nil, nil, burrito.WrapErrorf(err, getUserConfigError)
	}
	cooldown, err := time.ParseDuration(*config.ResolverCacheUpdateCooldown)
	if err != nil {
		return nil, nil, burrito.WrapErrorf(err, "Failed to parse resolver cache update cooldown.\nCooldown: %s", *config.ResolverCacheUpdateCooldown)
	}
	MeasureStart("Prepare for resolvers download")
	targetPath, err := getResolverCache(globalUserConfig.Resolvers[0])
	if err != nil {
		return nil, nil, burrito.WrapErrorf(err, resolverPathCacheError, globalUserConfig.Resolvers[0])
	}
	// Create resolver cache directory if not exists
	dir, _ := filepath.Split(targetPath)
	err = os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, nil, burrito.WrapErrorf(err, osMkdirError, targetPath)
	}
	resolverFilePaths := make([]string, len(globalUserConfig.Resolvers))
	// Download the resolvers to the cache path
	for i, shortUrl := range globalUserConfig.Resolvers {
		// Get the save path and resolve the URL
		cachePath, err := getResolverCache(shortUrl)
		if err != nil {
			return nil, nil, burrito.WrapErrorf(err, resolverPathCacheError, shortUrl)
		}
		url, path, err := resolveResolverUrl(shortUrl)
		if err != nil {
			return nil, nil, burrito.WrapErrorf(err, resolverResolveUrlError, shortUrl)
		}
		joinedPath := filepath.Join(cachePath, path)
		pathCheck := func() error {
			info, err := os.Stat(joinedPath)
			if err != nil {
				if os.IsNotExist(err) {
					return burrito.WrapErrorf(err, "Resolver file does not exist.\nPath: %s", joinedPath)
				} else {
					return burrito.WrapErrorf(err, "Failed to get resolver file info.\nPath: %s", joinedPath)
				}
			} else if info.IsDir() {
				return burrito.WrapErrorf(err, "Resolver file is a directory.\nPath: %s", joinedPath)
			}
			resolverFilePaths[i] = joinedPath
			return nil
		}
		// If the repo exist, pull it
		stat, err := os.Stat(filepath.Join(cachePath, ".git"))
		if err == nil && stat.IsDir() {
			info, _ := os.Stat(cachePath)
			if forceUpdate || info.ModTime().Before(time.Now().Add(cooldown*-1)) {
				Logger.Infof("Updating resolver %s", shortUrl)
				MeasureStart("Pull repository %s", shortUrl)
				output, err := RunGitProcess([]string{"pull"}, cachePath)
				MeasureEnd()
				err = os.Chtimes(cachePath, time.Now(), time.Now())
				if err != nil {
					Logger.Debugf("Failed to update cache file modification time.\nPath: %s", cachePath)
				}
				// If pull failed, delete the repo and clone it again
				if err != nil {
					Logger.Debug(strings.Join(output, "\n"))
					Logger.Warnf("Failed to pull repository, recreating repository.\nURL: %s", url)
					err = os.RemoveAll(cachePath)
				} else {
					err := pathCheck()
					if err != nil {
						return nil, nil, burrito.PassError(err)
					}
					continue
				}
			} else {
				err := pathCheck()
				if err != nil {
					return nil, nil, burrito.PassError(err)
				}
				continue
			}
		}
		err = os.MkdirAll(cachePath, 0755)
		if err != nil {
			return nil, nil, burrito.WrapErrorf(err, osMkdirError, cachePath)
		}
		Logger.Infof("Downloading resolver %s", shortUrl)
		MeasureStart("Clone repository %s", shortUrl)
		output, err := RunGitProcess([]string{"clone", url, ".", "--depth", "1"}, cachePath)
		if err != nil {
			Logger.Error(strings.Join(output, "\n"))
			return nil, nil, burrito.WrapErrorf(err, "Failed to clone repository.\nURL: %s", url)
		}
		MeasureEnd()
		err = os.Chtimes(cachePath, time.Now(), time.Now())
		if err != nil {
			Logger.Debugf("Failed to update cache file modification time.\nPath: %s", cachePath)
		}
		err = pathCheck()
		if err != nil {
			return nil, nil, burrito.PassError(err)
		}
	}
	if err != nil {
		return nil, nil, burrito.WrapError(err, "Failed to download the resolvers")
	}
	return globalUserConfig.Resolvers, resolverFilePaths, nil
}

// getResolversMap downloads and lazily loads the resolverMap from the
// resolver.json files if it is already loaded, it returns the map
func getResolversMap(refreshResolvers bool) (*map[string]ResolverMapItem, error) {
	if resolverMap != nil {
		return resolverMap, nil
	}
	urls, resolvedPaths, err := DownloadResolverMaps(refreshResolvers)
	if err != nil {
		Logger.Warnf(
			"Failed to download resolver map: %s", err.Error())
	}
	result := make(map[string]ResolverMapItem)
	// Load all resolver files into a map, where the ke is the URL of the resolver
	// file and the value is the content of the file. Based on this map and
	// the combined user config, the final resolver map is created.
	resolvers := make(map[string]interface{})
	loadResolversFromPath := func(urls, paths []string) error {
		for i, path := range paths {
			f, err := os.ReadFile(path)
			if err != nil {
				return burrito.WrapErrorf(err, fileReadError, path)
			}
			resolverData := make(map[string]interface{})
			err = json.Unmarshal(f, &resolverData)
			if err != nil {
				return burrito.WrapErrorf(err, jsonUnmarshalError, path)
			}
			resolvers[urls[i]] = resolverData
		}
		return nil
	}
	err = loadResolversFromPath(urls, resolvedPaths)
	if err != nil {
		return nil, burrito.WrapError(err, "Failed to load the global resolvers")
	}
	// Create the final resolver map
	for _, resolverUrl := range urls {
		resolverData, ok := resolvers[resolverUrl].(map[string]interface{})
		if !ok {
			return nil, burrito.WrapErrorf(
				err, "Failed to get the resolver data.\nURL: %s", resolverUrl)
		}
		resolverResolversData, ok := resolverData["filters"].(map[string]interface{})
		if !ok {
			return nil, burrito.WrapErrorf(
				err, "Failed load resolvers from the resolver file.\nURL: %s",
				resolverUrl)
		}
		for key, value := range resolverResolversData {
			castValue, ok := value.(map[string]interface{})
			if !ok {
				return nil, burrito.WrapErrorf(
					err, "Invalid resolver data.\nURL: %s",
					resolverUrl)
			}
			result[key], err = ResolverMapFromObject(castValue)
			if err != nil {
				return nil, burrito.WrapErrorf(
					err, "Invalid resolver data.\nURL: %s",
					resolverUrl)
			}
		}
	}
	resolverMap = &result
	return resolverMap, nil
}

func ResolverMapFromObject(obj map[string]interface{}) (ResolverMapItem, error) {
	result := ResolverMapItem{}
	// Url
	urlObj, ok := obj["url"]
	if !ok {
		return result, burrito.WrappedErrorf(jsonPropertyMissingError, "url")
	}
	url, ok := urlObj.(string)
	if !ok {
		return result, burrito.WrappedErrorf(jsonPropertyTypeError, "url", "string")
	}
	result.Url = url
	return result, nil
}

// ResolveUrl tries to resolve the URL to a filter based on a shortName. If
// it fails it updates the resolver.json file and tries again
func ResolveUrl(shortName string, refreshResolvers bool) (string, error) {
	const resolverLoadError = "Unable to load the name to URL resolver map."
	resolver, err := getResolversMap(refreshResolvers)
	if err != nil {
		return "", burrito.WrapError(err, resolverLoadError)
	}
	filterMap, ok := (*resolver)[shortName]
	if !ok {
		// Try to find a close match
		keys := make([]string, 0, len(*resolver))
		for k := range *resolver {
			keys = append(keys, k)
		}
		find, _ := fuzzy.Extract(shortName, keys, 5)
		if find.Len() > 0 {
			return "", burrito.WrappedErrorf(
				"The filter doesn't have known mapping to URL in the URL "+
					"resolver.\n"+
					"Filter name: %s\n"+
					"Did you mean \"%s\"?",
				shortName, find[0].Match)
		}
		return "", burrito.WrappedErrorf(
			"The filter doesn't have known mapping to URL in the URL "+
				"resolver.\n"+
				"Filter name: %s",
			shortName)
	}
	return filterMap.Url, nil
}
