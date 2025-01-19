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
    }
  }
})
