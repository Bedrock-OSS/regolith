---
permalink: /docs/nim-filters
layout: single
classes: wide
title: Nim Filters
sidebar:
  nav: "sidebar"
---

## How to install Nim
```
  1. download the newest version of choosenim from https://nim-lang.org/install_windows.html
  2. open a terminal in the directory you downloaded choosenim to
  3. run the choosenim executable from the terminal with the --firstInstall flag
```

The syntax for running a nim filter is this:

```json
{
  "runWith": "nim",
  "script": "./filters/example.nim"
}
```

## Requirements and Dependencies

If your filter has dependencies, put a .nimble file in the same directory as your .nim file.
Documentation on how to make a .nimble file is located [here](https://github.com/nim-lang/nimble#creating-packages).
