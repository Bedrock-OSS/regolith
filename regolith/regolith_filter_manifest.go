package regolith

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"runtime"
	"sort"
	"strings"

	"net/http"

	"github.com/Bedrock-OSS/go-burrito/burrito"
	"golang.org/x/mod/semver"
)

type OsMatcher struct {
	ExpectedOs   *string `json:"os,omitempty"`
	ExpectedArch *string `json:"arch,omitempty"`
}

type DeclaredFilterInner struct {
	Url  string     `json:"url,omitempty"`
	When *OsMatcher `json:"when,omitempty"`
}

type ManisfestDeclaredFilter struct {
	Path     *string                          `json:"path,omitempty"`
	Versions map[string][]DeclaredFilterInner `json:"versions,omitempty"`
}

type RepositoryManifest struct {
	FormatVersion string                             `json:"formatVersion,omitempty"`
	Filters       map[string]ManisfestDeclaredFilter `json:"filters,omitempty"`
}

func (manifest *RepositoryManifest) FindPath(filterId string) (*string, error) {
	filter, ok := manifest.Filters[filterId]

	if !ok {
		return nil, burrito.WrappedErrorf(`Invalid filter name requested from repository:
Requested Name: %s`, filterId)
	}

	out := filter.Path

	if out == nil {
		return nil, burrito.WrappedErrorf("Requested path on a filter which does not have a path specified:\nFilter name: %s", filterId)
	}

	return out, nil
}

func (manifest *RepositoryManifest) IsUrlBased(filterId string) (*bool, error) {
	filter, ok := manifest.Filters[filterId]

	if !ok {
		return nil, burrito.WrappedErrorf("Invalid filter name tested: %s", filterId)
	}

	if filter.Path == nil {
		v := true
		return &v, nil
	} else {
		v := false
		return &v, nil
	}
}

func (manifest *RepositoryManifest) ResolveUrlForFilter(filterId, version string) (*string, error) {
	if value, err := manifest.IsUrlBased(filterId); err == nil {
		if !*value {
			return nil, burrito.WrappedErrorf("Filter %s is not URL based", filterId)
		}
	} else {
		return nil, err
	}

	if version != "HEAD" && version != "latest" && !semver.IsValid("v"+version) {
		return nil, burrito.WrappedErrorf("Version for the filter %s is not in one of the valid formats! It must be \"HEAD\", \"latest\", or a valid semver!")
	}

	// We can ignore the ok check since this same check is performed in `IsUrlBased`
	filter := manifest.Filters[filterId]

	// Handles the simple case
	if version != "HEAD" && version != "latest" {
		inners, ok := filter.Versions[version]

		if !ok {
			return nil, burrito.WrappedErrorf("Version %s not found in the manifest for %s", version, filterId)
		}

		return compatableUrl(&inners), nil
	}

	versionList := make([]string, 0)

	for v := range filter.Versions {
		versionList = append(versionList, "v"+v)
	}

	semver.Sort(versionList)
	sort.Sort(sort.Reverse(sort.StringSlice(versionList)))

	for _, v := range versionList {
		versionList := filter.Versions[v[1:]]

		if url := compatableUrl(&versionList); url != nil {
			return url, nil
		}
	}

	return nil, nil
}

func compatableUrl(urls *[]DeclaredFilterInner) *string {
	var ret *string

	for _, f := range *urls {
		if f.When == nil {
			ret = &f.Url
		}
	}

	for _, f := range *urls {
		if f.When == nil {
			continue
		}

		matches := f.When.MatchesCurrentHost()

		if matches {
			ret = &f.Url
			return ret
		}

	}

	return ret
}

func (matcher *OsMatcher) MatchesCurrentHost() bool {
	if matcher.ExpectedArch != nil {
		if *matcher.ExpectedArch != runtime.GOARCH {
			return false
		}
	}

	if matcher.ExpectedOs != nil {
		if *matcher.ExpectedOs != runtime.GOOS {
			return false
		}
	}

	return true
}

