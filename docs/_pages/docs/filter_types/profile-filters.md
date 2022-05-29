---
permalink: /docs/profile-filters
layout: single
classes: wide
title: Profile Filters
sidebar:
  nav: "sidebar"
---

Profile filters are a convinient way of working with multiple profiles on one project. They save you from writing repetitive code by letting you run one filter from another. Recursive dependencies are not allowed.

## Running profile filter

Unlike other filters, the profile filter doesn't need to have a filter definition. You can always use it as long as you have multiple profiles.

The syntax for running a profile is this:

```json
{
  "profile": "my_profile"
}
```

Simply add that to the filters list of a profile.

Here is a complete `config.json` example:
```json
{
  "name": "nested_profilces_example",
  "author": "Bedrock-OSS",
  "packs": {
    "behaviorPack": "./packs/BP",
    "resourcePack": "./packs/RP"
  },
  "regolith": {
    "profiles": {
      "default": {
        "filters": [
          {
            "filter": "example_filter_1"
          }
        ],
        "export": {
          "target": "local",
          "readOnly": false
        }
      },
      "extended_default": {
        "filters": [
          {
            "profile": "default"
          },
          {
            "filter": "example_2"
          }
        ],
        "export": {
          "target": "local",
          "readOnly": false
        }
      }
    },
    "filterDefinitions": {
      "example_filter_1": {
        "runWith": "exe",
        "exe": "./example_1"
      },
      "example_filter_2": {
        "runWith": "exe",
        "exe": "./example_2"
      } 
    },
    "dataPath": "./packs/data"
  }
}
```

In this example `extended_default` profile runs the `default` profile and then it runs additional filter called `example_2`.
