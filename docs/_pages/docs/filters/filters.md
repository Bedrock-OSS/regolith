---
permalink: /docs/filter-types
layout: single
classes: wide
title: Filters
sidebar:
  nav: "sidebar"
---

A filter is any program or script that takes the files inside of your RP and BP and *transforms* them in some way. Many of these filters have already been written, and are included as part of the [standard library](/regolith/docs/standard-filters). You may also be interested in [community filters](/regolith/docs/community-filters)

At it's core, you can think of a filter as the ability to run arbitrary code during the compilation process. This allows you to accomplish a number of tasks:

 - Linting and error checking
 - Code generation/automation
 - Interpreting custom syntax

## Installing and Using Filters

To start using a filter, you need to do three things:

 1) Ensure you can run the filter
 2) Install the filter
 3) Add the filter to the profile which you would like to use it.

### Filter Dependencies

Filters are written in programming languages. These languages may not be installed on your computer by default. Before installing a filter, you should ensure you have the proper programming installed. The "Filter Types" documentation documentation has detailed installation instructions for every regolith-supported language. 

For example if the filter relies on python, you can find [installation instructions here](regolith/docs/python-filters).

### Installing a Filter

Regolith contains a powerful installation command, which will download a filter from github, and install any required libraries for you. In general, the format is like this:

`regolith install <location>`

The value of `location` will depend on where the filter is hosted. This is explained bellow.

{: .notice--warning}
The `install` command relies on `git`. [You may download git here](https://git-scm.com/download/win).

### Adding Filter to Profile

After installing, the filter will appear inside of `filter_definitions` of `config.json`. You can now add this filter to a profile like this:

```json
"dev": {
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

### Standard Library
The standard library is a special [github repository](https://github.com/Bedrock-OSS/regolith-filters) hosted in the Bedrock OSS organization. This repository is special, since filters here can be accessed by name, instead of by URL. This makes the standard library very easy to use for beginners!

The full list of available standard-library filters can be found [here](/regolith/docs/standard-library).

To install, you may use the name directly: e.g., `regolith install json_cleaner`

{: .notice--warning}
This  is equivalent to `regolith install https://github.com/Bedrock-OSS/regolith-filters/tree/master/json_cleaner`, but since its hosted in our repository, you can just use `json_cleaner`!

### Community Filters

Community filters use the same format as standard filters, except instead of being hosted in our library repository, they can be contained in any github repository. To install a community filter, you will need to use the full URL:

Example: `regolith install https://github.com/Bedrock-OSS/regolith-filters/tree/master/json_cleaner`

## Filter Versions



## Local Filters

Local filters are great for quickly prototyping, or for personal filters that you do not want to share with anyone. In this case, you can simply run a local file:


```json
{
    "runWith": "python",
    "script": "./filters/example.py"
}
```

The `.` path will be local to the root of the regolith project.