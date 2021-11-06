---
permalink: /docs/nim-filters
layout: single
classes: wide
title: Nim Filters
sidebar:
  nav: "sidebar"
---

The syntax for running a nim filter is this:

```json
{
  "runWith": "nim",
  "script": "./filters/example.nim"
}
```

## How to install Nim
```
  1. [Download the newest version of choosenim.](https://nim-lang.org/install_windows.html)
  2. Open a terminal in the same directory that you downloaded `choosenim`.
  3. Run the choosenim executable from the terminal with the `--firstInstall` flag.
```

## Requirements and Dependencies

If your filter has dependencies, put a .nimble file in the same directory as your .nim file.
Documentation on how to make a .nimble file is located [here](https://github.com/nim-lang/nimble#creating-packages).
