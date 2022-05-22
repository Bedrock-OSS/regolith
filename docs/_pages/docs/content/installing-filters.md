---
permalink: /docs/installing-filters
layout: single
classes: wide
title: Installing Filters
sidebar:
  nav: "sidebar"
---

To start using a filter, you need to do four things:

 1. Ensure you can run the filter
 2. Install the filter
 3. Add the filter to the profile which you would like to use it.
 4. Run your profile, to test it out!

### Filter Dependencies

Filters are written in [programming languages](https://www.wikiwand.com/en/Programming_language). These languages may not be installed on your computer by default. Before installing a filter, you should ensure you have the proper programming language installed. The "Filter Types" documentation has detailed installation instructions for every regolith-supported language!

For example if the filter relies on python, you can find [installation instructions here](regolith/docs/python-filters).

### Installing a Filter

Regolith contains a powerful installation command, which will download a filter from github, and install any required libraries for you. In general, the format is like this:

`regolith install <filter_identifier>`

The value of `filter_identifier` will depend on where the filter is hosted, but in general the format is: `github.com/<user>/<repository>/<folder>`

{: .notice--warning}
The `install` command relies on `git`. [You may download git here](https://git-scm.com/download/win).

### Adding Filter to Profile

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

## Filter Types

Regolith comes with a rich ecosystem of existing filters, which broadly fits into two categories:

### Standard Library

The standard library is a special [github repository](https://github.com/Bedrock-OSS/regolith-filters) hosted in the Bedrock OSS organization. This repository is special, since filters here can be accessed by name, instead of by URL. This makes the standard library very easy to use for beginners!

The full list of available standard-library filters can be found [here](/regolith/docs/standard-library).

To install, you may use the name directly: e.g., `regolith install json_cleaner`

{: .notice--warning}
This  is equivalent to `regolith install github.com/Bedrock-OSS/regolith-filters/json_cleaner`, but since it's hosted in our repository, you can just use `json_cleaner`!

### Community Filters

Community filters use the same format as standard filters, except instead of being hosted in our library repository, they can be contained in any github repository. To install a community filter, you will need to use the full identifier:

Example: `regolith install github.com/Bedrock-OSS/regolith-filters/json_cleaner`

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

When installing, you can optionally include a version key after two `==`:

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

Generally speaking, updating your filters only makes sense when you're working with unpinned versions. Pinned filters will always report themselves as up to date, unless you explicitly ask for a new version.

Commands:
 - `regolith update <filter_name>`
 - `regolith update-all`

