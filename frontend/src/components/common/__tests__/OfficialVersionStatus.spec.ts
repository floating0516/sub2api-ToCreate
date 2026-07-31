import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import OfficialVersionStatus from '../OfficialVersionStatus.vue'
import {
  OFFICIAL_RELEASES_URL,
  OFFICIAL_REPOSITORY_URL
} from '@/constants/version'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string, params?: Record<string, string>) =>
        params?.version ? `${key}:${params.version}` : key
    })
  }
})

describe('OfficialVersionStatus', () => {
  it('links to the official repository and release when an update is available', () => {
    const wrapper = mount(OfficialVersionStatus, {
      props: {
        currentVersion: '0.1.168',
        latestVersion: '0.1.169',
        hasUpdate: true,
        releaseUrl: 'https://github.com/Wei-Shaw/sub2api/releases/tag/v0.1.169'
      }
    })

    expect(wrapper.get('[data-testid="official-repository-link"]').attributes('href')).toBe(
      OFFICIAL_REPOSITORY_URL
    )
    expect(wrapper.get('[data-testid="official-update-link"]').attributes('href')).toBe(
      'https://github.com/Wei-Shaw/sub2api/releases/tag/v0.1.169'
    )
    expect(wrapper.text()).toContain('version.officialUpdateAvailable:0.1.169')
    expect(wrapper.text()).toContain('version.officialUpdateCustomHint')
  })

  it('falls back to the official releases page when release metadata is unavailable', () => {
    const wrapper = mount(OfficialVersionStatus, {
      props: {
        currentVersion: '0.1.168',
        latestVersion: '0.1.169',
        hasUpdate: true
      }
    })

    expect(wrapper.get('[data-testid="official-update-link"]').attributes('href')).toBe(
      OFFICIAL_RELEASES_URL
    )
  })

  it('shows a clear status when the official version check has not returned data', () => {
    const wrapper = mount(OfficialVersionStatus, {
      props: {
        currentVersion: '0.1.168'
      }
    })

    expect(wrapper.find('[data-testid="official-update-link"]').exists()).toBe(false)
    expect(wrapper.get('[data-testid="official-version-unavailable"]').text()).toContain(
      'version.officialCheckUnavailable'
    )
  })
})
