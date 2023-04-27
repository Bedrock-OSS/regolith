---
title: Installing and Updating Filters
---

# Installing and Updating Filters

To start using a filter, you need to do four things:

 1. Ensure you can run the filter
 2. Install the filter
 3. Add the filter to the profile which you would like to use it.
 4. Run your profile, to test it out!

## Filter Dependencies

Filters are written in [programming languages](https://www.wikiwand.com/en/Programming_language). These languages may not be installed on your computer by default. Before installing a filter, you should ensure you have the proper programming language installed. The "Filter Types" documentation has detailed installation instructions for every regolith-supported language!

For example, if the filter relies on Python, you can find installation instructions [here](/guide/python-filters).

## Installing a Filter

Regolith contains a powerful installation command, which will download a filter from GitHub, and install any required libraries for you. In general, the format is like this: `regolith install <filter_identifier>`

The value of `filter_identifier` will depend on where the filter is hosted. Filters listed on the [Bedrock-OSS/regolith-filter-resolver](https://github.com/Bedrock-OSS/regolith-filter-resolver/blob/main/resolver.json) repository can be installed by their name. For example, to install the `name_ninja` filter, you would run the:

```
regolith install name_ninja
```
If the filter is not listed on the resolver repository, you will need to use the following format:
`github.com/<user>/<repository>/<folder>`.

For example, to install `name_ninja` using the full format, you would run:

```
regolith install github.com/Bedrock-OSS/regolith-filters/name_ninja
```
The longer form can be used to install filters from private repositories.


::: warning
The `install` command relies on `git`. You may download git [here](https://git-scm.com/download/win).
:::

## Adding Filter to Profile

After installing, the filter will appear inside of `filter_definitions` of `config.json`. You can now add this filter to a profile like this:

```json
"default": {
  "export": {
    "readOnly": false,
    "target": "development"
  },
  "filters": [
    {
      "filter": "FILTER_NAME",
    }
  ]
}
```

## Install All

Regolith is intended to be used with git version control, and by default the `.regolith` folder is ignored. That means that when you collaborate on a project, or simply re-clone your existing projects, you will need an easy way to download all the filters again!

You may use the command `regolith install-all`, which will check `config.json`, and install every filter in the `filterDefinitions`.

{: .notice--warning}
This is only intended to be used with existing projects. To install new filters, use `regolith install`.

## Filter Versioning

Filters in Regolith are optionally versioned with a [semantic version](https://semver.org/). As filters get updated, new versions will be released, and you can optionally update.

{: .notice--warning}
If you don't specify a version, the `install` command will pick a sensible default. First, it will search for the latest release. If that doesn't exist (such as a filter that has no versions), it will select the latest commit in the repository. In both cases, the installed version will be `pinned`.

### Installing a Specific Version

When installing, you can optionally include a version key after two  equals signs (`==`):

 - ⭐ Version: `regolith install name_ninja==1.2.8`
 - Unpinned Head: `regolith install name_ninja==HEAD`
 - Unpinned Latest: `regolith install name_ninja==latest`
 - SHA: `regolith install name_ninja==adf506df267d10189b6edcdfeec6c560247b823f`

### Pinned Versions

In your `config.json`, every filter will include a `version` field, which specifies which version of the filter to use. By default, this version will be `pinned`, meaning that it won't be updated, even if new versions release. This provides you safety, and ensures that your projects will continue to operate without interruption even if filters release breaking changes.

Optionally, you may mark filters as `unpinned`, which signifies that your project wants the latest version of the filter, no questions asked. There are two available `unpinned` versions:
 - `latest` points to the latest released version tag.
 - `HEAD` points to the latest commit of the repository, regardless of release tags.

### Updating your Filters

If you want to update the version of the filter used in your project, you can use the `regolith install` command again. By default, the `install` command is not allowed to update existing filters, but you can use the `--update` or `--force` flag to change this behavior. The flag must be used after the `install` arguments.

```
regolith install name_ninja --update
```

Alternatively, you can modify the `version` field in `config.json` and run `regolith install-all`. Regolith install-all is useful for working in a team, when other team members may have to update or add filters to the project.

### Updating resolvers

When using short names for filters, Regolith uses a resolver file from a remote repository to determine the URL of the filter.
By default, this remote repository is cached and only updated after 5 minutes since last update. If you want to update the resolver file immediately, you can use the `regolith update-resolver` command.

Alternatively, you can use the `--force-resolver-update` flag to force the resolvers to update when installing a filter.

```
regolith install name_ninja --force-resolver-update
```

### Updating filter cache

Regolith caches the filter repository when you install online filters. By default, the repository cache is updated every 5 minutes. 
However, if you need to update the cache immediately, you can use the `--force-filter-update` flag while installing a filter.

```bash
regolith install name_ninja --force-filter-update
# OR
regolith install-all --force-filter-update
```