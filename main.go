package main

import (
	"fmt"
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
	status := make(chan regolith.UpdateStatus)
	go regolith.CheckUpdate(version, status)
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
				Usage: "Runs Regolith, and generates compiled RP and BP, which will be exported to the destination specified in the config.",
				Action: func(c *cli.Context) error {
					initRegolith(debug)
					args := c.Args().Slice()
					var profile string

					if len(args) == 0 {
						profile = "dev"
					} else {
						profile = args[0]
					}

					err := regolith.RunProfile(profile)
					if err != nil {
						return regolith.WrapErrorf(err, "Failed to run profile '%s'", profile)
					}
					return nil
				},
			},
			{
				Name: "install",
				Usage: `Installs the filters defined in the 'filter_definitions' of the config file. 
						The '--add' flag can be used to specify a filter by name, which will add the filter to the config file and install it.`,
				Action: func(c *cli.Context) error {
					initRegolith(debug)

					if c.Bool("add") {
						filterNames := c.StringSlice("add")
						err := regolith.AddFilters(filterNames, c.Bool("force"))
						if err != nil {
							return regolith.WrapErrorf(err, "Failed to install filters with 'add': '%v'", filterNames)
						}
					} else {
						configJson, err := regolith.LoadConfigAsMap()
						config, err := regolith.ConfigFromObject(configJson)

						if err != nil {
							return regolith.WrapError(err, "Failed to parse 'config.json' file")
						}
						err = config.InstallFilters(c.Bool("force"))
						if err != nil {
							return regolith.WrapError(err, "could not install filters")
						}
					}
					return nil
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "force",
						Aliases: []string{"f"},
						Usage:   "Force the operation, overriding potential safeguards.",
					},
					&cli.StringSliceFlag{
						Name:    "add",
						Aliases: []string{"a"},
						Usage:   "Specify a remote filter to add to the project and install it.",
					},
				},
			},
			{
				Name:  "init",
				Usage: "Initialize a Regolith project in the current directory.",
				Action: func(c *cli.Context) error {
					initRegolith(debug)
					err := regolith.InitializeRegolithProject(regolith.StringArrayContains(c.FlagNames(), "force"))
					if err != nil {
						return regolith.WrapError(err, "Failed to initialize project.")
					}
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
			{
				Name:  "clean",
				Usage: "Cleans cache from the .regolith folder.",
				Action: func(c *cli.Context) error {
					initRegolith(debug)
					return regolith.CleanCache()
				},
			},
			{
				Name:  "unlock",
				Usage: "Unlocks Regolith, to enable use of Remote and Local filters.",
				Action: func(c *cli.Context) error {
					initRegolith(debug)
					return regolith.Unlock()
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
	result := <-status
	if result.Err != nil {
		regolith.Logger.Warn("Update check failed")
	} else if result.ShouldUpdate {
		_, _ = fmt.Fprintln(color.Output, color.GreenString("New version available!"))
		_, _ = fmt.Fprintln(color.Output, color.GreenString(*result.Url))
	}
}

func initRegolith(debug bool) {
	//goland:noinspection GoBoolExpressions
	regolith.InitLogging(debug)
}
