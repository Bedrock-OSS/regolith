---
title: User Configuration
---

# User Configuration

User configuration file is stored in the Regolith app data folder. On Windows, it's
`%localappdata%\regolith\user_config.json`**\***. The file is used to store the user
preferences for Regolith.

## Available Options

### `use_project_app_data_storage: bool`

Default: `false`

If set to `true`, the Regolith projects will store their cache (filters, their dependencies, etc.) in the app data folder, instead of the `.regolith` folder in the project folder.

### `username: string`

Default: `"Your name"`

The username of the user, which will be used in the `author` field of the `manifest.json` file when creating a new project.

### `resolvers: list[string]`

Default: `["github.com/Bedrock-OSS/regolith-filter-resolver/resolver.json"]`

A list of resolvers, which will be used to resolve filter names to URLs for downloding when using the `regolith install` command. The default URL is always added to the end of the list. Note that the "URLs" used by the resolvers are not actual URLs. They have two parts, separated by `/`. The first part is an url to a repository on GitHub, and the second part is a path to the resolver file relative to the root of the repository. For example, the default resolver is on the `github.com/Bedrock-OSS/regolith-filter-resolver` repository, in the `resolver.json` file, but `github.com/Bedrock-OSS/regolith-filter-resolver/resolver.json` is not a valid URL.

### `resolver_cache_update_cooldown: string`

Default: `"5m"`

The cooldown between cache updates for the resolvers. The cooldown is specified in the [Go duration format](https://pkg.go.dev/time#ParseDuration).

### `filter_cache_update_cooldown: string`

Default: `"5m"`

The cooldown between cache updates for the filters. The cooldown is specified in the [Go duration format](https://pkg.go.dev/time#ParseDuration).

## The `regolith config` command

The `regolith config` command is used to manage the user configuration of Regolith. It can access and modify
the user configuration file. The data is stored in the application data folder in the
"user_config.json" file.

The behavior of the command changes based on the used flags and the number of provided arguments.
The cheetsheet below shows the possible combinations of flags and arguments and what they do:

- `regolith config` - printing all properties
- `regolith config <key>` - printing specified property
- `regolith config <key> <value>` - setting property value
- `regolith config <key> --delete` - deleting a property
- `regolith config <key> <value> --append` - appending to a list proeprty
- `regolith config <key> <value> --index <index>` - replacing item in a list property
- `regolith config <key> --index <index> --delete` - deleting item in a list property

The commands that print text can take the `--full` flag to print configuration with the default values
included (if they're not defined in the config file). Without the flag, the undefined properties
will be printed as null or empty list.

## The structure of the user configuration file

The `user_config.json` file is just a regular JSON file without any nesting. You can edit it manually
if you want to but you don't have to because everything can be done with the `regolith config` command.

## Example config file
```json
{
	"use_project_app_data_storage": false,
	"username": "Bedrock-OSS",
	"resolvers": [
		"github.com/Bedrock-OSS/regolith-filter-resolver/resolver.json"
	]
}
```

----

::: info
On other platforms you can refer to Go's [os.UserCacheDir](https://pkg.go.dev/os#UserCacheDir) documentation. It's in "regolith" subdirectory of the path returned by this function.
:::