func decodeSingleFilter(obj map[string]interface{}) (*ManisfestDeclaredFilter, error) {
	result := ManisfestDeclaredFilter{}
	var path *string = nil

	pathValue, ok := obj["path"].(string)
	if ok {
		path = &pathValue
	}

	versions, ok := obj["versions"].(map[string]interface{})

	result.Path = path
	result.Versions = make(map[string][]DeclaredFilterInner, len(versions))

	if ok {
		for version, versionObj := range versions {
			versionInformation, ok := versionObj.(map[string]interface{})

			if !ok {
				return nil, burrito.WrappedErrorf(jsonPropertyTypeError, version, "object")
			}

			if !semver.IsValid("v" + version) {
				return nil, burrito.WrappedErrorf("Malformed semver. The semver for a filter must be a valid semver. Current: %s", version)
			}

			urls, ok := versionInformation["urls"].([]interface{})

			if !ok {
				return nil, burrito.WrappedErrorf(jsonPropertyMissingError, "urls")
			}

			for _, inner := range urls {
				var matcher *OsMatcher = nil

				inner := inner.(map[string]interface{})

				url, ok := inner["url"].(string)

				if !ok {
					return nil, burrito.WrappedErrorf(jsonPropertyMissingError, "url")
				}

				when, ok := inner["when"].(map[string]interface{})

				if ok {
					var eos *string = nil
					var earch *string = nil

					os, ok := when["os"].(string)

					if ok {
						eos = &os
					}

					arch, ok := when["arch"].(string)

					if ok {
						earch = &arch
					}

					matcher = &OsMatcher{
						ExpectedOs:   eos,
						ExpectedArch: earch,
					}
				}
				result.Versions[version] = append(result.Versions[version], DeclaredFilterInner{
					Url:  url,
					When: matcher,
				})
			}

		}
	}

	return &result, nil
}

func RepositoryManifestFromObject(obj map[string]interface{}) (*RepositoryManifest, error) {
	result := RepositoryManifest{}

	formatVersion, ok := obj["formatVersion"].(string)

	if !ok {
		return nil, burrito.WrappedErrorf(jsonPropertyMissingError, "formatVersion")
	}
	filters, ok := obj["filters"].(map[string]interface{})

	if !ok {
		return nil, burrito.WrappedErrorf(jsonPropertyMissingError, "filters")
	}

	result.FormatVersion = formatVersion
	result.Filters = make(map[string]ManisfestDeclaredFilter, len(filters))

	for n, raw := range filters {
		json, ok := raw.(map[string]interface{})

		if !ok {
			return nil, burrito.WrappedErrorf(jsonPropertyTypeError, "filters", "object")
		}

		filter, err := decodeSingleFilter(json)

		if err != nil {
			return nil, burrito.WrapErrorf(err, "Failed to decode the filter declaration for %s", n)
		}

		result.Filters[n] = *filter
	}

	return &result, nil
}

func ManifestForRepo(url string) (*RepositoryManifest, error) {
	// https://raw.githubusercontent.com/<user-name>/<project-name>/HEAD/regolith_filter_manifest.json is the end result ideally
	// The url passed in should be in the format github.com/<user-name>/<project-name> so its a trivial transformation

	chunks := strings.Split(url, "/")

	if len(chunks) < 2 {
		return nil, burrito.WrappedErrorf("Manifest url has an invalid format! It should be \"github.com/<user-name>/<project-name>\" it was: %s", url)
	}

	projectName := chunks[len(chunks)-1]
	userName := chunks[len(chunks)-2]

	manifestURL := fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/HEAD/regolith_filter_manifest.json", userName, projectName)

	Logger.Debugf("Testing URL %s for a manifest", manifestURL)

	var err error

	result, err := http.Get(manifestURL)

	if err != nil {
		return nil, err
	}

	defer result.Body.Close()

	if result.StatusCode >= 400 {
		// This handles the case when the repository doesnt have a manifest
		return nil, nil
	}

	Logger.Debugf("Found manifest at %s", manifestURL)

	var bytes bytes.Buffer

	_, err = io.Copy(&bytes, result.Body)

	if err != nil {
		return nil, burrito.WrapErrorf(err, "Failed to clone body of %s manifest", url)
	}

	object := make(map[string]interface{})

	err = json.Unmarshal(bytes.Bytes(), &object)

	if err != nil {
		return nil, burrito.WrapErrorf(err, "Failed to decode manifest into json. From the repo: %s", url)
	}

	manifest, err := RepositoryManifestFromObject(object)

	if err != nil {
		return nil, err
	}

	return manifest, nil
}
