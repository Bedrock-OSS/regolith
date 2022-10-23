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

const regolithDesc = `
Regolith is a tool designed to make it easier to work on add-ons for Minecraft: Bedrock Edition
using scripts and executables called filters. Regolith serves as a platform for installing and
running filters.
`
const regolithRunDesc = `
This command runs Regolith using the profile specified in arguments. The profile must be defined in
the "config.json" file of the project. If the profile name is not specified, Regolith uses "default"
profile.
`
const regolithWatchDesc = `
This command starts Regolith in the watch mode. This mode will trigger the "regolith run" command
every time a change in files of the project's RP, BP, or data folders is detected. "regolith watch"
uses the same syntax as "regolith run". You can use "regolith help run" to learn more about the
command.
`
const regolithUpdateDesc = `
This command updates the Regolith filters specified in the arguments. The filters must be defined in
the "config.json" file of the project in the "filterDefinitions" section. Regolith updates the
filters to the version specified in the "config.json" file. This version does not necessarily have
to be the latest existing version of the filter.

If you want to update to the latest version of the filter, use the "regolith install" command with
the "--force" flag. You can use "regolith help install" to learn more about the "install" command.
`
const regolithUpdateAllDesc = `
Runs "regolith update" for all of the filters defined in the "filterDefinitions" list of the
"config.json" file. This command is equivalent to running the "regolith update" command with all of
the filters passed as arguments. You can learn more about the "regolith update" command by using
"regolith help update".
`
const regolithInstallDesc = `
Downloads and installs Regolith filters from the Internet, and adds them to the "filterDefinitions"
list of the project's "config.json" file. This command accepts multiple arguments, each of which
specifies what filter to install and what version to install. The syntax of the argument can have
the following forms:
- <FILTER_NAME>
- <FILTER_NAME>==<VERSION>
- <FILTER_URL>
- <FILTER_URL>==<VERSION>

Where:
- <FILTER_NAME> is the name of the filter to be resolved to URL using the Bedrock-OSS filter
  resolver repository (github.com/Bedrock-OSS/regolith-filter-resolver).
- <FILTER_URL> is the URL to the filter. Using this instead of <FILTER_NAME> lets you skip the
  resolver step and download the filters which aren't known to the resolver (for example from
  private repositories).

  The URL should follow format: <FILTER_REPOSITORY_URL>/<FILTER_NAME>.  Note that this is not a
  valid URL on GitHub because it's not how GitHub creates the URLs to subdirectories.

  To access a "name_ninja" filter from the Bedrock-OSS repository using <FILTER_URL> you would use:
  "github.com/Bedrock-OSS/regolith-filters/name_ninja" because the filter is on
  "Bedrock-OSS/regolith-filters" repository in "name_ninja" folder but this is not a valid URL on
  the GitHub website.
- <VERSION> is an optional part of the argument that you can add to specify what version of the
  filter you want to install. You can specify the version you want in multiple ways:
  - Using a semantic version of the filter (like 1.2.3)
  - Using the "latest" keyword. This option searches for the latest commit that tags the version of
    the filter.
  - Using SHA of the commit on the GitHub repository.
  - Using the "HEAD" keyword. This option looks for the latest SHA of the main branch of the
    repository.
  - Using a git tag used on the repository.

  The semantic version format is internally changed into a tag that contains two parts separated
  with a dash (-) symbol. For example argument "name_ninja==1.0.0" would be resolved by Regolith
  into: "github.com/Bedrock-OSS/regolith-filters/name_ninja==name_ninja-1.0.0".

  If no <VERSION> is specified, Regolith tries to download the filter using "latest" mode first, and
  when it fails (due to not being able to find any tags that refer to the version of the filter on
  the repository), it tries to download using "HEAD".

The "regolith install" combined with the "--force" flag can be used to change/update filters saved
in the "config.json". Unlike "regolith update" which updates to the version specified in the config
file, "regolith install --force" can modify the content of the "config.json".
`
const regolithInstallAllDesc = `
This command installs all of the filters that aren't installed already from the "filterDefinitions"
list in the "config.json"  file. It is useful when you're starting to work on a project which
already has a "config.json" file with a bunch of filters defined in it.

"regolith install-all --force" can be used to forcefully reinstall every filter on the project to
match the version from the "config.json" file. It is similar to "regolith update-all" except, the
update-all command doesn't reinstall the filters that are installed already, with the version that
matches the definition from the "config.json" file.
`
const regolithInitDesc = `
Initializes a new Regolith project in the current directory. The folder used for a new project must
be an empty directory. This command creates "config.json" and a few empty folders to be used for
RP, BP, data, and Regolith cache (.regolith folder).
`
const regolithCleanDesc = `
This command cleans the Regolith cache files for the currently opened project. With the default
Regolith configuration, the cache of Regolith is stored in the ".regolith" folder (which you can
find at the root of the project). When "config.json" sets the "useAppData" property to true, the
cache is stored in the user data folder, in a path based on a hash of the project's root folder
path. "regolith clean" always cleans both cache folders, regardless of the "useAppData" property.

Cache files include scripts/executables of Regolith filters, their virtual environments, and a list
of files recognized by Regolith as previous outputs.

After "regolith clean", usually two additional steps need to be done before running the project
again:
- the filters need to be reinstalled if there are any
- the output paths of Regolith must be deleted if the project ran before

The second step is necessary because the clean command deletes the file that stores the list of the
files created by Regolith. As a safety measure, Regolith never deletes the files that it can't
recognize so running it after "clean" would result in an error saying that Regolith stopped to
protect your files.

If you're using the "useAppData" property in your projects. It is recommended to periodically clean
the Regolith data folder to remove the cache files of the projects that you don't work on anymore.
You can clear caches of all projects stored in user data by using the "--user-cache" flag.
`

