---
permalink: /docs/shell-filters
layout: single
classes: wide
title: Shell Filters
sidebar:
  nav: "sidebar"
---

Shell Filters allow you to run arbitrary shell commands. This is useful for running packaged scripts (`.exe` files), or for running scripts that are not natively supported in Regolith.

## Running arbitrary Shell command as Filter

The syntax for running a shell script is this:

```json
{
  "runWith": "shell",
  "command": "echo 'hello world'"
}
```

Here is another example:

```json
{
  "runWith": "shell",
  "command": "python -u ./filters/my_filter.py"
}
```