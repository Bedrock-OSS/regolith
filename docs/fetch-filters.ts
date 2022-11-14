import { writeFile } from 'node:fs/promises'
import { join, resolve } from 'node:path'
import { $fetch } from 'ohmyfetch'

async function writeStandardLibrary(): Promise<void> {
  const headerContent = [
    '---',
    'title: Standard Library',
    '---',
    '',
    `<!-- This page is auto-generated. To edit it, you'll need to edit the "docs/fetch-filters.ts" -->`,
    '',
    '# Standard Library',
    '',
    'The Standard Library is a special set of filters, written by the Regolith maintainers. Standard Filters offers the safest, easiest, and best support.',
    '',
    '## Standard Filters'
  ]
  const footerContent = [
    'The full, up to date list of filters can be found on our github. We are looking into maintaining a list here, but for now please visit our GitHub.',
    '',
    '## Using a Standard Filter',
    '',
    'You may install standard filters by name. For example: `regolith install name_ninja`.',
    '',
    'The syntax for standard filters usage is like this:',
    '',
    '```json',
    '{',
    '  "filter": "<filter_name>",',
    '  "settings" { ... } // Optional',
    '}',
    '```'
  ]

  const baseUrl = 'https://github.com/Bedrock-OSS/regolith-filters/tree/master/'
  const rawBaseUrl =
    'https://raw.githubusercontent.com/Bedrock-OSS/regolith-filters/master/'
  const ignoreList = ['future']

  const getFilters = async (): Promise<Record<string, any>[]> => {
    const data = await $fetch<Record<string, any>[]>(
      'https://api.github.com/repos/bedrock-oss/regolith-filters/contents/'
    )
    const filters = data.filter((filter) => filter.type === 'dir')
    return filters
  }
  const getFilterLink = (name: string): string => baseUrl + name
  const getFilterDescription = async (name: string): Promise<string> => {
    const filterFileUrl = rawBaseUrl + name + '/filter.json'
    const data = await $fetch(filterFileUrl, { parseResponse: JSON.parse })
    return data.description || 'No description.'
  }
  const formatFilters = async (): Promise<string[]> => {
    let columns: string[] = []
    for (const { name } of await getFilters()) {
      if (ignoreList.includes(name)) continue
      console.log(name)
      columns.push(
        `| [${name}](${getFilterLink(name)}) | ${await getFilterDescription(
          name
        )} |`
      )
    }
    return columns
  }

  const fileContent = [
    ...headerContent,
    '',
    '| Filter | Description |',
    '| ------ | ----------- |',
    ...(await formatFilters()),
    '',
    ...footerContent,
    ''
  ].join('\n')

  await writeFile(join(resolve(), 'docs', 'standard-library.md'), fileContent)
}

await writeStandardLibrary()
