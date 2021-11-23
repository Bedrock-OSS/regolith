---
permalink: /docs/configuration
layout: single
classes: wide
title: Configuration File
sidebar:
  nav: "sidebar"
---

The configuration of regolith is stored completely in `config.json`, at the top level of your Regolith project. This file will be created when you run `regolith init`.

## Project Config Standard
Regolith follows the [Project Config Standard](https://github.com/Bedrock-OSS/project-config-standard). This config is a shared format, used by programs that interact with Minecraft Bedrock, such as [bridge](https://editor.bridge-core.app/).

Here is an example of the format:
```json
{
  "name": "example_project", // Project name
  "author": "Regolith Team", // Author
  "packs": {
    "behaviorPack": "./packs/BP",
    "resourcePack": "./packs/RP"
  }
}
```

## Regolith Configuration

Alongside the standard configuration options. Regolith introduces its own namespace, where our settings are defined.

Example config:

```json
{
  "name": "Project Name",
  "author": "Your name",
  "packs": {
    "behaviorPack": "./packs/BP",
    "resourcePack": "./packs/RP"
  },
  "regolith": {
    "profiles": {
      "dev": {
        "dataPath": "./packs/data",
        "filters": [
          {
            "filter": "hello_world"
          }
        ],
        "export": {
          "target": "development"
        }
      }
    }
  }
}
```

## Properties Explained

`profiles` is a list of filters and export options, that can be run using `regolith run`. Learn more [here](/regolith/docs/profiles).

