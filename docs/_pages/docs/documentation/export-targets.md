---
permalink: /docs/export-targets
layout: single
classes: wide
title: Export Targets
sidebar:
  nav: "sidebar"
---

Export Targets determine where your generated files will go, after Regolith is finished compiling. You can set this target at the top level of Regolith, but it can be overridden inside individual profiles, if needed.

Export is an object, and the keys inside determine how it will function. The `target` key is required, but some export targets require additional keys.

# Configuration

Some configuration properties may be used with all export targets.

## readOnly

`readOnly` changes the permissions of exported files to read-only. The default value is `false`. This property can be used to protect against accidental editing of files that should only be edited by Regolith!

# Export Targets

These are the export targets that Regolith offers.

## Development

The development export target will place the compiled packs into your `com.mojang` `development_*_packs` folder, in a new folder called `<name>_BP` or `<name>_RP`.

```json
"export": {
    "target": "development"
}
```

## Local

This export target will place the compiled packs into a folder called `build`, created in your regolith project. This export target is mostly useful for quick testing.

```json
"export": {
    "target": "local"
}
```

## Exact

The Exact export target will place the files to specific, user specified locations. This is useful when you need absolute control over Regoliths export functionality.

`rpPath` and `bpPath` are required options.

Example:

```json
"export": {
    "target": "exact",
    "rpPath": "...",
    "bpPath": "...
}
```

## World

The World export target will place the compiled files into a specific world. This is useful for teams that prefer working in-world, as opposed to in the development pack folders.

You need to use *either* `worldName` or `worldPath` to select the world.

Example:

```json
"export": {
    "target": "world",
    "worldName": "...",  // This
    "worldPath": "..."   // OR this
}
```

## Preview

The development export target will place the compiled packs into your (minecraft preview) `com.mojang` `development_*_packs` folder, in a new folder called `<name>_bp` or `<name>_rp`.

```json
"export": {
    "target": "preview"
}
```
