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


![](/regolith/assets/images/introduction/project_folder.png)


## Compiling

In the simplest case, Regolith can be used to move your packs from the project folder, into your target location (usually the development folders in `com.mojang`). Each time you run regolith, the packs will be moved over, and updated.

However, Regoliths real value preposition is the ability to run *arbitrary code during this copy*. 

We refer to these scripts and programs as `filters`. Here is the flow:
- `RP`, `BP` and `data` folder are copied into a `temp` folder
- Every filter is executed in-order, editing the `temp` folder in-place
- The contents of `RP` and `BP` are moved into your export location
- The contents of `data` is moved back into your data location

This compilation flow allows you to make programmatic changes to your compiled addon, without effecting your source files.  

Since the data folder is saved back to our project, we can store persistent data here. 

## Filters

A filter is any program or script that takes the files inside of your RP and BP and *transforms* them in some way. Many of these filters have already been written, and are included as part of the *standard library*. 

For example, one of our standard filters is called `psd_convert`, which *filters* photoshop files in your addon, and turns them into `.png` files.

With this filter turned on, you can place `.psd` files directly into `RP/textures/*` folder! By the time your files reach `com.mojang`, the photoshop files will be replaced by a normal images -Minecraft won't know the difference!

You can write filters in Python, Javescript, Java, or any other language, using our shell integration, or select from the list of pre-written standard and community filters.

## Regolith Use Cases

### Extending the Addon Syntax

Regolith allows you to create and extend the addon-syntax arbitrarily. As long as you can write a filter to interpret the new syntax, and compile it into valid addon-syntax, then anything goes! 

For example, we allow the creation of `.molang` files -a file format that doesn't exist in Bedrock! Regolith is responsible for converting these files into a format Minecraft understands. 

With Regolith, you are empowered to write addons with an extended syntax -and Bedrock won't even know the difference!

### Non-Destructive Editing

Imagine you have a script that loops over every entity, and creates some language-code translation for it. 

Lets say your entity `regolith:big_zombie` becomes named `Big Zombie`.

If we run this script, and copy the files into our `en_US.lang`, we've saved ourself a lot of time, but we've also introduced a problem: We've *destructively edited our addon*. What this means, is we've mixed up our tool-generated content, with out hand-written content. 

Imagine we add more entities, and run our script again: Now we are in the painful position of "merging" the new, tool generated content, with our addons `en_US.lang`, which contains the old entity names alongside custom language codes. 

This is called *destructive editing*, and Regolith fixes it!

A comparable Regolith filter would only edit your `en_US.lang` during compilation, and it would do it automatically, everysingle time. 

This means as you add new entities, the names will be handled for you, without you ever seeing the names in `en_US.lang`, or needing to re-run the script. 

In other words, Regolith layers compiled content on top of your hand written content, leaving you free to create your content, without working around tool-generated content.




