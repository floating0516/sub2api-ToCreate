import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

const { getConfig, updateConfig, testConfig } = vi.hoisted(() => ({
  getConfig: vi.fn(),
  updateConfig: vi.fn(),
  testConfig: vi.fn()
}))

vi.mock('@/api/admin/customBuild', () => ({
  getCustomUpdateResolverConfig: getConfig,
  testCustomUpdateResolverConfig: testConfig,
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
    testConfig.mockReset()
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

  it('tests a newly entered key without clearing the input', async () => {
    testConfig.mockResolvedValue({
      ok: true,
      model: 'gpt-5.6-luna',
      latency_ms: 248
    })
    const wrapper = mount(CustomUpdateResolverSettings)
    await flushPromises()

    const input = wrapper.get('input[type="password"]')
    await input.setValue('sk-current-secret')
    await wrapper.get('[data-testid="custom-update-resolver-test"]').trigger('click')
    await flushPromises()

    expect(testConfig).toHaveBeenCalledWith({
      base_url: 'https://api.lihe.chat',
      model: 'gpt-5.6-luna',
      api_key: 'sk-current-secret'
    })
    expect((input.element as HTMLInputElement).value).toBe('sk-current-secret')
    expect(wrapper.text()).toContain('version.customUpdateResolverTestSuccess')
  })

  it('uses the saved key when the API key field is blank', async () => {
    getConfig.mockResolvedValue({
      ...defaultConfig,
      api_key_configured: true,
      saved: true
    })
    testConfig.mockResolvedValue({
      ok: true,
      model: 'gpt-5.6-luna',
      latency_ms: 91
    })
    const wrapper = mount(CustomUpdateResolverSettings)
    await flushPromises()

    await wrapper.get('[data-testid="custom-update-resolver-test"]').trigger('click')
    await flushPromises()

    expect(testConfig).toHaveBeenCalledWith({
      base_url: 'https://api.lihe.chat',
      model: 'gpt-5.6-luna'
    })
  })

  it('disables connection testing until an API key is available', async () => {
    const wrapper = mount(CustomUpdateResolverSettings)
    await flushPromises()

    expect(
      wrapper.get('[data-testid="custom-update-resolver-test"]').attributes('disabled')
    ).toBeDefined()
  })

  it('shows a safe connection failure message', async () => {
    testConfig.mockRejectedValue({
      response: { data: { message: 'The API key was rejected by the upstream service' } }
    })
    const wrapper = mount(CustomUpdateResolverSettings)
    await flushPromises()

    await wrapper.get('input[type="password"]').setValue('sk-rejected-secret')
    await wrapper.get('[data-testid="custom-update-resolver-test"]').trigger('click')
    await flushPromises()

    expect(wrapper.text()).toContain('The API key was rejected by the upstream service')
    expect(wrapper.html()).not.toContain('sk-rejected-secret')
  })
})
