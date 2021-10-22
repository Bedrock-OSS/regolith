---
permalink: /docs/custom-filters
layout: single
classes: wide
title: Custom Filters
sidebar:
  nav: "sidebar"
---

Regolith allows you to write your own local files, and register them directly as filters. Special support is provided for filters written with Python, Java, and Node JS,

To start writing a custom filter, you first need a script to run. We will use `hello_world.py` as an example:

```py
print("Hello world!")
```

## Creating Scripts

New filters can be defined anywhere, but the suggested location is in `filters` folder, at the top level of the Regolith project. This folder isn't created for you, so you need to create it yourself. 

Once this folder is created, you can add your scripts here. You can organize with sub-folders if desired.

## Filter Data

The accepted flow for Regolith is to store configuration scripts and configs inside of the `data` folder. This folder has special support that makes it easy to access during compilation. Read more about the data folder [here](/regolith/docs/data-folder).

## Running Custom Filter

Now you can register your script by placing the following into your profile filters list, just like a standard filter.

Example:

```json
{
  "runWith": "python",
  "script": "./filters/hello_world.py"
}
```

You can use the following "runWith" types:
 - python
 - node
 - java
 - nim
 - shell

Please see the dedicated pages for these run-types for more information. TODO

## Filter Arguments

Fundamentally, a Regolith run target is a wrapper around generating a system command, and running it. For example, the following json will generate the command: `python ./filters/hello_world.py`.

```json
{
  "runWith": "python",
  "script": "./filters/hello_world.py"
}
```

If you need to pass additional command line arguments, you can do so like this:

`"arguments": "-u --no-console"` (example of additional console commands)

These console arguments will be passed into the script as args.

## Filter Settings

The settings object is a special property of run targets, that allow passing in minified/stringified json as an arg. 

Here is an example:

```json
{
  "runWith": "python",
  "script": "./filters/message.py",
  "settings": {
    "message": "Hello World!"
  }
}
```

This will generate: `python ./filters/message.py {'hello':'world'}`

This is useful for passing user-defined settings into your filter. Simply handle the first argument in the argument array, and interpret it as json!

For example:

```py
TODO
```
