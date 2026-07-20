import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import OpenAIQuotaUsageChart from '../OpenAIQuotaUsageChart.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

vi.mock('vue-chartjs', () => ({
  Line: {
    name: 'Line',
    props: ['data', 'options', 'plugins'],
    template: '<div data-testid="line-chart"></div>'
  }
}))

describe('OpenAIQuotaUsageChart', () => {
  it('maps samples to a linear time axis bounded to 0-100 percent', () => {
    const wrapper = mount(OpenAIQuotaUsageChart, {
      props: {
        samples: [
          {
            cycle_id: 1,
            observed_at: '2026-07-18T10:00:00Z',
            used_percent: 16,
            local_tokens: 1200
          },
          {
            cycle_id: 1,
            observed_at: '2026-07-18T10:05:00Z',
            used_percent: 18.5,
            local_tokens: 2500000
          }
        ],
        resetTimes: []
      }
    })

    const line = wrapper.getComponent({ name: 'Line' })
    const data = line.props('data') as any
    const options = line.props('options') as any
    expect(data.datasets[0].data).toEqual([
      { x: Date.parse('2026-07-18T10:00:00Z'), y: 16, localTokens: 1200 },
      { x: Date.parse('2026-07-18T10:05:00Z'), y: 18.5, localTokens: 2500000 }
    ])
    expect(options.scales.x.type).toBe('linear')
    expect(options.scales.y.min).toBe(0)
    expect(options.scales.y.max).toBe(100)
    const tooltipLines = options.plugins.tooltip.callbacks.label({
      parsed: { y: 18.5 },
      raw: data.datasets[0].data[1]
    })
    expect(tooltipLines).toEqual([
      'admin.accounts.openaiQuotaHistory.usageLegend: 18.5%',
      `admin.accounts.openaiQuotaHistory.localTokens: ${Number(2500000).toLocaleString()}`
    ])
  })

  it('draws reset timestamps as vertical red dashed markers', () => {
    const wrapper = mount(OpenAIQuotaUsageChart, {
      props: {
        samples: [
          {
            cycle_id: 1,
            observed_at: '2026-07-18T10:00:00Z',
            used_percent: 16,
            local_tokens: 1200
          },
          {
            cycle_id: 2,
            observed_at: '2026-07-18T10:10:00Z',
            used_percent: 0,
            local_tokens: 0
          }
        ],
        resetTimes: ['2026-07-18T10:05:00Z']
      }
    })

    const line = wrapper.getComponent({ name: 'Line' })
    const plugin = (line.props('plugins') as any[])[0]
    const ctx = {
      save: vi.fn(),
      restore: vi.fn(),
      setLineDash: vi.fn(),
      beginPath: vi.fn(),
      moveTo: vi.fn(),
      lineTo: vi.fn(),
      stroke: vi.fn(),
      strokeStyle: '',
      lineWidth: 0
    }
    plugin.afterDatasetsDraw({
      ctx,
      chartArea: { left: 0, right: 100, top: 10, bottom: 90 },
      scales: { x: { getPixelForValue: () => 50 } }
    })

    expect(ctx.setLineDash).toHaveBeenCalledWith([5, 4])
    expect(ctx.moveTo).toHaveBeenCalledWith(50, 10)
    expect(ctx.lineTo).toHaveBeenCalledWith(50, 90)
    expect(ctx.stroke).toHaveBeenCalledOnce()
    expect(ctx.strokeStyle).toMatch(/#(dc2626|f87171)/)
  })

  it('shows an empty state when no samples have been collected', () => {
    const wrapper = mount(OpenAIQuotaUsageChart, {
      props: { samples: [], resetTimes: [] }
    })

    expect(wrapper.findComponent({ name: 'Line' }).exists()).toBe(false)
    expect(wrapper.text()).toContain('admin.accounts.openaiQuotaHistory.noSamples')
  })
})
