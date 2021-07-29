package main

import (
	"bedrock-oss.github.com/regolith/src"
	"fmt"
	"github.com/urfave/cli/v2"
	"os"
)

var (
	commit      string
	version     = "unversioned"
	date        string
	buildSource = "DEV"
)

func main() {
	src.CustomHelp()
	err := (&cli.App{
		Name:                 "Regolith",
		Usage:                "A bedrock addon compiler pipeline",
		EnableBashCompletion: true,
		Version:              version,
		Metadata: map[string]interface{}{
			"Commit":      commit,
			"Date":        date,
			"BuildSource": buildSource,
		},
		Commands: []*cli.Command{
			{
				Name:  "build",
				Usage: "Placeholder",
				Action: func(c *cli.Context) error {
					src.LoadConfig()
					return nil
				},
			},
			{
				Name:  "install",
				Usage: "Placeholder",
				Action: func(c *cli.Context) error {
					fmt.Println("Placeholder")
					return nil
				},
			},
		},
	}).Run(os.Args)
	if err != nil {
		panic(err)
	}
}
