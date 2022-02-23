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

Since the data folder is saved back to your project, you can store persistent data there. 

## Filters

A filter is any program or script that takes the files inside of your RP and BP and *transforms* them in some way. Many of these filters have already been written, and are included as part of the [standard library](regolith/docs/standard-filters). 

For example, one of our standard filters is called `texture_convert`, which *filters* image formats for photo editing programs, and converts them into `.png` files.

With this filter turned on, you can place Photoshop, Krita, or Gimp files directly into `RP/textures/*` folder! By the time your files reach `com.mojang`, the `.psd` files will be replaced by a normal `.png` -Minecraft won't know the difference!

### Creating your own Filters

You can write filters in Python, Javescript, Java, or any other language, using our shell integration. You can learn more about [creating custom filters here](/regolith/docs/custom-filters).

## Why Regolith?

### Extending the Addon Syntax

Regolith allows you to create and extend addon-syntax. As long as you can write a filter to interpret the new syntax, and compile it into valid addon-syntax, then anything goes! 

For example, the [subfunctions](https://github.com/Nusiq/regolith-filters/tree/master/subfunctions) community filter allows you to define functions within functions, without creating an additional file:

```s
# Some code
function <aaa>:
    # The code of the subfunction
    execute @a ~ ~ ~ function <bbb>:
        # The code of the nested subfunction
# Some other code
```

With Regolith, you are empowered to write addons with an extended syntax -and Bedrock won't even know the difference!

### Non-Destructive Editing

Imagine you have a script that loops over every entity, and creates some language-code translation for it. 

Lets say your entity `regolith:big_zombie` becomes named `Big Zombie`.

If you run this script, and copy the files into your `en_US.lang`, you've saved yourself a lot of time, but you've also introduced a problem: You've *destructively edited your addon*. What this means, is that you have mixed up your tool-generated content, with your hand-written content. 

Imagine you add more entities, and run your script again: Now you are in the painful position of "merging" the new tool generated content, with your addons `en_US.lang` file, which you may have edited in the interim.

This is called *destructive editing*, and Regolith fixes it!

A comparable Regolith filter would not suffer from this problem, because you never directly edit tool generated content. Your Regolith project folder contains only human written content, and your `com.mojang` folder contains only tool-generated content.

This means as you add new entities, the names will be handled for you, without you ever seeing the names in `en_US.lang`, or needing to re-run the script!

In other words, Regolith adds compiled content on top of your hand written content, leaving you free to create your content, without working around tool-generated content.

*If this sounds interesting to you, you might be interested in the [name ninja filter](https://github.com/Bedrock-OSS/regolith-filters/tree/master/name_ninja).*



