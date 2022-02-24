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

If you run into issues installing, you can check our [troubleshooting guide](/regolith/docs/troubleshooting) for tips.

{: .notice--warning}
**Warning:** Regolith is a command line application. It assumes some familiarity with how to navigate and use the command line. You can get a [refresher](https://tutorial.djangogirls.org/en/intro_to_command_line/) here.



## Creating a new Project

To create a new project, navigate to a blank folder, and run `regolith init`. This will create a few files:


![](/regolith/assets/images/introduction/project_folder.png)

In short:
 - `.regolith` is a special folder that regolith uses to store data. You don't need to look here.
 - `packs/BP` stores your behavior pack.
 - `packs/RP` stores your resource pack.
 - `packs/data` is a special folder that filters can use to store data.
 - `config.json` is the configuration file for Regolith.

## config.json

Next, open up `config.json`. We will be configuring a few fields here, for your addon.

```json
{
  "name": "Project name", // Enter your pack name here. (Example: spooky_gravestones)
  "author": "Your name",  // Enter your author name here. (example: SirLich)
  "packs": {
    "behaviorPack": "./packs/BP",
    "resourcePack": "./packs/RP"
  },
  "regolith": {
    "profiles": {
      "dev": {
        "filters": [],
        "export": {
          "target": "development",
          "readOnly": false
        }
      }
    },
    "filterDefinitions": {},
    "dataPath": "./packs/data"
  }
}
```

Later on you can play with the additional configuration options, but for now, just set a project name, and author name.

{: .notice}
We suggest using a name like `dragons` or `cars` for the project name, as opposed to `My Dragon Adventure Map`, since the project name will be used as the folder name for the final export.

## Creating your Addon

At this point, you will want to add some files into your regolith project. If you have an existing project, you can copy/paste the files into the `RP` (resource pack) and `BP` (behavior pack) folders. 

If you don't have an addon prepared, you may also create a fresh one directly in your project folder, following the normal rules. Add a `manifest.json`, a `pack_icon.png`, and any other files you want. The files should go directly into the `RP` and `BP` folders, like this:

![](/regolith/assets/images/introduction/project_folder2.png)


## Running Regolith

To run regolith, open up a terminal and type `regolith run`. This will run the default profile (dev) from `config.json`. When you run this command, Regolith will copy/paste your addon into the `development` folders inside of `com.mojang`. If you navigate there, you should be able to see your pack folders, with a name like `project_name_bp`, and `project_name_rp`.

{: .notice--warning}
Every time you want to update your addon, re-run this command.

Later on, you can experiment with creating multiple [profiles](/regolith/docs/profiles) -for example, one for `dev` and one for `packaging`.

## Adding your first Filter

Regolith contains a very powerful filter system, that allows you write filters in many languages, as well as running existing filters from the internet. For now, we will simply use the [standard library](/regolith/docs/standard-filters), which is a set of approved filters that we maintain. 

As an example, we will use the `texture_list` filter, which automatically creates the `texture_list.json` file for you. To learn more about this file, and why automating it is helpful, read [here](https://wiki.bedrock.dev/visuals/textures-list.html).

### Installing

You can install this filter by running `regolith install texture_list`, which will make the filter available for use. 

The last step is selecting where/when the filter will run. In our case, we want to run the filter every time we export using the default `dev` profile.

You should adjust the dev profile in `config.json` to look like this:

```json
"dev": {
  "export": {
    "readOnly": false,
    "target": "development"
  },
  "filters": [
    {
      "filter": "texture_list",
    }
  ]
}
```

### Running

Now, you can re-run `regolith run`.

Check `com.mojang`, and open the new `texture_list.json` file in `RP/textures/texture_list.json`. Every time you run regolith, this file will be re-created, based on your current textures. No need to manually edit it ever again!

{: .notice--warning}
`Warning:` If your resource pack already contains `texture_list.json`, you should delete it. You don't need to manually worry about it anymore -Regolith will handle it!

{: .notice--warning}
`Warning:` If your project doesn't have any textures, than `texture_list.json` will simply create a blank file `[]`. Consider adding some textures to see the filter at work!

## Whats Next

Now that you've created your first Regolith project, and installed your first filters, you are well on your way to being a Regolith expert! You should check out the [standard library](/regolith/docs/standard-filters), to see if additional filters might be useful for you.

Otherwise, you can learn about writing [custom filters](/regolith/docs/custom-filters).