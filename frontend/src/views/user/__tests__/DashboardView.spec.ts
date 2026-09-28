import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, shallowMount } from '@vue/test-utils'

import DashboardView from '../DashboardView.vue'

const getDashboardStats = vi.hoisted(() => vi.fn())
const getDashboardModelTrends = vi.hoisted(() => vi.fn())
const getDashboardTrend = vi.hoisted(() => vi.fn())
const getPaymentSummary = vi.hoisted(() => vi.fn())
const refreshUser = vi.hoisted(() => vi.fn())

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
      locale: { value: 'zh-CN' }
    })
  }
})

vi.mock('@/api/usage', () => ({
  usageAPI: {
    getDashboardStats,
    getDashboardModelTrends,
    getDashboardTrend
  }
}))

vi.mock('@/api/payment', () => ({
  paymentAPI: {
    getSummary: getPaymentSummary
  }
}))

vi.mock('@/api/keys', () => ({
  keysAPI: {
    list: vi.fn()
  }
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    user: {
      balance: 128.5,
      created_at: '2026-08-01T00:00:00+08:00'
    },
    refreshUser
  })
}))

describe('user DashboardView', () => {
  beforeEach(() => {
    getDashboardStats.mockReset()
    getDashboardModelTrends.mockReset()
    getDashboardTrend.mockReset()
    getPaymentSummary.mockReset()
    refreshUser.mockReset()

    getDashboardStats.mockResolvedValue({
      total_requests: 12,
      total_tokens: 3456,
      total_input_tokens: 2000,
      total_output_tokens: 1000,
      total_cache_creation_tokens: 200,
      total_cache_read_tokens: 256,
      today_actual_cost: 4.82,
      average_duration_ms: 850,
      rpm: 1,
      tpm: 20
    })
    getDashboardModelTrends.mockResolvedValue({ trend: [] })
    getDashboardTrend.mockResolvedValue({ trend: [] })
    getPaymentSummary.mockResolvedValue({
      net_paid: 100,
      balance_paid: 80,
      platform_granted: 20
    })
    refreshUser.mockResolvedValue(undefined)
  })

  it('shows the user-timezone daily usage amount beside the available balance', async () => {
    const wrapper = shallowMount(DashboardView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: true,
          DashboardDateRangePicker: true,
          DashboardUsageCalendar: true,
          DashboardTrendChart: true
        }
      }
    })

    await flushPromises()

    expect(getDashboardStats).toHaveBeenCalledWith({
      summary_only: true,
      timezone: expect.any(String)
    })
    const accountCard = wrapper.findAll('.dashboard-metric-card')[0]
    expect(accountCard.get('.dashboard-metric-scope-detail').text()).toContain('dashboard.overview.todayUsageAmount')
    expect(accountCard.get('.dashboard-metric-scope-detail').text()).toContain('4.82')
  })

  it('labels hourly usage with the bucket start time', async () => {
    const wrapper = shallowMount(DashboardView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: true,
          DashboardDateRangePicker: true,
          DashboardUsageCalendar: true,
          DashboardTrendChart: true
        }
      }
    })

    await flushPromises()
    const hourlyButton = wrapper.findAll('.dashboard-segmented button')
      .find((button) => button.text() === 'dashboard.overview.hourly')
    expect(hourlyButton).toBeDefined()
    await hourlyButton!.trigger('click')
    await flushPromises()

    const labels = wrapper.getComponent({ name: 'DashboardTrendChart' }).props('labels') as string[]
    expect(labels[0]).toBe('00:00')
    expect(labels[23]).toBe('23:00')
  })

  it('orders model series by total token usage instead of trusting response rank', async () => {
    getDashboardModelTrends.mockResolvedValue({
      trend: [
        { date: '2026-08-01', model: 'model-low', rank: 1, requests: 1, total_tokens: 20 },
        { date: '2026-08-01', model: 'model-high', rank: 3, requests: 1, total_tokens: 90 },
        { date: '2026-08-02', model: 'model-low', rank: 1, requests: 1, total_tokens: 10 },
        { date: '2026-08-02', model: 'model-high', rank: 3, requests: 1, total_tokens: 60 }
      ]
    })

    const wrapper = shallowMount(DashboardView, {
      global: {
        stubs: {
          AppLayout: { template: '<div><slot /></div>' },
          Icon: true,
          DashboardDateRangePicker: true,
          DashboardUsageCalendar: true,
          DashboardTrendChart: true
        }
      }
    })

    await flushPromises()

    const series = wrapper.getComponent({ name: 'DashboardTrendChart' }).props('series') as Array<{
      label: string
      color: string
      total: number
    }>
    expect(series.map((item) => item.label)).toEqual(['model-high', 'model-low'])
    expect(series.map((item) => item.total)).toEqual([150, 30])
    expect(new Set(series.map((item) => item.color)).size).toBe(2)
    expect(wrapper.findAll('.dashboard-legend-rank').map((item) => item.text())).toEqual(['1', '2'])
  })
})
