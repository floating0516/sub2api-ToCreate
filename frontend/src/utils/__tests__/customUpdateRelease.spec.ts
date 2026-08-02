import { describe, expect, it } from 'vitest'
import { resolveCustomUpdateReleaseURL } from '../customUpdateRelease'

describe('custom update release URL', () => {
  it('accepts the published GitHub Release matching the reported tag', () => {
    const releaseURL =
      'https://github.com/floating0516/sub2api-ToCreate/releases/tag/tocreate-v0.1.169-tc1.24'

    expect(
      resolveCustomUpdateReleaseURL({
        release_status: 'published',
        release_tag: 'tocreate-v0.1.169-tc1.24',
        release_url: releaseURL
      })
    ).toBe(releaseURL)
  })

  it('rejects mismatched, failed, and non-GitHub release URLs', () => {
    expect(
      resolveCustomUpdateReleaseURL({
        release_status: 'published',
        release_tag: 'tocreate-v0.1.169-tc1.24',
        release_url:
          'https://github.com/floating0516/sub2api-ToCreate/releases/tag/tocreate-v0.1.168-tc1.23'
      })
    ).toBe('')
    expect(
      resolveCustomUpdateReleaseURL({
        release_status: 'failed',
        release_tag: 'tocreate-v0.1.169-tc1.24',
        release_url:
          'https://github.com/floating0516/sub2api-ToCreate/releases/tag/tocreate-v0.1.169-tc1.24'
      })
    ).toBe('')
    expect(
      resolveCustomUpdateReleaseURL({
        release_status: 'published',
        release_tag: 'tocreate-v0.1.169-tc1.24',
        release_url: 'javascript:alert(1)'
      })
    ).toBe('')
  })
})
