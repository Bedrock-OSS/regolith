---
title: Project Configuration File
---

# Configuration File

The configuration of Regolith project is stored inside of `config.json`, at the top level of your Regolith project. This file will be created when you run `regolith init`.

## Project Config Standard

Regolith follows the [Project Config Standard](https://github.com/Bedrock-OSS/project-config-standard). This config is a shared format, used by programs that interact with Minecraft projects, such as [bridge](https://editor.bridge-core.app/).

## Regolith Configuration

Regolith builds on this standard with the addition of the `regolith` namespace, which is where all regolith-specific information is stored.

::: warning
This page only shows an example configuration. There are other documentation pages to fully explain concepts such as `filters` and `profiles`.
:::

Example config, with many options explained:

```json
{
  // These fields come from project standard
  "name": "Project Name",
  "author": "Author Name",
  "packs": {
    // You should create your packs directly within these folders.
    // Example: 'regolith_project/packs/BP/manifest.json'
    "behaviorPack": "./packs/BP",
    "resourcePack": "./packs/RP"
  },

  // These fields are for Regolith specifically
  "regolith": {
    // Profiles are a list of filters and export information, which can be run with 'regolith run <profile>'
    "profiles": {
      // 'default' is the default profile. You can add more.
      "default": {

        // Every profile contains a list of filters to run, in order.
        "filters": [
          {
            // Filter name, as defined in filter_definitions
            "filter": "name_ninja",

            // Settings object, which configure how name_ninja will run (optional)
            "settings": {
              "language": "en_GB.lang"
            }
          },
          {
            // A second filter, which will run after 'name_ninja'
            "filter": "bump_manifest",

            // Arguments list is a list of arguments to pass to the command that runs the filter (optional).
            // If filter uses both settings and arguments, the settings json is passed as the first argument.
            "arguments": ["-regolith"],
            
            // "disabled" is a bolean that determines whether or not to run this filter (optional).
            "disabled": true,

            // "when" lets you specify a condition, which determines whether or not to run
            // this filter. "when" has access to following variables:
            // - "os" - the operating system (e.g. "windows", "linux", "darwin")
            // - "arch" - the architecture (e.g. "amd64", "arm64")
            // - "version" - the version of regolith or 0.0.0 when running from source
            // - "debug" - whether the debug flag is passed to regolith or not
            // - "profile" - current profile being run
            // - "filterLocation" - absolute location of the filter folder
            "when": "os == 'windows' && arch == 'amd64'"
          }
        ],

        // Export target defines where your files will be exported
        "export": {
          "target": "development",
          "readOnly": false
        }
      }
    },

    // Filter definitions contains a full list of installed filters, known to Regolith.
    // You may install more filters using 'regolith install <identifier>'
    "filterDefinitions": {
      "name_ninja": {
        "version": "1.0"
      },
      "bump_manifest": {
        "version": "1.0"
      }
    },

    // The path to your regolith data folder, which contains configuration files for your filter.
    "dataPath": "./packs/data"
  }
}
```
