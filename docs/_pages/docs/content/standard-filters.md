---
permalink: /docs/standard-filters
layout: single
classes: wide
title: Standard Filters
sidebar:
  nav: "sidebar"
---

The Standard Library is a special set of filters, approved or written by the Regolith maintainers. Standard Filters offers the safest, easiest, and best support. 

Please be aware that when running in safe mode, standard filters are the only filters allowed.

## Using a Standard Filter

The syntax for standard filters is like this:

```json
{
  "filter": "<filter_name>",
  "settings" { ... } // Optional
}
```


| Filter | Description |
| ------ | ----------- |
| [blockbench_convert](https://github.com/Bedrock-OSS/regolith-filters/tree/master/blockbench_convert) | Converts blockbench models into `.geometry.json` files. |
| [bump_manifest](https://github.com/Bedrock-OSS/regolith-filters/tree/master/bump_manifest) | Bumps the manifest version in your RP and BP. Good for multiplayer testing where you need to avoid pack-caching issues. |
| [fix_emissive](https://github.com/Bedrock-OSS/regolith-filters/tree/master/fix_emissive) | Fixes emissive issues in your textures, by removing the color data from fully transparent pixels. |
| [json_cleaner](https://github.com/Bedrock-OSS/regolith-filters/tree/master/json_cleaner) | Removes comments from all json files in the project. Useful, since some filters cannot understand files with comments. |
| [kra_convert](https://github.com/Bedrock-OSS/regolith-filters/tree/master/kra_convert) | Converts Krita files into `.png` files with the same name and path. |
| [psd_convert](https://github.com/Bedrock-OSS/regolith-filters/tree/master/psd_convert) | Converts Photoshop files into `.png` files with the same name and path. |
| [texture_list](https://github.com/Bedrock-OSS/regolith-filters/tree/master/texture_list) | Automatically creates the `texture_list.json` file, based on the images you've added into your resource pack. |
