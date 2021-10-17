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

After writing this in your config, you will need to run `regolith install`. This will download any new filters into your `.regolith` folder.

After installing, you may run normally.

## Filters

The full list of filters can be found on our github. We are looking into maintaining a list here, but for now please visit our github. 

You may use any folder name (such as `bump_manifest`) as the filter name.