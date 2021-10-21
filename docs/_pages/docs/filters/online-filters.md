---
permalink: /docs/online-filters
layout: single
classes: wide
title: Online Filters
sidebar:
  nav: "sidebar"
---

Regolith allows custom filters to be placed on GitHub. This is perfect for a filter that you want to make public, or potentially share internally in a team. 

The standard [filters library](/regolith/docs/standard-filters) is a good reference for how to structure an online filter, but we will also explain here.

## Running an Online Filter

To add an online filter to your profile, you must write like this:

```json
{
  "url": "https://github.com/username/repository/folder"
}
```

You must run `regolith install` before you can compile. This will pull the files into your machine, and the next run will use these cached files. 

## Updating an Online Filter

To update your online filters, you may use `regolith install --force`, or you can manually delete the cached folder, and reinstall by using `regolith install`

## Creating Online Filter

To create an online filter, your github project needs to be structured a little special. For starters, every filter will get its own folder, at the top of the github project. This folder name is very important, as it will be the name of the filter.

Next, you need to add your scripts and programs into this folder.

Next, you need to create a `filter.json` file, which contains the following:

```json
{
  "filters": [
    {
      "runWith": "python",
      "script": "./hello_world.py"
    }
  ]
}
```

As you can see, the format is pretty much identical to a [local filter](/regolith/docs/filter-types#local-filters). 

## Data Folder

If you need some default configuration files for your remote filter, you can create a folder called `data` in your filter folder. Here, you can store your default configuration files. When a user runs `regolith install`, this data folder will be moved into their data folder, namespaced under the name of the filter. 

You can learn more about this flow [here](/regolith/docs/data-folder).