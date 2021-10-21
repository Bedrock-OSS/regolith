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

## Requirements and Dependencies

If your filter has dependencies, put a .nimble file in the same directory as your .nim file.
Documentation on how to make a .nimble file is located [here](https://github.com/nim-lang/nimble#creating-packages).
