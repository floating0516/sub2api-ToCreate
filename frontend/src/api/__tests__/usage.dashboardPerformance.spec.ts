import { beforeEach, describe, expect, it, vi } from 'vitest'

const { get } = vi.hoisted(() => ({
  get: vi.fn(),
}))

vi.mock('@/api/client', () => ({
  apiClient: {
    get,
  },
}))

import { usageAPI } from '@/api/usage'

describe('dashboard usage api performance options', () => {
  beforeEach(() => {
    get.mockReset()
    get.mockResolvedValue({ data: {} })
  })

  it('requests range totals without endpoint breakdowns', async () => {
    await usageAPI.getStatsSummaryByDateRange('2026-08-01', '2026-08-07')

    expect(get).toHaveBeenCalledWith('/usage/stats', {
      params: {
        start_date: '2026-08-01',
        end_date: '2026-08-07',
        summary_only: true,
      },
    })
  })

  it('requests the lightweight lifetime dashboard summary', async () => {
    await usageAPI.getDashboardStats({ summary_only: true })

    expect(get).toHaveBeenCalledWith('/usage/dashboard/stats', {
      params: { summary_only: true },
    })
  })

  it('loads all top model series through one endpoint', async () => {
    await usageAPI.getDashboardModelTrends({
      start_date: '2026-08-01',
      end_date: '2026-08-07',
      granularity: 'day',
      limit: 8,
    })

    expect(get).toHaveBeenCalledWith('/usage/dashboard/model-trends', {
      params: {
        start_date: '2026-08-01',
        end_date: '2026-08-07',
        granularity: 'day',
        limit: 8,
      },
    })
  })
})
