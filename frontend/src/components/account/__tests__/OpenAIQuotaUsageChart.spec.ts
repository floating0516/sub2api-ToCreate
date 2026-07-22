import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { formatCurrency } from '@/utils/format'
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
            local_cost_usd: 1.2
          },
          {
            cycle_id: 1,
            observed_at: '2026-07-18T10:05:00Z',
            used_percent: 18.5,
            local_cost_usd: 2500.125
          }
        ],
        resetMarkers: []
      }
    })

    const line = wrapper.getComponent({ name: 'Line' })
    const data = line.props('data') as any
    const options = line.props('options') as any
    expect(data.datasets[0].data).toEqual([
      { x: Date.parse('2026-07-18T10:00:00Z'), y: 16, localCostUSD: 1.2 },
      { x: Date.parse('2026-07-18T10:05:00Z'), y: 18.5, localCostUSD: 2500.125 }
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
      `admin.accounts.openaiQuotaHistory.localUsage: ${formatCurrency(2500.125)}`
    ])
  })

  it('draws manual, official, and unknown resets in distinct colors', () => {
    const wrapper = mount(OpenAIQuotaUsageChart, {
      props: {
        samples: [
          {
            cycle_id: 1,
            observed_at: '2026-07-18T10:00:00Z',
            used_percent: 16,
            local_cost_usd: 1.2
          },
          {
            cycle_id: 2,
            observed_at: '2026-07-18T10:10:00Z',
            used_percent: 0,
            local_cost_usd: 0
          }
        ],
        resetMarkers: [
          { cycleId: 1, observedAt: '2026-07-18T10:03:00Z', source: 'manual' },
          { cycleId: 2, observedAt: '2026-07-18T10:05:00Z', source: 'provider' },
          { cycleId: 3, observedAt: '2026-07-18T10:07:00Z', source: 'unknown' }
        ]
      }
    })

    const line = wrapper.getComponent({ name: 'Line' })
    const plugin = (line.props('plugins') as any[])[0]
    const strokeStyles: string[] = []
    const ctx: any = {
      save: vi.fn(),
      restore: vi.fn(),
      setLineDash: vi.fn(),
      beginPath: vi.fn(),
      moveTo: vi.fn(),
      lineTo: vi.fn(),
      stroke: vi.fn(),
      lineWidth: 0
    }
    Object.defineProperty(ctx, 'strokeStyle', {
      get: () => strokeStyles[strokeStyles.length - 1] ?? '',
      set: (value: string) => strokeStyles.push(value)
    })
    plugin.afterDatasetsDraw({
      ctx,
      chartArea: { left: 0, right: 100, top: 10, bottom: 90 },
      scales: {
        x: {
          getPixelForValue: (value: number) => (
            value === Date.parse('2026-07-18T10:03:00Z')
              ? 30
              : value === Date.parse('2026-07-18T10:05:00Z') ? 50 : 70
          )
        }
      }
    })

    expect(ctx.setLineDash).toHaveBeenCalledWith([5, 4])
    expect(ctx.moveTo).toHaveBeenNthCalledWith(1, 30, 10)
    expect(ctx.moveTo).toHaveBeenNthCalledWith(2, 50, 10)
    expect(ctx.moveTo).toHaveBeenNthCalledWith(3, 70, 10)
    expect(ctx.lineTo).toHaveBeenNthCalledWith(1, 30, 90)
    expect(ctx.lineTo).toHaveBeenNthCalledWith(2, 50, 90)
    expect(ctx.lineTo).toHaveBeenNthCalledWith(3, 70, 90)
    expect(ctx.stroke).toHaveBeenCalledTimes(3)
    expect(strokeStyles[0]).toMatch(/#(16a34a|4ade80)/)
    expect(strokeStyles[1]).toMatch(/#(dc2626|f87171)/)
    expect(strokeStyles[2]).toMatch(/#(6b7280|9ca3af)/)
  })

  it('emits the selected marker when its dashed line is clicked', () => {
    const wrapper = mount(OpenAIQuotaUsageChart, {
      props: {
        samples: [
          {
            cycle_id: 1,
            observed_at: '2026-07-18T10:00:00Z',
            used_percent: 16,
            local_cost_usd: 1.2
          }
        ],
        resetMarkers: [
          { cycleId: 7, observedAt: '2026-07-18T10:03:00Z', source: 'unknown' }
        ]
      }
    })

    const line = wrapper.getComponent({ name: 'Line' })
    const plugin = (line.props('plugins') as any[])[0]
    const canvas = { style: { cursor: '' } }
    const chart = {
      canvas,
      chartArea: { left: 0, right: 100, top: 10, bottom: 90 },
      scales: { x: { getPixelForValue: () => 40 } }
    }
    plugin.afterEvent(chart, { event: { type: 'mousemove', x: 43, y: 50 } })
    expect(canvas.style.cursor).toBe('pointer')
    plugin.afterEvent(chart, { event: { type: 'click', x: 43, y: 50 } })
    expect(wrapper.emitted('reset-marker-click')).toEqual([[
      {
        cycleId: 7,
        observedAt: '2026-07-18T10:03:00Z',
        source: 'unknown'
      }
    ]])
  })

  it('shows an empty state when no samples have been collected', () => {
    const wrapper = mount(OpenAIQuotaUsageChart, {
      props: { samples: [], resetMarkers: [] }
    })

    expect(wrapper.findComponent({ name: 'Line' }).exists()).toBe(false)
    expect(wrapper.text()).toContain('admin.accounts.openaiQuotaHistory.noSamples')
  })
})
