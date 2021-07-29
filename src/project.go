package src

import (
	"encoding/json"
	"github.com/fatih/color"
	"io/ioutil"
	"log"
)

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

func LoadConfig() Project {
	file, err := ioutil.ReadFile("manifest.json")
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
