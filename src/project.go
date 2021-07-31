package src

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"github.com/fatih/color"
)

const MANIFEST_NAME = "regolith.json"

type Project struct {
	Profiles map[string]Profile `json:"profiles"`
}

type Profile struct {
	Unsafe  bool     `json:"unsafe"`
	Filters []Filter `json:"filters"`
}

type Filter struct {
	Definition string `json:"definition"`
}

func IsConfigExists() bool {
	info, err := os.Stat(MANIFEST_NAME)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func LoadConfig() Project {
	file, err := ioutil.ReadFile(MANIFEST_NAME)
	if err != nil {
		log.Fatal(color.RedString("Couldn't find manifest.json!"))
	}
	var result Project
	err = json.Unmarshal(file, &result)
	if err != nil {
		log.Fatal(color.RedString("Couldn't load manifest.json: "), err)
	}
	return result
}

func InitializeRegolithProject() bool {
	if IsConfigExists() {
		log.Fatal(color.RedString("Could not initialize Regolith project. File %s already exists.", MANIFEST_NAME))
		return false
	} else {
		log.Println(color.GreenString("Initializing Regolith project..."))
		file, err := os.Create(MANIFEST_NAME)
		if err != nil {
			log.Fatal(color.RedString("Could not create manifest.json: "), err)
		}
		defer file.Close()
		file.WriteString("{\"profiles\":{\"default\":{\"unsafe\":false,\"filters\":[]}}}")
		log.Println(color.GreenString("Regolith project initialized."))
		return true
	}
}
