package regolith

import (
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

func decodeSingleFilter(obj map[string]interface{}) (*ManisfestDeclaredFilter, error) {
	result := ManisfestDeclaredFilter{}
	var path *string = nil

	pathValue, ok := obj["path"].(string)
	if ok {
		path = &pathValue
	}

	versions, ok := obj["versions"].(map[string]interface{})

	if !ok {
		return nil, burrito.WrappedErrorf(jsonPropertyMissingError, "versions")
	}

	result.Path = path
	result.Versions = make(map[string][]DeclaredFilterInner, len(versions))

	for version, versionObj := range versions {
		versionInformation, ok := versionObj.(map[string]interface{})

		if !ok {
			return nil, burrito.WrappedErrorf(jsonPropertyTypeError, version, "object")
		}

		if !semver.IsValid("v" + version) {
			return nil, burrito.WrappedErrorf("Malformed semver. The semver for a filter must be a valid semver. Current: %s", version)
		}

		urls, ok := versionInformation["urls"].([]map[string]interface{})

		if !ok {
			return nil, burrito.WrappedErrorf(jsonPropertyMissingError, "urls")
		}

		for _, inner := range urls {
			var matcher *OsMatcher = nil

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
