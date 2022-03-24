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
					args := c.Args().Slice()
					var profile string
					if len(args) != 0 {
						profile = args[0]
					}
					return regolith.Run(profile, debug)
				},
			},
			{
				Name: "update",
				Usage: `It updates filters listed in "filters" parameter. The
				names of the filters must be already present in the
				filtersDefinitions list in the config.json file.`,
				Action: func(c *cli.Context) error {
					return regolith.Update(c.Args().Slice(), debug)
				},
			},
			{
				Name: "update-all",
				Usage: `It updates all of the filters listed in the
				filtersDefinitions which aren't version locked.`,
				Action: func(c *cli.Context) error {
					return regolith.UpdateAll(debug)
				},
			},
			{
				Name:  "install-all",
				Usage: `Installs all of the filters from filtersDefintions of config.json file and their dependencies.`,
				Action: func(c *cli.Context) error {
					force := c.Bool("force")
					return regolith.InstallAll(force, debug)
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
				Name:  "install",
				Usage: `Installs specific filters from the Internet and adds them to the filtersDefinitions list in the config.json file.`,
				Action: func(c *cli.Context) error {
					force := c.Bool("force")
					filters := c.Args().Slice()
					// Filter out the --force flag
					for i, f := range filters {
						if f == "--force" {
							filters = append(filters[:i], filters[i+1:]...)
							break
						}
					}
					return regolith.Install(filters, force, debug)
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
				Name:  "init",
				Usage: "Initialize a Regolith project in the current directory.",
				Action: func(c *cli.Context) error {
					return regolith.Init(debug)
				},
			},
			{
				Name:  "clean",
				Usage: "Cleans cache from the .regolith folder.",
				Action: func(c *cli.Context) error {
					return regolith.Clean(debug)
				},
			},
			{
				Name:  "unlock",
				Usage: "Unlocks Regolith, to enable use of Remote and Local filters.",
				Action: func(c *cli.Context) error {
					return regolith.Unlock(debug)
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
		regolith.Logger.Debug(result.Err)
	} else if result.ShouldUpdate {
		_, _ = fmt.Fprintln(color.Output, color.GreenString("New version available!"))
		_, _ = fmt.Fprintln(color.Output, color.GreenString(*result.Url))
	}
}
