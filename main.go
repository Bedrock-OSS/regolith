package main

import (
	"os"

	"github.com/fatih/color"

	"bedrock-oss.github.com/regolith/regolith"
	"github.com/urfave/cli/v2"
)

var (
	commit      string
	version     = "unversioned"
	date        string
	buildSource = "DEV"
)

func main() {
	regolith.CustomHelp()
	var debug bool
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
		Writer:    color.Output,
		ErrWriter: color.Error,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:        "debug",
				Aliases:     []string{"d"},
				Usage:       "Enables debugging.",
				Destination: &debug,
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "run",
				Usage: "Runs Regolith, and generates cooked RP and BP, which will be exported per the config.",
				Action: func(c *cli.Context) error {
					initRegolith(debug)
					args := c.Args().Slice()
					var profile string

					if len(args) == 0 {
						profile = "dev"
					} else {
						profile = args[0]
					}

					return regolith.RunProfile(profile)
				},
			},
			{
				Name:  "install",
				Usage: "Installs dependencies into the .regolith folder.",
				Action: func(c *cli.Context) error {
					initRegolith(debug)
					return regolith.InstallDependencies()
				},
			},
			{
				Name:  "init",
				Usage: "Initialize a Regolith project in the current directory.",
				Action: func(c *cli.Context) error {
					initRegolith(debug)
					return regolith.InitializeRegolithProject(regolith.StringArrayContains(c.FlagNames(), "force"))
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "Force the operation, overriding potential safeguards.",
					},
				},
			},
			{
				Name:  "clean",
				Usage: "Cleans cache from the .regolith folder.",
				Action: func(c *cli.Context) error {
					initRegolith(debug)
					return regolith.CleanCache()
				},
			},
		},
	}).Run(os.Args)
	if err != nil {
		regolith.Logger.Error(err)
	} else {
		regolith.InitLogging(false)
		regolith.Logger.Info(color.GreenString("Finished"))
	}
}

func initRegolith(debug bool) {
	//goland:noinspection GoBoolExpressions
	regolith.InitLogging(debug)
	go regolith.CheckUpdate(version)
	regolith.RegisterFilters()
}
