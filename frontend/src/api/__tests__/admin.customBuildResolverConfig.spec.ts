import { beforeEach, describe, expect, it, vi } from 'vitest'

const client = vi.hoisted(() => ({
  get: vi.fn(),
  put: vi.fn(),
  post: vi.fn()
}))

vi.mock('@/api/client', () => ({ apiClient: client }))

import {
  getCustomUpdateResolverConfig,
  testCustomUpdateResolverConfig,
  updateCustomUpdateResolverConfig
} from '@/api/admin/customBuild'

describe('custom update resolver config API', () => {
  beforeEach(() => {
    client.get.mockReset()
    client.put.mockReset()
    client.post.mockReset()
  })

  it('loads resolver settings without expecting a returned secret', async () => {
    const response = {
      base_url: 'https://api.lihe.chat',
      model: 'gpt-5.6-luna',
      reasoning_effort: 'max',
      api_key_configured: true,
      saved: true,
      default_base_url: 'https://api.lihe.chat',
      default_model: 'gpt-5.6-luna'
    }
    client.get.mockResolvedValue({ data: response })

    const result = await getCustomUpdateResolverConfig()

    expect(client.get).toHaveBeenCalledWith('/admin/custom-build/update/resolver-config')
    expect(result).toEqual(response)
    expect(result).not.toHaveProperty('api_key')
  })

  it('sends a new key only in the save request', async () => {
    const request = {
      base_url: 'https://api.lihe.chat',
      model: 'gpt-5.6-terra',
      api_key: 'sk-resolver-secret'
    }
    const response = {
      base_url: request.base_url,
      model: request.model,
      reasoning_effort: 'max',
      api_key_configured: true,
      saved: true,
      default_base_url: 'https://api.lihe.chat',
      default_model: 'gpt-5.6-luna'
    }
    client.put.mockResolvedValue({ data: response })

    const result = await updateCustomUpdateResolverConfig(request)

    expect(client.put).toHaveBeenCalledWith(
      '/admin/custom-build/update/resolver-config',
      request
    )
    expect(JSON.stringify(result)).not.toContain(request.api_key)
  })

  it('sends the current form values to the connection test endpoint', async () => {
    const request = {
      base_url: 'https://api.lihe.chat',
      model: 'gpt-5.6-luna',
      api_key: 'sk-current-secret'
    }
    const response = { ok: true, model: request.model, latency_ms: 248 }
    client.post.mockResolvedValue({ data: response })

    const result = await testCustomUpdateResolverConfig(request)

    expect(client.post).toHaveBeenCalledWith(
      '/admin/custom-build/update/resolver-config/test',
      request,
      { timeout: 65000 }
    )
    expect(result).toEqual(response)
    expect(JSON.stringify(result)).not.toContain(request.api_key)
  })
})
