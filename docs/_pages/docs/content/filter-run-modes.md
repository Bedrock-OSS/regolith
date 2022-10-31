---
permalink: /docs/filter-run-modes
layout: single
classes: wide
title: Working with Filters
sidebar:
  nav: "sidebar"
---

There are 3 ways of running Regolith:
- `regolith run`
- `regolith watch`
- `regolith tool`

## Run and Watch Commands

Regolith `run` and `watch` are very similar to each other. They run a profile. The difference is
that the `watch` command will watch for changes in the RP, BP and [data](/regolith/docs/data-folder) folders
and rerun the profile when they change. The `run` command will run the profile only once.

The syntax for run and watch command is:
```
regolith run [profile-name]
```
```
regolith watch [profile-name]
```
Where `[profile-name]` is the name of the profile defined your "config.json" file you want to run.
The `[profile-name]` is optional. If you don't specify it, the `"default"` profile will be run.

A single run performs the following steps:
1. Copy your source files into a temporary folder.
2. Runs all of the filters of the profile.
3. Moves the files to the target location defined in the "export" property of the profile.

Filters work on the copies of RP, BP, and data. Thanks to the use of copies, RP
and BP cannot be modified by the filters. The data folder can be modified
because after a successful run Regolith moves the files of the copy to the
the original data folder (this is useful for the filters so that they can store
some data between runs).

## Tool Command - Running Regolith Destructively

Running Regolith with `regolith run` or `regolith watch` is a safe operation because the filters can
only modify the data folder but not RP and BP. Sometimes you want to modify the RP and BP directly
in a destructive way. This is where the tool-filters come in handy. You can use any filter as a tool
by running the `regolith tool` command. Unlike the `regolith run` command, the `regolith tool`
command runs only one filter instead of running entire profile.

The command is used like this:
```
regolith tool <filter-name> [args...]
```

The `filter-name` is the name of one of the filters installed in your project. The `args` is a list of arguments passed to the filter.
