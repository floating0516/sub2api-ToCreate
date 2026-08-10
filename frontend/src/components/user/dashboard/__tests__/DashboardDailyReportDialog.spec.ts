import { flushPromises, mount } from '@vue/test-utils'
import { defineComponent } from 'vue'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import DashboardDailyReportDialog from '../DashboardDailyReportDialog.vue'

const getDailyReport = vi.hoisted(() => vi.fn())

vi.mock('@/api/usage', () => ({
  usageAPI: { getDailyReport }
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      locale: { value: 'zh-CN' },
      t: (key: string) => key
    })
  }
})

const BaseDialogStub = defineComponent({
  props: ['show', 'title'],
  emits: ['close'],
  template: '<div v-if="show"><slot /></div>'
})

const report = {
  date: '2026-08-04',
  timezone: 'Asia/Shanghai',
  generated_at: '2026-08-05T00:00:00Z',
  summary: {
    requests: 200,
    input_tokens: 1_000_000,
    output_tokens: 500_000,
    cache_creation_tokens: 0,
    cache_read_tokens: 900_000,
    total_tokens: 1_500_000,
    model_count: 3,
    average_tokens_per_request: 7_500,
    average_duration_ms: 800,
    cache_hit_rate: 60
  },
  comparison: {
    previous_requests: 180,
    previous_total_tokens: 1_200_000,
    request_change_pct: 11.1,
    token_change_pct: 25
  },
  models: [
    { model: 'gpt-primary', requests: 120, input_tokens: 800_000, output_tokens: 400_000, cache_creation_tokens: 0, cache_read_tokens: 700_000, total_tokens: 1_200_000, share: 80 },
    { model: 'claude-secondary', requests: 60, input_tokens: 150_000, output_tokens: 100_000, cache_creation_tokens: 0, cache_read_tokens: 150_000, total_tokens: 250_000, share: 16.7 },
    { model: 'gemini-light', requests: 20, input_tokens: 50_000, output_tokens: 0, cache_creation_tokens: 0, cache_read_tokens: 50_000, total_tokens: 50_000, share: 3.3 }
  ],
  narrative: '今天的模型阵容很热闹。',
  ai_generated: true,
  generator_model: 'internal-report-writer'
}

describe('DashboardDailyReportDialog', () => {
  beforeEach(() => {
    getDailyReport.mockReset()
    getDailyReport.mockResolvedValue(report)
  })

  it('renders ranked models as compact items with calls and Token usage', async () => {
    const wrapper = mount(DashboardDailyReportDialog, {
      props: {
        show: true,
        date: '2026-08-04',
        timezone: 'Asia/Shanghai'
      },
      global: {
        stubs: {
          BaseDialog: BaseDialogStub,
          Icon: true,
          LoadingSpinner: true
        }
      }
    })

    await flushPromises()

    const models = wrapper.findAll('[data-test="daily-report-model"]')
    expect(models).toHaveLength(3)
    expect(models.map((item) => item.get('strong').text())).toEqual([
      'gpt-primary',
      'claude-secondary',
      'gemini-light'
    ])
    expect(models[0].get('[data-test="model-requests"]').text()).toContain('120')
    expect(models[0].get('[data-test="model-tokens"]').text()).toContain('1.2M')
    expect(models[0].text()).toContain('80.0%')
    expect(wrapper.text()).not.toContain('internal-report-writer')
    expect(wrapper.text()).not.toContain('dashboard.dailyReport.templateGenerated')
  })
})
