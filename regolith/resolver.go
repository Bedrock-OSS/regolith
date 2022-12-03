package regolith

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Bedrock-OSS/go-burrito/burrito"

	"github.com/hashicorp/go-getter"
)

const (
	// regolithConfigPath is a path to the regolith config relative to
	// UserCacheDir()
	regolithConfigPath = "regolith"
	// resolverUrl is  the default URL to the resolver.json file
	resolverUrl = "github.com/Bedrock-OSS/regolith-filter-resolver/resolver.json"
)

type ResolverMapItem struct {
	Url string `json:"url"`
}

// resolverMap is a lazy loaded map with combined resolver.json files. This
// map should never be modified directly. Use getResolversAsMap() instead.
var resolverMap *map[string]ResolverMapItem

// GetRegolithAppDataPath returns path to the regolith files in user app data
func GetRegolithAppDataPath() (string, error) {
	path, err := os.UserCacheDir()
	if err != nil {
		return "", burrito.WrappedError(osUserCacheDirError)
	}
	return filepath.Join(path, regolithConfigPath), nil
}

// resolveResolverUrl resolves the resolver URL from the short name to a full
// URL that can be used by go-getter
func resolveResolverUrl(url string) (string, error) {
	urlParts := strings.Split(url, "/")
	if len(urlParts) < 4 {
		return "", burrito.WrappedErrorf(
			"Incorrect URL format.\n" +
				"Expected format:" +
				"github.com/<user-name>/<repo-name>/<path-to-the-resolver-file>")
	}
	repoUrl := strings.Join(urlParts[0:3], "/")
	path := strings.Join(urlParts[3:], "/")
	sha, err := GetHeadSha(repoUrl)
	if err != nil {
		return "", burrito.WrapError(
			err,
			"Failed to get the HEAD of the repository with the resolver.json file")
	}
	return fmt.Sprintf("git::https://%s/%s?ref=%s", repoUrl, path, sha), nil
}

// DownloadResolverMaps downloads the resolver.json files
func DownloadResolverMaps() error {
	Logger.Info("Downloading resolvers")

	// Define function to download group of resolvers
	downloadResolvers := func(urls []string, root string) error {
		targetPath := filepath.Join(root, "resolvers")
		tmpPath := filepath.Join(root, ".resolvers-tmp")
		tmpResolversPath := filepath.Join(tmpPath, "resolvers")
		tmpUndoPath := filepath.Join(tmpPath, "undo")
		// Create target directory if not exists
		err := os.MkdirAll(targetPath, 0755)
		if err != nil {
			return burrito.WrapErrorf(err, osMkdirError, targetPath)
		}
		// Prepare the temporary directory
		err = os.RemoveAll(tmpPath)
		if err != nil {
			return burrito.WrapErrorf(err, osRemoveError, tmpPath)
		}
		err = os.MkdirAll(tmpResolversPath, 0755)
		if err != nil {
			return burrito.WrapErrorf(err, osMkdirError, tmpResolversPath)
		}
		err = os.MkdirAll(tmpUndoPath, 0755)
		if err != nil {
			return burrito.WrapErrorf(err, osMkdirError, tmpUndoPath)
		}
		defer os.RemoveAll(tmpPath) // Schedule for deletion
		// Prepare the revertibleFsOperations object
		revertibleOps, err := NewRevertibleFsOperations(tmpUndoPath)
		if err != nil {
			return burrito.WrapErrorf(err, newRevertibleFsOperationsError, tmpUndoPath)
		}
		defer revertibleOps.Close() // Must be called before os.RemoveAll(tmpPath)
		// Download the resolvers to the tmp path
		for i, shortUrl := range urls {
			// Get the save path and resolve the URL
			savePath := filepath.Join(
				tmpResolversPath, fmt.Sprintf("resolver_%d.json", i))
			url, err := resolveResolverUrl(shortUrl)
			if err != nil {
				return burrito.WrapError(
					err,
					"Failed to resolve the URL of the resolver file for the download.\n"+
						"Short URL: "+shortUrl)
			}
			Logger.Debugf("Downloading resolver using URL: %s", url)
			err = getter.GetFile(savePath, url)
			if err != nil {
				return burrito.WrapErrorf(err, "Failed to download the file.\nURL: %s", url)
			}
			// Add "url" property to the resolver file
			fileData := make(map[string]interface{})
			f, err := os.ReadFile(savePath)
			if err != nil {
				return burrito.WrapErrorf(err, fileReadError, savePath)
			}
			err = json.Unmarshal(f, &fileData)
			if err != nil {
				return burrito.WrapErrorf(err, jsonUnmarshalError, savePath)
			}
			fileData["url"] = shortUrl
			// Save the file with the "url" property
			f, _ = json.MarshalIndent(fileData, "", "\t")
			err = os.WriteFile(savePath, f, 0644)
			if err != nil {
				return burrito.WrapErrorf(err, fileWriteError, savePath)
			}
		}
		// Make sure that the target directory is empty
		err = revertibleOps.DeleteDir(targetPath)
		if err != nil {
			revertibleOps.Undo() // Don't handle the error. I don't care.
			return burrito.WrapErrorf(err, osRemoveError, targetPath)
		}
		err = revertibleOps.MkdirAll(targetPath)
		if err != nil {
			revertibleOps.Undo() // Don't handle the error. I don't care.
			return burrito.WrapErrorf(err, osMkdirError, targetPath)
		}
		// Move the resolvers to the target path
		err = revertibleOps.MoveOrCopyDir(tmpResolversPath, targetPath)
		if err != nil {
			revertibleOps.Undo() // Don't handle the error. I don't care.
			return burrito.WrapErrorf(err, moveOrCopyError, tmpResolversPath, targetPath)
		}
		return nil
	}
	// Download the global resolvers
	appDataPath, err := GetRegolithAppDataPath()
	if err != nil {
		return burrito.WrapError(err, getRegolithAppDataPathError)
	}
	globalUserConfig, err := getGlobalUserConfig()
	globalUserConfig.fillDefaults() // The file must have the default resolver URL
	if err != nil {
		return burrito.WrapError(err, getUserConfigError)
	}
	err = downloadResolvers(globalUserConfig.Resolvers, appDataPath)
	if err != nil {
		return burrito.WrapError(err, "Failed to download the resolvers")
	}
	return nil
}

