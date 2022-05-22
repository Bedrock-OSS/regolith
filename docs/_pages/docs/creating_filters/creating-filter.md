---
permalink: /docs/creating-a-filter
layout: single
classes: wide
title: Your first Filter
sidebar:
  nav: "sidebar"
---

This step-by-step tutorial will guide you through the creation of your first Regolith filter, in the Python programming language. If you're new to Regolith, you may enjoy trying out some of our [standard filters](/regolith/docs/standard-filters) first.

This page is an in-depth tutorial for Regolith filter creation, including detailed information about data-flow and semantics. If you just want to get started programming, you may consider checking out the documentation on [custom filters.](/regolith/docs/custom-filters)
{: .notice--warning}

## Installing Python

Before you can begin, you need to ensure that Python is installed on your system. There are [download instructions](/regolith/docs/python-filters) on our Python Filter page.

## Getting Started

If you don't have one yet, you should create a new Regolith project by navigating to a blank folder, and typing `regolith init`.

During this tutorial, `project` refers to the project folder you just created!
{: .notice--warning}

The first step is creating a new python file in `project/filters/giant_mobs.py`. Our first step will be simply printing `hello world!`, so add `print("hello world!")` into your new filter file.

You can also create a small addon in `RP` and `BP`. For this tutorial, you need at least a working `manifest.json`, and a single working behavior entity, such as one copied from the vanilla files.

### Testing your Filter

At this point, you need to edit your `config.json` so that `giant_mobs` is registered as a local filter. Your config should look something like this:

```json
{
  "name": "Giant Mobs",
  "author": "Your Name",
  "packs": {
    "behaviorPack": "./packs/BP",
    "resourcePack": "./packs/RP"
  },
  "regolith": {
    "profiles": {
      "default": {
        "filters": [
          {
            "filter": "giant_mobs"
          }
        ],
        "export": {
          "target": "development",
          "readOnly": true
        }
      }
    },
    "filterDefinitions": {
      "giant_mobs": {
        "runWith": "python",
        "script": "./filters/big_mobs.py"
      }
    },
    "dataPath": "./packs/data"
  }
}
```

You can now run `regolith run default`. If everything went well, you should be able to navigate into Minecraft, and create a new world with your RP and BP added. Every time you want to test, re-run `regolith run default`. The "default" profile is special, so you can run it without specifying a profile name (`regolith run`).

## Understanding Regolith

Regolith projects are structured such that every filter will be run inside of a temporary folder, with all addon and configuration files available. You should *destructively edit these files in-place*! The files will be copied and moved around for you.

The general structure will look like:

```
tmp folder
 - RP
   - manifest.json
   - ... other files
 - BP
   - manifest.json
   - ... other files
 - data
   - giant_mobs
     - ... some config files
 - giant_mobs.py
```

So for our cases, that means that we can access `RP` and `BP` folder to edit the addon files within.

## Filter Goal

In this filter, we are going to "filter" every entity in the project, and double it's size! We do this by reading every entity file, searching for the "scale" component, and editing it.

To get started, you should structure your python file a bit better:

py
```
def main():
  print("put code here!")

main()
```
