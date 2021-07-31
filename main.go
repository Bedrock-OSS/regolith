package main

import (
	"fmt"
	"os"

	"bedrock-oss.github.com/regolith/src"
	"github.com/urfave/cli/v2"
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
			{
				Name:  "init",
				Usage: "Initialize a Regolith project in the current directory.",
				Action: func(c *cli.Context) error {
					src.InitializeRegolithProject(src.StringArrayContains(c.FlagNames(), "force"))

					return nil
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "Force the operateion, overriding potential safeguards.",
					},
				},
			},
			{
				Name:  "childproc",
				Usage: "Running a child-process!",
				Action: func(c *cli.Context) error {
					src.RunChildProc()
					return nil
				},
			},
		},
	}).Run(os.Args)
	if err != nil {
		panic(err)
	}
}