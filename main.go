package main

import (
	"os"

	color2 "github.com/fatih/color"

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
		Writer:    color2.Output,
		ErrWriter: color2.Error,
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

					return src.RunProfile(profile)
				},
			},
			{
				Name:  "install",
				Usage: "Installs dependencies into the .regolith folder.",
				Action: func(c *cli.Context) error {
					initRegolith(debug)
					src.InstallDependencies()
					return nil
				},
			},
			{
				Name:  "init",
				Usage: "Initialize a Regolith project in the current directory.",
				Action: func(c *cli.Context) error {
					initRegolith(debug)
					src.InitializeRegolithProject(src.StringArrayContains(c.FlagNames(), "force"))
					return nil
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "Force the operation, overriding potential safeguards.",
					},
				},
			},
		},
	}).Run(os.Args)
	if err != nil {
		src.Logger.Error(err)
	}
}

func initRegolith(debug bool) {
	//goland:noinspection GoBoolExpressions
	src.InitLogging(buildSource == "DEV" || debug)
	src.RegisterFilters()
}
