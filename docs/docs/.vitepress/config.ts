import { defineConfig } from 'vitepress'
import { getLatestVersion } from './utils/latestVersion'

const title = 'Regolith'
const description =
  'A flexible and language-agnostic addon-compiler for the Bedrock Edition of Minecraft.'
const url = 'https://bedrock-oss.github.io/regolith/'

export default defineConfig({
  title,
  description,
  lastUpdated: true,
  ignoreDeadLinks: true,
  cleanUrls: 'with-subfolders',
  base: '/regolith/',

  head: [
    ['link', { rel: 'icon', type: 'image/x-icon', href: 'favicon.ico' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:title', content: title }],
    ['meta', { property: 'og:url', content: url }],
    ['meta', { property: 'twitter:description', content: description }],
    ['meta', { property: 'twitter:title', content: title }],
    ['meta', { property: 'twitter:card', content: 'summary_large_image' }],
    ['meta', { property: 'twitter:url', content: url }]
  ],

  themeConfig: {
    logo: '/logo.png',

    editLink: {
      text: 'Suggest changes to this page.',
      pattern:
        'https://github.com/Bedrock-OSS/regolith/edit/main/docs/docs/:path'
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/Bedrock-OSS/regolith' },
      { icon: 'discord', link: 'https://discord.gg/XjV87YN' }
    ],

    footer: {
      message: 'Released under the MIT license.',
      copyright: `Copyright © ${new Date().getFullYear()} Bedrock OSS.`
    },

    nav: [
      {
        text: 'Guide',
        link: '/guide/what-is-regolith',
        activeMatch: '/guide/'
      },
      {
        text: 'Standard Library',
        link: '/standard-library'
      },
      {
        text: 'Community Filters',
        link: '/community-filters'
      },
      {
        text: 'Resources',
        items: [
          {
            text: 'Project Config Standard',
            link: 'https://github.com/Bedrock-OSS/project-config-standard'
          }
        ]
      },
      {
        text: await getLatestVersion(),
        items: [
          {
            text: 'Release Notes',
            link: 'https://github.com/Bedrock-OSS/regolith/releases'
          }
        ]
      }
    ],

    sidebar: {
      '/guide/': [
        {
          text: 'Introduction',
          collapsible: true,
          items: [
            {
              text: 'What Is Regolith?',
              link: '/guide/what-is-regolith'
            },
            {
              text: 'Installing',
              link: '/guide/installing'
            },
            {
              text: 'Getting Started',
              link: '/guide/getting-started'
            },
            {
              text: 'Troubleshooting',
              link: '/guide/troubleshooting'
            }
          ]
        },
        {
          text: 'Advanced',
          collapsible: true,
          items: [
            {
              text: 'Configuration File',
              link: '/guide/configuration'
            },
            {
              text: 'User Configuration',
              link: '/guide/user-configuration'
            },
            {
              text: 'Data Folder',
              link: '/guide/data-folder'
            },
            {
              text: 'Export Targets',
              link: '/guide/export-targets'
            },
            {
              text: 'Profiles',
              link: '/guide/profiles'
            },
            {
              text: 'Safety',
              link: '/guide/safety'
            }
          ]
        },
        {
          text: 'Filters',
          collapsible: true,
          items: [
            {
              text: 'Introduction',
              link: '/guide/filters'
            },
            {
              text: 'Local Filters',
              link: '/guide/local-filters'
            },
            {
              text: 'Custom Filters',
              link: '/guide/custom-filters'
            },
            {
              text: 'Online Filters',
              link: '/guide/online-filters'
            },
            {
              text: 'Installing Filters',
              link: '/guide/installing-filters'
            },
            {
              text: 'Filter Run Modes',
              link: '/guide/filter-run-modes'
            },
            {
              text: 'Create a Filter',
              link: '/guide/create-a-filter'
            }
          ]
        },
        {
          text: 'Filter Types',
          collapsible: true,
          items: [
            {
              text: 'Java Filters',
              link: '/guide/java-filters'
            },
            {
              text: '.NET Filters',
              link: '/guide/dotnet-filters'
            },
            {
              text: 'Nim Filters',
              link: '/guide/nim-filters'
            },
            {
              text: 'Python Filters',
              link: '/guide/python-filters'
            },
            {
              text: 'Shell Filters',
              link: '/guide/shell-filters'
            },
            {
              text: 'NodeJS Filters',
              link: '/guide/node-filters'
            },
            {
              text: 'Deno Filters',
              link: '/guide/deno-filters'
            },
            {
              text: 'Profile Filters',
              link: '/guide/profile-filters'
            }
          ]
        }
      ]
    }
  }
})
