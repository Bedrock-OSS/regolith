"""
Simple script to populate the standard filters page.
"""

import requests

PAGE_PATH = './_pages/docs/content/standard-filters.md'
URL = 'https://api.github.com/repos/bedrock-oss/regolith-filters/contents/'
IGNORE = ['future']
BASE_CONTENT = """
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

""".lstrip()


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
    return response.json().get('description', 'No description.')

def main():
    """
    Main function
    """
    with open(PAGE_PATH, 'w') as page:
        page.write(BASE_CONTENT)
        page.write("\n")
        page.write('| Filter | Description |\n')
        page.write('| ------ | ----------- |\n')

        for filter_name in get_filters():
            if filter_name in IGNORE:
                continue
            page.write('| [{}]({}) | {} |\n'.format(
                filter_name, get_filter_link(filter_name), get_filter_description(filter_name)))

if __name__ == '__main__':
    main()