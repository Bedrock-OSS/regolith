---
permalink: /docs/filter-types
layout: single
classes: wide
title: Filter Types
sidebar:
  nav: "sidebar"
---

Regolith contains three distinct filter types, which allow you maximum flexibility in where to store your logic. 

## Standard Library

The standard library is a special [github repository](https://github.com/Bedrock-OSS/regolith-filters) hosted in the Bedrock OSS organization. 

This repository is special, since folders here can be accessed by-name inside of Regolith, instead of by URL. This makes the standard library very easy to use for beginners!

The full list of available standard-library filters can be found [here](/regolith/docs/standard-library).


Example:

```json
{
    "filter": "json_cleaner"
}
```

This filter is equivalent to `"url": "https://github.com/Bedrock-OSS/regolith-filters/tree/master/json_cleaner"`, but since its hosted in our repository, you can just use the folder name.

## Online Filters

Online filters use the same format as standard filters, except instead of being hosted in our library repository, they can be contained in any github repository. This allows you to easily share your filters with other people. Regolith will handle the installation process.

Example:

```json
{
    "url": "github.com/user/repo/folder"
}
```

## Local Filters

Local filters are great for quickly prototyping, or for personal filters that you do not want to share with anyone. In this case, you can simply run a local file:


```json
{
    "runWith": "python",
    "script": "./filters/example.py"
}
```

The `.` path will be local to the root of the regolith project.