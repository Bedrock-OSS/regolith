---
title: Export Targets
---

# Export Targets

Export Targets determine where your generated files will go, after Regolith is finished compiling. You can set this target at the top level of Regolith, but it can be overridden inside individual profiles, if needed.

Export is an object, and the keys inside determine how it will function. The `target` key is required, but some export targets require additional keys.

# Configuration

Some configuration properties may be used with all export targets.

## readOnly

`readOnly` changes the permissions of exported files to read-only. The default value is `false`. This property can be used to protect against accidental editing of files that should only be edited by Regolith!

# Export Targets

These are the export targets that Regolith offers.

## Development

The development export target will place the compiled packs into your `com.mojang` `development_*_packs` folders.

```json
"export": {
    "target": "development"
}
```

Optionally, you can use `rpName` and `bpName` to specify the names of the folders that will be created in the `development_*_packs` folders. You can read more about these options at the end of this page of the documentation.

```json
"export": {
    "target": "development",
    "rpName": "'my_rp'",
    "bpName": "'my_bp'"
}
```

## Local

This export target will place the compiled packs into a folder called `build`, created in your regolith project. This export target is mostly useful for quick testing.

```json
"export": {
    "target": "local"
}
```

Local export optionally accepts `rpName` and `bpName` to specify the names of the folders that will be created in the `build` folders. You can read more about these options at the end of this page of the documentation.

```json
"export": {
    "target": "local",
    "rpName": "'my_rp'",
    "bpName": "'my_bp'"
}
```



## Exact

The Exact export target will place the files to specific, user specified locations. This is useful when you need absolute control over Regoliths export functionality.

`rpPath` and `bpPath` are required options. Both paths support environment variables by using the `%VARIABLE_NAME%` syntax.

Example:

```json
"export": {
    "target": "exact",
    "rpPath": "...",
    "bpPath": "...
}
```

The exact export target doesn't support using `rpName` and `bpName`. The `rpPath` and `bpPath` should provide full paths to the desired locations.

## World

The World export target will place the compiled files into a specific world. This is useful for teams that prefer working in-world, as opposed to in the development pack folders.

You need to use *either* `worldName` or `worldPath` to select the world. `worldPath` supports environment variables by using the `%VARIABLE_NAME%` syntax.

Example:

```json
"export": {
    "target": "world",
    "worldName": "..."  // This
    // "worldPath": "..."   // OR this
}
```

Optionally, you can use `rpName` and `bpName` to specify the names of the folders that will be created in the world. You can read more about these options at the end of this page of the documentation.

```json
"export": {
    "target": "world",
    "worldPath": "...",
    "rpName": "'my_rp'",
    "bpName": "'my_bp'"
}
```


## Preview

The development export target will place the compiled packs into your **(minecraft preview)** `com.mojang` `development_*_packs` folder.

```json
"export": {
    "target": "preview"
}
```

Optionally, you can use `rpName` and `bpName` to specify the names of the folders that will be created in the `development_*_packs` folders. You can read more about these options at the end of this page of the documentation.
```json
"export": {
    "target": "preview",
    "rpName": "'my_rp'",
    "bpName": "'my_bp'"
}
```

# The `rpName` and `bpName` expressions

The `rpName` and `bpName` are expressions evaulated using the [go-simple-eval](https://github.com/stirante/go-simple-eval/) library. They let you specify the names of the folders of the exported packs in some of the export targets.

The go-simple-eval library allows you to use simple expressions to generate the names of the folders. The expressions can use the following variables:

- `project.name` - The name of the project.
- `project.author` - The author of the project.
- `os` - The host operating system.
- `arch` - The host architecture.
- `debug` - whether regolith is running in debug mode or not.
- `version` - The version of regolith.
- `profile` - The name of the profile being run.

Go-simple-eval can concatenate strings using the `+` operator. The strings must be enclosed in single quotes.
