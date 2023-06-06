package main

import (
	"fmt"
	"os"

	"github.com/Bedrock-OSS/go-burrito/burrito"
	"github.com/stirante/go-simple-eval/eval"

	"github.com/fatih/color"

	"github.com/Bedrock-OSS/regolith/regolith"
	"github.com/spf13/cobra"
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
const regolithApplyFilter = `
This command runs single selected filter and applies its changes to the project source files. Running
this is a destructive operation that modifies RP, BP and data folders, so it is recommended to be
cautious when using this command and to have a way to revert the changes (e.g. using Git).

Every filter can be used this way as long as it's defined in the "config.json" file in the
"filterDefinitions" section.

The "regolith apply-filter" command runs on a copy of the project's files and copies them back to the
project only if the filter is successful. This means that if the filter fails, the project's files
aren't modified.
`
const regolithInstallDesc = `
Downloads and installs Regolith filters from the internet, and adds them to the "filterDefinitions"
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
in the "config.json".
`
const regolithInstallAllDesc = `
This commands installs or updates all of the filters specified in the "filterDefinitions" list of
the "config.json" file. It makes sure that the versions of the filters defined in the
"filterDefinitions" list in sync with the actual versions of the installed filters.

It is useful when you're starting to work on a project which already has a "config.json" file with
a bunch of filters defined in it or when the config file was modified by someone else and you want
to make sure that your local copies of the filters is up to date.

By default, the filters that are already installed with a correct version are ignored. You can
change that by using the "--force" flag. "regolith install-all --force" forcefully reinstalls every
filter on the project.
`
const regolithInitDesc = `
Initializes a new Regolith project in the current directory. The folder used for a new project must
be an empty directory. This command creates "config.json" and a few empty folders to be used for
RP, BP, data, and Regolith cache (.regolith folder).
`
const regolithCleanDesc = `
This command clears the Regolith cache files for the currently open project. With the default
Regolith configuration, the Regolith cache is stored in the ".regolith" folder (which you can
find in the root of the project). If your user configuration has the "use_project_app_data_storage"
setting set to "true", the cache will be stored in the user data folder, in a path based on a hash
of the project root path. "regolith clean" will always clean both cache folders, regardless of the
"use_project_app_data_storage" setting.

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

If you're using the "use_project_app_data_storage" setting in your user configuration. It is
recommended to periodically clean the Regolith data folder to remove the cache files of the
projects that you don't work on anymore. You can clear caches of all projects stored in user data
by using the "--user-cache" flag.
`

const regolithConfigDesc = `
The config command is used to manage the user configuration of Regolith. It can access and modify
the user configuration file. The data is stored in the application data folder in the
"user_config.json" file.

The behavior of the command changes based on the used flags and the number of provided arguments.
The cheatsheet below shows the possible combinations of flags and arguments and what they do:

Printing all properties:                        regolith config
Printing specified property:                    regolith config <key>
Setting property value:                         regolith config <key> <value>
Deleting a property:                            regolith config <key> --delete
Appending to a list property:                   regolith config <key> <value> --append
Replacing item in a list property:              regolith config <key> <value> --index <index>
Deleting item in a list property:               regolith config <key> --index <index> --delete

The printing commands can take the --full flag to print configuration with the default values
included (if they're not defined in the config file). Without the flag, the undefined properties
will be printed as null or empty list.
`
const regolithUpdateResolversDesc = `
Updates every resolver repository in the "resolvers" list in the user configuration. This command 
is particularly useful if you are adding a new filter to the resolver file and want to ensure that 
the new filter is available in the Regolith.
`

func main() {
	// Schedule error handling
	var err error
	defer func() {
		if regolith.Logger == nil { // Logger is nil when the command is 'help' or 'completion'
			return
		}
		if err != nil {
			regolith.Logger.Error(err)
			os.Exit(1)
		} else {
			regolith.Logger.Info(color.GreenString("Finished"))
		}
	}()
	// Schedule update status check
	status := make(chan regolith.UpdateStatus)
	go regolith.CheckUpdate(version, status)
	defer func() {
		if regolith.Logger == nil { // Logger is nil when the command is 'help' or 'completion'
			return
		}
		updateStatus := <-status
		if updateStatus.Err != nil {
			regolith.Logger.Warn("Update check failed")
			regolith.Logger.Debug(*updateStatus.Err)
		} else if updateStatus.ShouldUpdate {
			_, _ = fmt.Fprintln(color.Output, color.GreenString("New version available!"))
			_, _ = fmt.Fprintln(color.Output, color.GreenString(*updateStatus.Url))
		}
	}()
	// Initialize simple eval
	eval.Init()

	// Root command
	var rootCmd = &cobra.Command{
		Use:     "regolith",
		Short:   "Addon Compiler for the Bedrock Edition of Minecraft",
		Long:    regolithDesc,
		Version: version,
	}
	subcommands := make([]*cobra.Command, 0)

	var force bool
	// regolith init
	cmdInit := &cobra.Command{
		Use:   "init",
		Short: "Initializes a Regolith project in current directory",
		Long:  regolithInitDesc,
		Run: func(cmd *cobra.Command, _ []string) {
			err = regolith.Init(burrito.PrintStackTrace, force)
		},
	}
	cmdInit.Flags().BoolVarP(
		&force, "force", "f", false, "Force the operation, overriding potential safeguards.")
	subcommands = append(subcommands, cmdInit)

	profiles := []string{"default"}
	// regolith install
	var update, resolverRefresh bool
	cmdInstall := &cobra.Command{
		Use:   "install [filters...]",
		Short: "Downloads and installs filters from the internet and adds them to the filterDefinitions list",
		Long:  regolithInstallDesc,
		Run: func(cmd *cobra.Command, filters []string) {
			if len(filters) == 0 {
				cmd.Help()
				return
			}
			err = regolith.Install(filters, force || update, resolverRefresh, cmd.Flags().Lookup("profile").Changed, profiles, burrito.PrintStackTrace)
		},
	}
	cmdInstall.Flags().BoolVarP(
		&force, "force", "f", false, "Force the operation, overriding potential safeguards.")
	cmdInstall.Flags().BoolVar(
		&resolverRefresh, "force-resolver-refresh", false, "Force resolvers refresh.")
	cmdInstall.Flags().BoolVarP(
		&force, "update", "u", false, "An alias for --force flag. Use this flag to update filters.")
	cmdInstall.Flags().StringSliceVarP(&profiles, "profile", "p", profiles, "Adds installed filters to the specified profiles. If no profile is provided, the filter will be added to the default profile.")
	cmdInstall.Flags().Lookup("profile").NoOptDefVal = "default"
	subcommands = append(subcommands, cmdInstall)

	// regolith install-all
	cmdInstallAll := &cobra.Command{
		Use:   "install-all",
		Short: "Installs all nonexistent or outdated filters defined in filterDefinitions list",
		Long:  regolithInstallAllDesc,
		Run: func(cmd *cobra.Command, _ []string) {
			err = regolith.InstallAll(force, burrito.PrintStackTrace)
		},
	}
	cmdInstallAll.Flags().BoolVarP(
		&force, "force", "f", false, "Force the operation, overriding potential safeguards.")
	subcommands = append(subcommands, cmdInstallAll)

	// regolith run
	cmdRun := &cobra.Command{
		Use:   "run [profile_name]",
		Short: "Runs Regolith using specified profile",
		Long:  regolithRunDesc,
		Run: func(cmd *cobra.Command, args []string) {
			var profile string
			if len(args) != 0 {
				profile = args[0]
			}
			err = regolith.Run(profile, burrito.PrintStackTrace)
		},
	}
	subcommands = append(subcommands, cmdRun)

	// regolith watch
	cmdWatch := &cobra.Command{
		Use:   "watch [profile_name]",
		Short: "Watches project files and automatically runs Regolith when they change",
		Long:  regolithWatchDesc,
		Run: func(cmd *cobra.Command, args []string) {
			var profile string
			if len(args) != 0 {
				profile = args[0]
			}
			err = regolith.Watch(profile, burrito.PrintStackTrace)
		},
	}
	subcommands = append(subcommands, cmdWatch)

	// regolith apply-filter
	cmdApplyFilter := &cobra.Command{
		Use:   "apply-filter <filter_name> [filter_args...]",
		Short: "Runs selected filter to destructively modify the project files",
		Long:  regolithApplyFilter,
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				cmd.Help()
				return
			}
			filter := args[0]
			filterArgs := args[1:] // First arg is the filter name
			err = regolith.ApplyFilter(filter, filterArgs, burrito.PrintStackTrace)
		},
	}
	subcommands = append(subcommands, cmdApplyFilter)

	// regolith config
	cmdConfig := &cobra.Command{
		Use:   "config [key] [value]",
		Short: " Print or modify the user configuration.",
		Long:  regolithConfigDesc,
		Run: func(cmd *cobra.Command, args []string) {
			regolith.InitLogging(burrito.PrintStackTrace)
			full, _ := cmd.Flags().GetBool("full")
			delete, _ := cmd.Flags().GetBool("delete")
			append, _ := cmd.Flags().GetBool("append")
			index, _ := cmd.Flags().GetInt("index")
			err = regolith.ManageConfig(burrito.PrintStackTrace, full, delete, append, index, args)
		},
	}
	cmdConfig.Flags().BoolP("full", "f", false, "When printing, prints the full configuration including default values.")
	cmdConfig.Flags().BoolP("delete", "d", false, "Delete property")
	cmdConfig.Flags().BoolP("append", "a", false, "Append value to array property")
	cmdConfig.Flags().IntP("index", "i", -1, "The index of the array property on which to act")
	subcommands = append(subcommands, cmdConfig)

	// regolith clean
	var userCache bool
	cmdClean := &cobra.Command{
		Use:   "clean",
		Short: "Cleans Regolith cache",
		Long:  regolithCleanDesc,
		Run: func(cmd *cobra.Command, _ []string) {
			err = regolith.Clean(burrito.PrintStackTrace, userCache)
		},
	}
	cmdClean.Flags().BoolVarP(
		&userCache, "user-cache", "u", false, "Clears all caches stored in user data, instead of the cache of "+
			"the current project")
	subcommands = append(subcommands, cmdClean)

	// regolith update-resolvers
	cmdUpdateResolvers := &cobra.Command{
		Use:   "update-resolvers",
		Short: "Updates cached resolver repositories",
		Long:  regolithUpdateResolversDesc,
		Run: func(cmd *cobra.Command, _ []string) {
			err = regolith.UpdateResolvers(burrito.PrintStackTrace)
		},
	}
	subcommands = append(subcommands, cmdUpdateResolvers)

	// add --debug and --timings flag to every command
	for _, cmd := range subcommands {
		cmd.Flags().BoolVarP(&burrito.PrintStackTrace, "debug", "", false, "Enables debugging")
		cmd.Flags().BoolVarP(&regolith.EnableTimings, "timings", "", false, "Enables timing information")
	}
	// Build and run CLI
	rootCmd.AddCommand(subcommands...)
	rootCmd.Execute()
}
