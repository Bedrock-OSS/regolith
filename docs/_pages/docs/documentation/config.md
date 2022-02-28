---
permalink: /docs/configuration
layout: single
classes: wide
title: Configuration File
sidebar:
  nav: "sidebar"
---

The configuration of regolith is stored inside of `config.json`, at the top level of your Regolith project. This file will be created when you run `regolith init`.

## Project Config Standard
Regolith follows the [Project Config Standard](https://github.com/Bedrock-OSS/project-config-standard). This config is a shared format, used by programs that interact with Minecraft projects, such as [bridge](https://editor.bridge-core.app/).

## Regolith Configuration

Regolith builds on this standard with the addition of the `regolith` namespace, which is where all regolith-specific information is stored:

Example config, with many options explained:

```json
{
  // These fields come from project standard
  "name": "Project name",
  "author": "Your name",
  "packs": {
    "behaviorPack": "./packs/BP",
    "resourcePack": "./packs/RP"
  },

  // These fields are for Regolith specifically
  "regolith": {

    // Profiles are a list of filters and export information, which can be run with 'regolith run <profile>'
    "profiles": {
      "dev": {
        // Every profile contains a list of filters to run
        "filters": [
          // Filter name, as defined in filter_definitions
          "filter": "name_ninja",

          // Settings object, which configure how name_ninja will run
          "settings": {
            "language": "en_GB.lang"
          }
        ],

        // Export target defines where your files will be exported
        "export": {
          "target": "development",
          "readOnly": false
        }
      }
    },

    // Filter definitions contains a full list of installed filters, known to Regolith
    "filterDefinitions": {
      "name_ninja": {
        "version": "1.0"
      }
    },
    "dataPath": "./packs/data"
  }
}
```
