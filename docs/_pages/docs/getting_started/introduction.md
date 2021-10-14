---
permalink: /docs/introduction
layout: single
classes: wide
title: Introduction
sidebar:
  nav: "sidebar"
---

Regolith is an Addon Compiler for the Bedrock Edition of Minecraft.

Much like [bridge v2](https://editor.bridge-core.app/), Regolith introduces the concept of a "project folder", where your addons are written, including the RP, the BP, and any models, textures or configuration files. This single-folder-structure is great for version control, and allows you to keep your "source-of-truth" outside of com.mojang.

Here is what a newly initialized Regolith project looks like:


![](assets/images/docs/introduction/project-folder.png)


## Compiling

In the simplest case, Regolith can be used to move your packs from the project folder, into your target location (usually the development folders in `com.mojang`). Each time you run regolith, the packs will be moved over, and updated.

However, Regoliths real value preposition is the ability to run *arbitrary code during this copy*. 

We refer to these scripts and programs as `filters`. Here is the flow:
- RP and BP are copied into a temporary folder
- Every filter is executed in-order, editing the `temp` folder in-place
- The contents of `temp` are moved into your export location

## Filters

A filter is any program or script that takes the files inside of your RP and BP and *transforms* them in some way. Many of these filters have already been written, and are included as part of the *standard library*. 

For example, one of our standard filters is called `psd_convert`, which *filters* photoshop files in your addon, and turns them into `.png` files.

With this filter turned on, you can place `.psd` files directly into `RP/textures/*` folder! By the time your files reach `com.mojang`, the photoshop files will be replaced by a normal images -Minecraft won't know the difference!

## Extending the Addon Syntax

Regolith allows you to create and extend the addon-syntax arbitrarily. As long as you can write a filter to interpret the new syntax, and compile it into valid addon-syntax, then anything goes! 

A few fun examples, from our standard library:
- Automatically convert `.psd` into `.png`
- Automatically convert `.bbmodel` into `.geo.json`
- Automatically generate `texture_list.json`
- Automatically create many simple items at once, based on a new `simple_items.json` file.



