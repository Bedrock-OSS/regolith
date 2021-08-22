package src

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/fatih/color"
	getter "github.com/hashicorp/go-getter"
)

func UrlToPath(url string) string {
	return ".regolith/cache/" + url
}

func FilterNameToUrl(name string) string {
	return "github.com/Bedrock-OSS/regolith-filters//" + name
}

func IsRemoteFilterCached(url string) bool {

	_, err := os.Stat(UrlToPath(url))
	if err != nil {
		return false
	}
	return true
}

func DownloadFileTest() {
	fmt.Println("HELLO WORD!")
	fileUrl := "github.com/Bedrock-OSS/regolith-filters//texture_list"

	getter.Get("./.regolith/cache/test", fileUrl)
}

func GatherDependencies() []string {
	project := LoadConfig()
	var dependencies []string
	for _, profile := range project.Profiles {
		for _, filter := range profile.Filters {
			if filter.Url != "" {
				dependencies = append(dependencies, filter.Url)
			}

			if filter.Filter != "" {
				dependencies = append(dependencies, FilterNameToUrl(filter.Filter))
			}
		}
	}
	return dependencies
}

func InstallDependencies() {
	log.Println(color.GreenString("Installing dependencies..."))
	log.Println(color.YellowString("Warning: This may take a while..."))

	err := os.MkdirAll(".regolith/cache", 0777)
	if err != nil {
		log.Fatal(color.RedString("Could not create .regolith/cache: "), err)
	}

	dependencies := GatherDependencies()
	for _, dependency := range dependencies {
		err := InstallDependency(dependency)
		if err != nil {
			log.Fatal(color.RedString("Could not install dependency %s: ", dependency), err)
		}
	}

	log.Println(color.GreenString("Dependencies installed."))
}

func InstallDependency(url string) error {
	log.Println(color.GreenString("Installing dependency %s...", url))

	// Install the url into the cache folder

	path := UrlToPath(url)
	err := getter.Get(path, url)

	if err != nil {
		log.Fatal(color.RedString("Could not install dependency %s: ", url), err)
	}
	file, err := ioutil.ReadFile(path + "/filter.json")

	if err != nil {
		log.Fatal(color.RedString("Couldn't find %s/filter.json!", path), err)
	}

	var result Profile
	err = json.Unmarshal(file, &result)
	if err != nil {
		log.Fatal(color.RedString("Couldn't load %s/filter.json: ", path), err)
	}

	// Check whether filter, that the user wants to run meet the requirements
	checked := map[string]bool{}
	for _, filter := range result.Filters {
		if filter.RunWith != "" {
			if c, ok := checked[filter.RunWith]; ok || !c {
				if f, ok := FilterTypes[filter.RunWith]; ok {
					checked[filter.RunWith] = true
					f.install(filter, path)
				} else {
					Logger.Warnf("Filter type '%s' not supported", filter.RunWith)
				}
			}
		}
	}

	return nil
}
