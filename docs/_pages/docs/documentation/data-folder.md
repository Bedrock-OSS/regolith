---
permalink: /docs/data-folder
layout: single
classes: wide
title: Data Folder
sidebar:
  nav: "sidebar"
---

The Regolith `data` folder is a special folder where configuration files can be stored.

## Location

By default, the data folder is stored in `./packs/data`. This folder will be created for you when you run `regolith init`.

If you would like to change the data folder location, you may do so by editing `"dataPath": "./packs/data"`. Please be aware you will need to create the folder yourself!

## Remote Filter Installation

When a remote filter is installed, it has the opportunity to place some files into your data folder. 

If the remote filter repository contains a `data` folder, at the same level as `filter.json`, the contents will be moved into `data/filter_name/*`. This is our supported "first time setup" flow. If you're developing a remote filter, you are encouraged to use the data folder, and create configuration files with sensible defaults.

{: .notice--warning}
Don't worry! Your data won't be lost. `regolith install` will never overwrite your data files. If the folder is already in use during installation, a warning will be logged and the step will be skipped.

## Accessing the Data Folder

When Regolith runs, it will move the `data` folder into the `tmp` directory, along with the `RP` and `BP` folders. You can access the files here directly, just as you do the pack files.

For example: `(python)`

```py
with open('./data/bump_manifest/version.json', 'r') as f:
  print(json.load(f))
```

## Saving Data

When regolith is finished running, the data folder will be moved from the temporary location, back into the normal location. This flow allows you to store persistent data, by editing or creating new files. 

{: .notice--warning}
This stands in contrast to the `RP` and `BP` folders, which will not be saved back into the project!

For example: `(python)`

```py
with open('./data/bump_manifest/version.json', 'w') as f:
  json.dump({'version': '1.0'}, f)
```