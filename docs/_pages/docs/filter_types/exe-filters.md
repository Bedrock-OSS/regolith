---
permalink: /docs/exe-filters
layout: single
classes: wide
title: Executable Filters
sidebar:
  nav: "sidebar"
---

Exe filters allow the usage of arbitrary executables. This is useful for taking care of things that might not be supported by the other filtertypess

## Running arbitrary executable as a Filter

If you have have achieved the feat of having one executable for all platforms (or only care about one platform), you can simply do the following:

```json
{
  "runWith": "exe",
  "exe": "path/to/your/executable"
}
```

Alternatively, if you need to specify the executable based on platform, platform specific executables are supported:

```json
{
  "runWith": "exe",
  "exeWindows": "coolThing.exe",
  "exeMac": "coolThingMac",
  "exeLinux": "coolThingLinux"
}
```