func main() {
	status := make(chan regolith.UpdateStatus)
	go regolith.CheckUpdate(version, status)
	regolith.CustomHelp()
	err := (&cli.App{
		Name:                 "regolith",
		Usage:                "Addon Compiler for the Bedrock Edition of Minecraft",
		Description:          regolithDesc,
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
				Name:        "init",
				Usage:       "Initializes a Regolith project in current directory",
				Description: regolithInitDesc,
				Action: func(c *cli.Context) error {
					return regolith.Init(regolith.Debug)
				},
			},
			{
				Name:        "install",
				Usage:       "Downloads and installs filters from the Internet and adds them to the filterDefinitions list",
				Description: regolithInstallDesc,
				Action: func(c *cli.Context) error {
					force := c.Bool("force")
					filters := c.Args().Slice()
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
				Name:        "install-all",
				Usage:       "Installs all undownloaded filters defined in filterDefintions list",
				Description: regolithInstallAllDesc,
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
				Name:        "run",
				Usage:       "Runs Regolith using specified profile",
				Description: regolithRunDesc,
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
				Name:        "watch",
				Usage:       "Watches project files and automatically runs Regolith when they change",
				Description: regolithWatchDesc,
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
				Name:        "update",
				Usage:       "Updates specified filters to versions defined in filterDefinitions list",
				Description: regolithUpdateDesc,
				Action: func(c *cli.Context) error {
					return regolith.Update(c.Args().Slice(), regolith.Debug)
				},
			},
			{
				Name:        "update-all",
				Usage:       "Updates all filters to versions defined in filterDefinitions list",
				Description: regolithUpdateAllDesc,
				Action: func(c *cli.Context) error {
					return regolith.UpdateAll(regolith.Debug)
				},
			},
			{
				Name:        "clean",
				Usage:       "Cleans Regolith cache",
				Description: regolithCleanDesc,
				Action: func(c *cli.Context) error {
					userCache := c.Bool("user-cache")
					return regolith.Clean(regolith.Debug, userCache)
				},
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "user-cache",
						Aliases: []string{},
						Usage: "Clears all caches stored in user data, instead of the cache of " +
							"the current project",
					},
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
