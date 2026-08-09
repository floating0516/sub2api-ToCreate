import { beforeEach, describe, expect, it, vi } from 'vitest'
import { apiClient } from '@/api/client'
import { getOverview } from '@/api/admin/accountContributions'

vi.mock('@/api/client', () => ({
  apiClient: {
    get: vi.fn(),
  },
}))

describe('admin account contributions API', () => {
  beforeEach(() => {
    vi.mocked(apiClient.get).mockReset()
  })

  it('uses the administrator-only overview endpoint', async () => {
    const payload = { features: {}, stats: {}, contributors: [], contributions: [], earnings: [], payouts: [] }
    vi.mocked(apiClient.get).mockResolvedValue({ data: payload })

    await expect(getOverview()).resolves.toBe(payload)
    expect(apiClient.get).toHaveBeenCalledWith('/admin/account-contributions/overview')
  })
})
