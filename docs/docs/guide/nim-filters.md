---
title: Nim Filters
---

# Nim Filters

Nim is a statically typed compiled systems programming language. It combines successful concepts from mature languages like Python, Ada and Modula. 

## Installing Nim

  1. Download the newest version of [choosenim.](https://nim-lang.org/install_windows.html)
  2. Open a terminal in the same directory that you downloaded `choosenim`.
  3. Run `choosenim --firstInstall`

## Running Nim code as Filter

The syntax for running a nim filter is this:

```json
{
  "runWith": "nim",
  "script": "./filters/example.nim",

  // Optional property that defines the path to the folder with the *.nimble file
  "requirements": "./filters"
}
```

## Requirements and Dependencies

If your filter has dependencies, put a `.nimble` file in the same directory as your `.nim` file, alternatively
you can specify different path using the "requirements" property. Regolith will look for the nimble file in
the folder specified by "requirements" or if it's not defined in the same folder as the script.

Documentation on how to make a `.nimble` file is located [here](https://github.com/nim-lang/nimble#creating-packages).
