"""
Simple script to populate the standard filters page.
"""

import requests
import community
import os

STANDARD_PATH = 'docs/_pages/docs/filter_types/standard-library.md'
COMMUNITY_PATH = 'docs/_pages/docs/content/community-filters.md'
URL = 'https://api.github.com/repos/bedrock-oss/regolith-filters/contents/'
IGNORE = ['future']
BASE_CONTENT = """
---
permalink: /docs/standard-library
layout: single
classes: wide
title: Standard Library
sidebar:
  nav: "sidebar"
---

<!-- This page is auto-generated. To edit it, you'll need to change the filter_fetch.py -->

The Standard Library is a special set of filters, written by the Regolith maintainers. Standard Filters offers the safest, easiest, and best support. 

## Standard Filters

""".lstrip()

BASE_CONTENT_NEXT = """
The full, up to date list of filters can be found on our github. We are looking into maintaining a list here, but for now please visit our GitHub. 

## Using a Standard Filter

You may install standard filters by name. For example: `regolith install name_ninja`.

The syntax for standard filters usage is like this:

```json
{
  "filter": "<filter_name>",
  "settings" { ... } // Optional
}
```
"""


# Base URL, which will be used to generate the links
BASE_URL = 'https://github.com/Bedrock-OSS/regolith-filters/tree/master/'
RAW_BASE_URL = 'https://raw.githubusercontent.com/Bedrock-OSS/regolith-filters/master/'
def get_filters():
    """
    Returns a list of filters
    """
    response = requests.get(URL)
    response.raise_for_status()
    return [filter['name'] for filter in response.json() if filter['type'] == 'dir']

def get_filter_link(filter_name):
    """
    Returns the link to a filter
    """
    return BASE_URL + filter_name

def get_filter_description(filter_name):
    """
    Returns the description of a filter
    """
    
    filter_file_url = RAW_BASE_URL + filter_name + '/filter.json'
    response = requests.get(filter_file_url)
    response.raise_for_status()
    print(response.json())
    return response.json().get('description', 'No description.')

def main():
    """
    Main function
    """
    with open(STANDARD_PATH, 'w') as page:
        page.write(BASE_CONTENT)
        page.write('| Filter | Description |\n')
        page.write('| ------ | ----------- |\n')

        for filter_name in get_filters():
            if filter_name in IGNORE:
                continue
            page.write('| [{}]({}) | {} |\n'.format(
                filter_name, get_filter_link(filter_name), get_filter_description(filter_name)))
        page.write(BASE_CONTENT_NEXT)

    cFilters = community.comFilters()
    community.updateFilters(community.filterTable(cFilters), COMMUNITY_PATH)

if __name__ == '__main__':
    main()