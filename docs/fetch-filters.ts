import { writeFile } from 'node:fs/promises'
import { join, resolve } from 'node:path'
import { $fetch } from 'ohmyfetch'

const COMMENT = `<!-- This page is auto-generated. To edit it, you'll need to change the "docs/fetch-filters.ts" file -->`

async function writeStandardLibrary(): Promise<void> {
  console.log('- Fetching Standard Library -')

  const headerContent = [
    '---',
    'title: Standard Library',
    '---',
    '',
    COMMENT,
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
  console.log('Updated Standard Library')
}

interface Filter {
  name: string
  author: string
  lang: string
  description: string
  url
}

async function writeCommunityFilters(): Promise<void> {
  console.log('- Fetching Community Filters -')

  const headerContent = [
    '---',
    'title: Community Filters',
    '---',
    '',
    COMMENT,
    '',
    '# Community Filters',
    '',
    `The beauty of Regolith is that filters can be written and shared by anyone! This page contains an uncurated list of community filters. If your filter doesn't appear here, [let us know](https://discord.com/invite/XjV87YN)!`,
    '',
    '## Installing Community Filters',
    'Community filters are installed via a URL-like resource definition: `github.com/<username>/<repository>/<folder>`.',
    '',
    'For example `github.com/SirLich/echo-npc-regolith/echo`.',
    '',
    '::: warning',
    'Please use extreme caution when running unknown code. Regolith and its maintainers take no responsibility for any damages incurred by using the filters on this page. To learn more, please read our [safety page](/guide/safety).',
    ':::',
    '',
    '::: tip',
    'Having trouble? You can learn more about online filters [here](/guide/online-filters).',
    ':::',
    '',
    '## Filters'
  ]

  const findReadmeDescription = (content: string): string => {
    return (
      content.split('\n').find((line) => {
        if (!line.startsWith('#') && line !== '') return line
      }) || 'No description.'
    )
  }
  const fetchFilters = async (): Promise<Filter[]> => {
    const filters: Filter[] = []

    const repos = await $fetch(
      'https://api.github.com/search/repositories?q=topic:regolith-filter'
    )

    for (const repo of repos.items) {
      const author = repo.owner.login
      if (author === 'Bedrock-OSS') continue

      const rootDir = await $fetch(
        `https://api.github.com/repos/${repo.full_name}/contents`
      )
      for (const rootItem of rootDir) {
        let name = ''
        let lang = ''
        let description = ''
        let url = ''

        if (rootItem.type === 'dir') {
          const childDir = await $fetch(rootItem.url)

          for (const childItem of childDir) {
            switch (childItem.name) {
              case 'filter.json':
                const filter = await $fetch(childItem.download_url, {
                  parseResponse: JSON.parse
                })

                name = rootItem.name
                lang = filter.filters[0].runWith
                url = rootItem.html_url
                description =
                  filter?.filters[0]?.description ||
                  filter?.description ||
                  'No description.'
                continue

              case 'readme.md':
              case 'README.md':
                const readme = await $fetch(childItem.download_url)
                description = findReadmeDescription(readme)
                continue

              default:
                continue
            }
          }
        } else continue

        if (name && author && lang && description && url) {
          console.log(name)
          filters.push({
            name,
            author,
            lang,
            description,
            url
          })
        }
      }
    }

    return filters
  }
  const formatFilters = async (): Promise<string[]> => {
    const filters = await fetchFilters()
    const formattedFilters = filters.map(
      ({ author, description, lang, name, url }) => {
        return `| [${name}](${url}) | ${author} | ${lang} | ${description} |`
      }
    )
    return formattedFilters
  }

  const content = [
    ...headerContent,
    '',
    '| Name | Author | Language | Description |',
    '| ---- | ------ | -------- | ----------- |',
    ...(await formatFilters()),
    ''
  ].join('\n')

  await writeFile(join(resolve(), 'docs', 'community-filters.md'), content)
  console.log('Updated Community Filters')
}

await writeStandardLibrary()
await writeCommunityFilters()
