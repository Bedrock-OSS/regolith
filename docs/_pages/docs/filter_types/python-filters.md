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

## Running Python code as Filter

The syntax for running a python script is this:

```json
{
  "runWith": "python",
  "script": "./filters/example.py"
}
```

## Requirements and Dependencies

When installing, regolith will check for a `requirements.txt` file at the top level of the filter folder. It will attempt to install these dependencies into a venv, as described bellow.

When developing a python filter with dependencies, you must create this file. You can create a `requirements.txt` file yourself by using `pip freeze`. 

## Venv Handling

[Python Venvs](https://docs.python.org/3/library/venv.html) are flexible, lightweight "virtual environments". 

Regolith uses venvs to install dependencies, since it will prevent your global installation space from becoming polluted. When you install a python filter with dependencies, they will be installed into a venv, stored in `.regolith/cache/venvs/`.

By default, all filters will share a single venv.

In case of collision, you may use `"venvSlot": <int>` property in the filter, to claim a unique venv id. You will need to reinstall the filter.