// getResolversMap downloads and lazily loads the resolverMap from the
// resolver.json files if it is already loaded, it returns the map
func getResolversMap() (*map[string]ResolverMapItem, error) {
	if resolverMap != nil {
		return resolverMap, nil
	}
	err := DownloadResolverMaps()
	if err != nil {
		Logger.Warnf(
			"Failed to download resolver map: %s", err.Error())
	}
	result := make(map[string]ResolverMapItem)
	// Load all resolver files into a map, where the ke is the URL of the resovler
	// file and the value is the content of the file. Based on this map and
	// the combined user config, the final resolver map is created.
	resolvers := make(map[string]interface{})
	loadResolversFromPath := func(path string) error {
		globalResolvers, err := os.ReadDir(path)
		if err != nil {
			return burrito.WrapErrorf(
				err, "Failed to list files in the directory.\nPath: %s",
				path)
		}
		for _, resolver := range globalResolvers {
			if resolver.IsDir() {
				continue
			}
			filePath := filepath.Join(path, resolver.Name())
			f, err := os.ReadFile(filePath)
			if err != nil {
				return burrito.WrapErrorf(err, fileReadError, filePath)
			}
			resolverData := make(map[string]interface{})
			err = json.Unmarshal(f, &resolverData)
			if err != nil {
				return burrito.WrapErrorf(err, jsonUnmarshalError, filePath)
			}
			url, ok := resolverData["url"].(string)
			if !ok {
				return burrito.WrapErrorf(
					err,
					"Failed to get the URL of the resolver file.\nPath: %s",
					filePath)
			}
			resolvers[url] = resolverData
		}
		return nil
	}
	// Load the global resolvers
	appDataPath, err := GetRegolithAppDataPath()
	if err != nil {
		return nil, burrito.WrapError(err, getRegolithAppDataPathError)
	}
	globalResolversPath := filepath.Join(appDataPath, "resolvers")
	err = loadResolversFromPath(globalResolversPath)
	if err != nil {
		return nil, burrito.WrapError(err, "Failed to load the global resolvers")
	}
	// Get user config to access the list of resolvers
	userConfig, err := getCombinedUserConfig()
	if err != nil {
		return nil, burrito.WrapError(err, getUserConfigError)
	}
	// Create the final resolver map
	for _, resolverUrl := range userConfig.Resolvers {
		resolverData, ok := resolvers[resolverUrl].(map[string]interface{})
		if !ok {
			return nil, burrito.WrapErrorf(
				err, "Failed to get the resolver data.\nURL: %s", resolverUrl)
		}
		resolverResovlersData, ok := resolverData["filters"].(map[string]interface{})
		if !ok {
			return nil, burrito.WrapErrorf(
				err, "Failed load resolvers from the resolver file.\nURL: %s",
				resolverUrl)
		}
		for key, value := range resolverResovlersData {
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
func ResolveUrl(shortName string) (string, error) {
	const resolverLoadErrror = "Unable to load the name to URL resolver map."
	resolver, err := getResolversMap()
	if err != nil {
		return "", burrito.WrapError(err, resolverLoadErrror)
	}
	filterMap, ok := (*resolver)[shortName]
	if !ok {
		return "", burrito.WrappedErrorf(
			"The filter doesn't have known mapping to URL in the URL "+
				"resolver.\n"+
				"Filter name: %s\n",
			shortName)
	}
	return filterMap.Url, nil
}
