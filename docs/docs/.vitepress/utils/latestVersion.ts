import { $fetch } from 'ohmyfetch'

export async function getLatestVersion(): Promise<string> {
  const result = await $fetch<{
    tag_name: string
  }>('https://api.github.com/repos/Bedrock-OSS/regolith/releases/latest')

  return `v${result.tag_name}`
}
