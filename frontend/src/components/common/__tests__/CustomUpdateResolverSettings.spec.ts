import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const { getConfig, updateConfig } = vi.hoisted(() => ({
  getConfig: vi.fn(),
  updateConfig: vi.fn()
}))

vi.mock('@/api/admin/customBuild', () => ({
  getCustomUpdateResolverConfig: getConfig,
  updateCustomUpdateResolverConfig: updateConfig
}))

vi.mock('vue-i18n', () => ({
  useI18n: () => ({ t: (key: string) => key })
}))

import CustomUpdateResolverSettings from '../CustomUpdateResolverSettings.vue'

const defaultConfig = {
  base_url: 'https://api.lihe.chat',
  model: 'gpt-5.6-luna',
  reasoning_effort: 'max',
  api_key_configured: false,
  saved: false,
  default_base_url: 'https://api.lihe.chat',
  default_model: 'gpt-5.6-luna'
}

describe('CustomUpdateResolverSettings', () => {
  beforeEach(() => {
    getConfig.mockReset()
    updateConfig.mockReset()
    getConfig.mockResolvedValue({ ...defaultConfig })
  })

  it('saves Base URL, API key, and the selected model, then clears the secret input', async () => {
    updateConfig.mockResolvedValue({
      ...defaultConfig,
      model: 'gpt-5.6-terra',
      api_key_configured: true,
      saved: true,
      updated_at: '2026-08-01T08:00:00Z'
    })
    const wrapper = mount(CustomUpdateResolverSettings)
    await flushPromises()

    await wrapper.get('input[type="password"]').setValue('sk-resolver-secret')
    await wrapper.get('select').setValue('gpt-5.6-terra')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(updateConfig).toHaveBeenCalledWith({
      base_url: 'https://api.lihe.chat',
      model: 'gpt-5.6-terra',
      api_key: 'sk-resolver-secret'
    })
    expect((wrapper.get('input[type="password"]').element as HTMLInputElement).value).toBe('')
    expect(wrapper.html()).not.toContain('sk-resolver-secret')
  })

  it('keeps the existing API key when the secret field is left blank', async () => {
    getConfig.mockResolvedValue({
      ...defaultConfig,
      api_key_configured: true,
      saved: true
    })
    updateConfig.mockResolvedValue({
      ...defaultConfig,
      api_key_configured: true,
      saved: true
    })
    const wrapper = mount(CustomUpdateResolverSettings)
    await flushPromises()

    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(updateConfig).toHaveBeenCalledWith({
      base_url: 'https://api.lihe.chat',
      model: 'gpt-5.6-luna'
    })
  })
})
