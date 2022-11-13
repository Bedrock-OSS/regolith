---
permalink: /docs/online-filters
layout: single
classes: wide
title: Hosting your Filter
sidebar:
  nav: "sidebar"
---

Regolith allows custom filters to be placed on GitHub. This is perfect for a filter that you want to make public, or potentially share internally in a team.

The standard [filters library](/regolith/docs/standard-library) is a good reference for how to structure an online filter, but we will also explain here.

## Creating Online Filter

To create an online filter, your github project needs to be structured in a certain way. For starters, every filter needs its own folder, at the top of the github project. This folder name is very important, as it will be the name of the filter.

You should move your programs and scripts into this folder. When your filter is installed, everything in this folder will be downloaded.

### filter.json

`filter.json` is a special file, which you should place at the top level of your filters folder. Once again, check out the standard-library for examples of a property structured regolith filter.

```json
{
  // Description is a property which currently is not used by Regolith itself, but the web scraper uses
  // it to add descriptions to the filters page of the documentation website.
  "description": "A Hello World Filter - this will be displayed as a description on website pages.",

  // exportData is a property which tells Regolith whether or not to export the data folder back to the
  // source files after running the profile that uses this filter.
  "exportData": true,

  "filters": [
    {
      "runWith": "python",
      "script": "./hello_world.py"
    }
  ]
}
```

#### The `exportData` property

This option lets you create remote filters that can modify their data folder. The older versions of
Regolith like 0.0.18 used to always export the entire filter data folder back to the source files
making the `regolith run` command destructive. This feature was added to let the filters remember
their state between runs. Most of the filters don't need this feature, so moving the files back and
forth was a waste of processing time. The `exportData` property is now disabled by default, but it
can be enabled for filters that need it. The `exportData` property causes Regolith to export the
folder named the same as the filter back to the source files. This way you can have both filters
that can modify their data folder and filters that can't.

## Data Folder

If you need some default configuration files for your remote filter, you can create a folder called `data` in your filter folder. Here, you can store your default configuration files. When a user runs `regolith install`, this data folder will be moved into their data folder, namespaced under the name of the filter. 

You can learn more about this flow [here](/regolith/docs/data-folder).

## Test Folder

It may be useful to you to include a test project, or test files, which are useful for development, but don't need to be downloaded by the end user. Anything placed in the `test` folder will not be installed by Regolith, and you can use this space for your own development.