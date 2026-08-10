import { describe, expect, it, vi } from 'vitest'
import { mount } from '@vue/test-utils'

import DashboardTrendChart from '../DashboardTrendChart.vue'

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

vi.mock('echarts/core', () => ({ use: vi.fn() }))
vi.mock('echarts/charts', () => ({ LineChart: {} }))
vi.mock('echarts/components', () => ({
  AriaComponent: {},
  GridComponent: {},
  TooltipComponent: {}
}))
vi.mock('echarts/features', () => ({ UniversalTransition: {} }))
vi.mock('echarts/renderers', () => ({ CanvasRenderer: {} }))
vi.mock('vue-echarts', async () => {
  const { defineComponent } = await vi.importActual<typeof import('vue')>('vue')
  return {
    default: defineComponent({
      name: 'VChart',
      props: {
        option: { type: Object, required: true },
        updateOptions: { type: Object, required: true },
        autoresize: { type: [Boolean, Object], default: false }
      },
      template: '<div class="v-chart-stub" />'
    })
  }
})

describe('DashboardTrendChart', () => {
  it('replaces obsolete series and x-axis data when the trend query changes', () => {
    const wrapper = mount(DashboardTrendChart, {
      props: {
        labels: ['00:00', '01:00'],
        series: [{ label: 'gpt-5.6-sol', color: '#22a06b', values: [10, 20] }],
        loading: false
      },
      global: {
        stubs: {
          LoadingSpinner: true
        }
      }
    })

    expect(wrapper.getComponent({ name: 'VChart' }).props('updateOptions')).toEqual({
      notMerge: false,
      lazyUpdate: false,
      replaceMerge: ['series', 'xAxis']
    })
    expect(wrapper.getComponent({ name: 'VChart' }).props('option').tooltip.extraCssText).toContain(
      'overflow-wrap: anywhere'
    )
  })
})
