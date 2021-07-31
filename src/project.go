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

func InitializeRegolithProject(isForced bool) bool {

	// Do not attempt to initialize if project is already initialized (can be forced)
	if !isForced && IsConfigExists() {
		log.Fatal(color.RedString("Could not initialize Regolith project. File %s already exists.", MANIFEST_NAME))
		return false
	} else {
		log.Println(color.GreenString("Initializing Regolith project..."))

		if isForced {
			log.Println(color.YellowString("Warning: Initialization forced. Data may be lost."))
		}

		// Delete old configuration
		err := os.Remove(MANIFEST_NAME)
		if err != nil {
			log.Fatal(color.RedString("Could not delete %s: ", MANIFEST_NAME), err)
		}

		// Create new configuration
		file, err := os.Create(MANIFEST_NAME)
		if err != nil {
			log.Fatal(color.RedString("Could not create %s: ", MANIFEST_NAME), err)
		}
		defer file.Close()

		// Write default configuration
		file.WriteString("{\"profiles\":{\"default\":{\"unsafe\":false,\"filters\":[]}}}")
		log.Println(color.GreenString("Regolith project initialized."))
		return true
	}
}
