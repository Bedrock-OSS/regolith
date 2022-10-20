package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"

	"github.com/Bedrock-OSS/regolith/regolith"
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
				Destination: &regolith.Debug,
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
					return regolith.Run(profile, regolith.Debug)
				},
			},
			{
				Name:  "watch",
				Usage: "Watches the project files and runs specified Regolith profile when they change.",
				Action: func(c *cli.Context) error {
					args := c.Args().Slice()
					var profile string
					if len(args) != 0 {
						profile = args[0]
					}
					return regolith.Watch(profile, regolith.Debug)
				},
			},
			{
				Name: "update",
				Usage: `It updates filters listed in "filters" parameter. The
				names of the filters must be already present in the
				filtersDefinitions list in the config.json file.`,
				Action: func(c *cli.Context) error {
					return regolith.Update(c.Args().Slice(), regolith.Debug)
				},
			},
			{
				Name: "update-all",
				Usage: `It updates all of the filters listed in the
				filtersDefinitions which aren't version locked.`,
				Action: func(c *cli.Context) error {
					return regolith.UpdateAll(regolith.Debug)
				},
			},
			{
				Name:  "install-all",
				Usage: `Installs all of the filters from filtersDefintions of config.json file and their dependencies.`,
				Action: func(c *cli.Context) error {
					force := c.Bool("force")
					return regolith.InstallAll(force, regolith.Debug)
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
					return regolith.Install(filters, force, regolith.Debug)
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
					return regolith.Init(regolith.Debug)
				},
			},
			{
				Name:  "clean",
				Usage: "Cleans Regolith cache.",
				Action: func(c *cli.Context) error {
					userCache := c.Bool("user-cache")
					return regolith.Clean(regolith.Debug, userCache)
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "user-cache",
						Aliases: []string{},
						Usage: "Clears data of the projects cached in the " +
							"user app data folder. This is useful to clean " +
							"up leftover files from old projects that use " +
							"the \"useAppData\" option.",
					},
				},
			},
			{
				Name:  "unlock",
				Usage: "Unlocks Regolith, to enable use of Remote and Local filters.",
				Action: func(c *cli.Context) error {
					return regolith.Unlock(regolith.Debug)
				},
			},
		},
	}).Run(os.Args)
	if err != nil {
		regolith.Logger.Error(err)
		os.Exit(1)
	} else {
		regolith.InitLogging(false)
		regolith.Logger.Info(color.GreenString("Finished"))
	}
	result := <-status
	if result.Err != nil {
		regolith.Logger.Warn("Update check failed")
		regolith.Logger.Debug(*result.Err)
	} else if result.ShouldUpdate {
		_, _ = fmt.Fprintln(color.Output, color.GreenString("New version available!"))
		_, _ = fmt.Fprintln(color.Output, color.GreenString(*result.Url))
	}
}
