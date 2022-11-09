---
permalink: /docs/python-filters
layout: single
classes: wide
title: Python Filters
sidebar:
  nav: "sidebar"
---

Python is an interpreted high-level general-purpose programming language.

## Installing Python

Before you can run Python filters, you will need to [install python](https://www.python.org/downloads/).

Please ensure that you add python to your path:

![](/regolith/assets/images/installing/python.png)

We recommend that you download more or less recent versions of Python.

**Warning:** It's generally not acceptable to install python via the Microsoft Store. Python installed from here is not available on the path. If you have trouble running Python filters with Regolith, please reinstall using the link above.
{: .notice--warning}

## Running Python code as Filter

The syntax for running a python script is this:

```json
{
  "runWith": "python",
  "script": "./filters/example.py",

  // Optional property that defines the folder where regolith should look for the requriements.txt file
  "requirements": "./filters
}
```

## Requirements and Dependencies

When installing, regolith will check for a `requirements.txt` file. Regolith will look for the requirements file in
the path defined by the "requirements" property or if it's not specified, in the folder with the script.

If `reqirements.txt` file exits, Regolith will attempt to install these dependencies into a venv, as described below.

When developing a python filter with dependencies, you must create this file. You can create a `requirements.txt` file yourself by using `pip freeze`. 

## Venv Handling

[Python Venvs](https://docs.python.org/3/library/venv.html) are flexible, lightweight "virtual environments". 

Regolith uses venvs to install dependencies, since it will prevent your global installation space from becoming polluted. When you install a python filter with dependencies, they will be installed into a venv, stored in `.regolith/cache/venvs/`.

By default, all filters will share a single venv.

In case of collision, you may use `"venvSlot": <int>` property in the filter, to claim a unique venv id. You will need to reinstall the filter.
