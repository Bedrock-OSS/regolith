---
permalink: /docs/getting-started
layout: single
classes: wide
title: Getting Started
sidebar:
  nav: "sidebar"
---

To get started with Regolith, you should first read our [introduction](/regolith/docs/introduction) page, and the [installation](/regolith/docs/installing) page.

You can test for installation by running `regolith` inside of a terminal. This guide will assume you have installed regolith directly, but you can follow along with a stand-alone build. Just ensure that the executable is placed inside of your project folder.

## Creating a new Project

To create a new project, navigate to a blank folder, and run `regolith init`. This will create a few files:


![](/assets/images/docs/introduction/project-folder.png) ![](\docs\assets\images\docs\introduction\project-folder.png)

You may now create your addon inside of the RP and BP folders, or better yet, import them from an addon you are currently working on.

## config.json

Next, open up `config.json`. We will be configuring a few fields here, for your addon.

```json
{
  "name": "Project Name", // Put your project name here
  "author": "Your name", // Put your name here
  "packs": {
    "behaviorPack": "./packs/BP",
    "resourcePack": "./packs/RP"
  },
  "regolith": {
    "profiles": {
      "dev": {
        "unsafe": false,
        "filters": [],
        "export": {
          "clean": false,
          "target": "development"
        }
      }
    }
  }
}
```

Later on you can play with the additional configuration options, but for now, just set a project name, and author name.

## Running Regolith

To run regolith, open up a terminal and type `regolith run dev`. This will run the `dev` profile, which is the default in `config.json`. When you run this command, Regolith will copy/paste your addon into the `development` folders inside of `com.mojang`. If you navigate there, you should be able to see your pack folders, with a name like `project_name_bp`. 

Every time you want to update your addon, re-run this command. 

Later on, you can experiment with creating multiple profiles -for example, one for `dev` and one for `packaging`.

## Adding your first Filter

Regolith contains a very powerful filter system, that allows you write filters in many languages, as well as from the internet. For now, we will simply use the `standard library`, which is a set of approved filters that we maintain. 

To add a new filter to your profile, you must add items into your `"filters": [],`.

We will use the `texture_list` filter, which automatically creates the `texture_list.json` file for you. To learn more about this file, and why automating it is helpful, [read here](https://wiki.bedrock.dev/visuals/textures-list.html).

```json
"filters": [
  {
    "filter": "texture_list"
  }
]
```

`Warning:` If your resource pack already contains `texture_list.json`, you should delete it. You don't need to manually worry about it anymore -Regolith will handle it!

:`Warning:` If your project doesn't have any textures, than `texture_list.json` will simply create a blank file `[]`. Consider adding some textures to see the filter at work!

Run `regolith run` (since `dev` is the default profile, you can omit the argument). You will most likely crash, since the dependencies for `texture_list` were not installed yet. This is expected.

Run `regolith install`. This may take some time, but you only need to run it when you add new filters.

After installation is finished, you can run `regolith run` again.

Check `com.mojang`, and open the new `texture_list.json` file in `RP/textures/texture_list.json`. Every time you run regolith, this file will be re-created, based on your current textures. No need to manually edit it ever again!

## Whats Next

Now that you've created your first Regolith project, and installed your first filters, you are well on your way to being a Regolith expert! You should check out the standard library, to see if additional filters might be useful for you.

Otherwise, you can learn about writing custom filters here.