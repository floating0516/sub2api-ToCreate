import type { CustomUpdateStatus } from '@/api/admin/customBuild'

type CustomReleaseStatus = Pick<
  CustomUpdateStatus,
  'release_status' | 'release_tag' | 'release_url'
>

export function resolveCustomUpdateReleaseURL(
  status: CustomReleaseStatus | null | undefined
): string {
  if (status?.release_status !== 'published') return ''

  const releaseTag = status.release_tag?.trim()
  const releaseURL = status.release_url?.trim()
  if (!releaseTag || !releaseURL) return ''

  try {
    const parsed = new URL(releaseURL)
    const pathParts = parsed.pathname.split('/').filter(Boolean)
    const tag = pathParts[4] ? decodeURIComponent(pathParts[4]) : ''
    const isGitHubRelease =
      parsed.protocol === 'https:' &&
      parsed.hostname === 'github.com' &&
      parsed.port === '' &&
      parsed.username === '' &&
      parsed.password === '' &&
      pathParts.length === 5 &&
      pathParts[2] === 'releases' &&
      pathParts[3] === 'tag' &&
      tag === releaseTag

    return isGitHubRelease ? parsed.toString() : ''
  } catch {
    return ''
  }
}
