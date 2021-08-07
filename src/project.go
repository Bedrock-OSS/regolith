package src

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"

	"github.com/fatih/color"
)

const ManifestName = "regolith.json"

type Project struct {
	Profiles map[string]Profile `json:"profiles"`
}

type Profile struct {
	Unsafe  bool     `json:"unsafe"`
	Filters []Filter `json:"filters"`
}
type Filter struct {
	Name      string   `json:"name"`
	Location  string   `json:"location"`
	RunWith   string   `json:"run_with"`
	Arguments []string `json:"arguments"`
}

func IsConfigExists() bool {
	info, err := os.Stat(ManifestName)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func LoadConfig() Project {
	file, err := ioutil.ReadFile(ManifestName)
	if err != nil {
		log.Fatal(color.RedString("Couldn't find %s! Consider running 'regolith init'", ManifestName))
	}
	var result Project
	err = json.Unmarshal(file, &result)
	if err != nil {
		log.Fatal(color.RedString("Couldn't load %s: ", ManifestName), err)
	}
	return result
}

func InstallDependencies() {
	log.Println(color.GreenString("Installing dependencies..."))
	log.Println(color.YellowString("Warning: This may take a while..."))

	err := os.Mkdir(".regolith/cache", 777)
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

func InstallDependency(name string) error {
	log.Println(color.GreenString("Installing dependency %s...", name))
	// TODO!
	return nil
}

func InitializeRegolithProject(isForced bool) bool {

	// Do not attempt to initialize if project is already initialized (can be forced)
	if !isForced && IsConfigExists() {
		log.Fatal(color.RedString("Could not initialize Regolith project. File %s already exists.", ManifestName))
		return false
	} else {
		log.Println(color.GreenString("Initializing Regolith project..."))

		if isForced {
			log.Println(color.YellowString("Warning: Initialization forced. Data may be lost."))
		}

		// Delete old configuration
		err := os.Remove(ManifestName)

		// Create new configuration
		file, err := os.Create(ManifestName)
		if err != nil {
			log.Fatal(color.RedString("Could not create %s: ", ManifestName), err)
		}
		defer file.Close()

		// Write default configuration
		var jsonData interface{}
		json.Unmarshal(json.RawMessage(`{"profiles":{"default":{"unsafe":false,"filters":[]}}}`), &jsonData)
		jsonBytes, _ := json.MarshalIndent(jsonData, "", "\t")
		file.Write(jsonBytes)
		log.Println(color.GreenString("Regolith project initialized."))

		// Create .regolith folder
		err = os.Mkdir(".regolith", 777)
		if err != nil {
			log.Fatal(color.RedString("Could not create .regolith folder: "), err)
		}

		return true
	}
}
