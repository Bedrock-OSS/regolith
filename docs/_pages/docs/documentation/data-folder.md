---
permalink: /docs/data-folder
layout: single
classes: wide
title: Data Folder
sidebar:
  nav: "sidebar"
---

The Regolith `data folder` is a special folder where configs can be stored. The data folder is used to store additional configuration files and assets for filters.

## Location

By default, the data folder is stored in `./packs/data`. This folder will be created for you when you run `regolith init`.

If you would like to change the data folder location, you may do so by editing `"dataPath": "./packs/data"`. Please be aware you will need to create the folder yourself.

## Remote Filter Installation

When a remote filter is installed, it has the opportunity to place some files into your data folder. If the remote filter repository contains a its own `data` folder, the contents will be moved into `data/filter_name/*`. This is a little bit like an installation script, and allows the filter to handle its own first-time setup.

## Accessing the Data Folder

When Regolith runs, it will move the `data` folder into the `tmp` directory, along with the `RP` and `BP` folders. 

For example: `(python)`

```py
with open('./data/bump_manifest/version.json', 'r') as f: 
  print(json.load(f))
```

## Saving Data

When regolith is finished running, the data folder will be moved from the temporary location, back into the normal location. This flow allows you to store persistent data, by editing or creating new files. 

This stands in contrast to the `RP` and `BP` folders, which will not be saved back into the project.

For example: `(python)`

```py
with open('./data/bump_manifest/version.json', 'w') as f: 
  json.dump({'version': '1.0'}, f)
```