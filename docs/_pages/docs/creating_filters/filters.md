---
permalink: /docs/filter-types
layout: single
classes: wide
title: Filters
sidebar:
  nav: "sidebar"
---

A filter is any program or script that takes the files inside of your RP and BP and *transforms* them in some way. Many of these filters have already been written, and are included as part of the [standard library](/regolith/docs/standard-library). You may also be interested in [community filters](/regolith/docs/community-filters)

At it's core, you can think of a filter as the ability to run arbitrary code during the compilation process. This allows you to accomplish a number of tasks:

 - Linting and error checking
 - Code generation/automation
 - Interpreting custom syntax


## Filter Versions



## Local Filters

Local filters are great for quickly prototyping, or for personal filters that you do not want to share with anyone. In this case, you can simply run a local file:


```json
{
    "runWith": "python",
    "script": "./filters/example.py"
}
```

The `.` path will be local to the root of the regolith project.