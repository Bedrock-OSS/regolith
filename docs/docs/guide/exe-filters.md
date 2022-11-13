---
title: Executable Filters
---

# Executable Filters

Exe filters allow the usage of executables.

## Running executable as a Filter

If you have have achieved the feat of having one executable for all platforms (or only care about one platform), you can simply do the following:

```json
{
  "runWith": "exe",
  "exe": "path/to/your/executable"
}
```

Alternatively, if you need to specify the executable based on platform. You can do it by using the `when` property to enable/disable the filter based on the
platform:

```json
{
  "runWith": "exe",
  "exe": "coolThing.exe",
  "when": "os == \"windows\""
},
{
  "runWith": "exe",
  "exe": "coolThingLinux",
  "when": "os == \"linux\""
},
{
  "runWith": "exe",
  "exe": "coolThingMac",
  "when": "os == \"darwin\""
},
```

You can learn more about the `when` property on in the documentation of the "config.json" file
[here](/guide/configuration#regolith-configuration).
