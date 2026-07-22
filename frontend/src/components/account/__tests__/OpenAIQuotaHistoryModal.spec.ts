import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'
import OpenAIQuotaHistoryModal from '../OpenAIQuotaHistoryModal.vue'
import { getOpenAIQuotaHistory, setOpenAIQuotaResetSource } from '@/api/admin/accounts'
import type { Account } from '@/types'

vi.mock('@/api/admin/accounts', () => ({
  getOpenAIQuotaHistory: vi.fn(),
  setOpenAIQuotaResetSource: vi.fn()
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

const account: Account = {
  id: 44,
  name: 'lihe2568@gmail.com',
  platform: 'openai',
  type: 'oauth',
  proxy_id: null,
  concurrency: 3,
  priority: 50,
  status: 'active',
  error_message: null,
  last_used_at: null,
  expires_at: null,
  auto_pause_on_expired: false,
  created_at: '2026-06-25T05:38:07Z',
  updated_at: '2026-07-18T03:43:25Z',
  schedulable: true,
  rate_limited_at: null,
  rate_limit_reset_at: null,
  overload_until: null,
  temp_unschedulable_until: null,
  temp_unschedulable_reason: null,
  session_window_start: null,
  session_window_end: null,
  session_window_status: null
}

const mountModal = () => mount(OpenAIQuotaHistoryModal, {
  props: { show: true, account },
  global: {
    stubs: {
      Teleport: true,
      Transition: false,
      OpenAIQuotaUsageChart: true
    }
  }
})

describe('OpenAIQuotaHistoryModal', () => {
  beforeEach(() => {
    vi.mocked(getOpenAIQuotaHistory).mockReset()
    vi.mocked(setOpenAIQuotaResetSource).mockReset()
    vi.mocked(setOpenAIQuotaResetSource).mockResolvedValue(undefined)
  })

  it('shows the current cycle and closed reset cycles', async () => {
    vi.mocked(getOpenAIQuotaHistory).mockResolvedValue({
      current: {
        id: 3,
        cycle_started_at: '2026-07-19T03:21:00Z',
        last_observed_at: '2026-07-19T03:43:25Z',
        last_used_percent: 1,
        peak_used_percent: 1,
        provider_reset_at: '2026-07-26T03:20:48Z'
      },
      history: [
        {
          id: 2,
          cycle_started_at: '2026-07-18T03:25:00Z',
          last_observed_at: '2026-07-19T03:20:00Z',
          last_used_percent: 12,
          peak_used_percent: 32,
          reset_observed_at: '2026-07-19T03:21:00Z',
          reset_to_percent: 0,
          detection_reason: 'manual_reset',
          automatic_reset_source: 'manual',
          reset_source: 'manual'
        },
        {
          id: 1,
          cycle_started_at: '2026-07-17T07:06:46Z',
          last_observed_at: '2026-07-17T07:06:46Z',
          last_used_percent: 28,
          peak_used_percent: 28,
          reset_observed_at: '2026-07-18T03:25:00Z',
          reset_to_percent: 0,
          detection_reason: 'usage_drop',
          automatic_reset_source: 'unknown',
          reset_source: 'provider',
          reset_source_override: 'provider'
        }
      ],
      samples: [
        {
          cycle_id: 1,
          observed_at: '2026-07-17T07:06:46Z',
          used_percent: 28,
          local_cost_usd: 1.2
        },
        {
          cycle_id: 3,
          observed_at: '2026-07-19T03:21:00Z',
          used_percent: 0,
          local_cost_usd: 0
        }
      ],
      has_more: false
    })

    const wrapper = mountModal()
    await flushPromises()

    expect(getOpenAIQuotaHistory).toHaveBeenCalledWith(44)
    expect(wrapper.text()).toContain('lihe2568@gmail.com')
    expect(wrapper.text()).toContain('1%')
    expect(wrapper.text()).toContain('28%')
    expect(wrapper.text()).toContain('0%')
    expect(wrapper.text()).not.toContain('usage_drop')
    expect(wrapper.getComponent({ name: 'OpenAIQuotaUsageChart' }).props('resetMarkers')).toEqual([
      { cycleId: 2, observedAt: '2026-07-19T03:21:00Z', source: 'manual' },
      { cycleId: 1, observedAt: '2026-07-18T03:25:00Z', source: 'provider' }
    ])

    wrapper.getComponent({ name: 'OpenAIQuotaUsageChart' }).vm.$emit('reset-marker-click', {
      cycleId: 2,
      observedAt: '2026-07-19T03:21:00Z',
      source: 'manual'
    })
    await wrapper.vm.$nextTick()
    expect(wrapper.find('[data-testid="reset-source-editor"]').exists()).toBe(true)
    const sourceOptions = wrapper.findAll('[role="radio"]')
    await sourceOptions[2].trigger('click')
    await wrapper.get('[data-testid="save-reset-source"]').trigger('click')
    await flushPromises()
    expect(setOpenAIQuotaResetSource).toHaveBeenCalledWith(44, 2, 'provider')
    expect(getOpenAIQuotaHistory).toHaveBeenCalledTimes(2)
    wrapper.unmount()
  })

  it('shows stable empty states before the first reset', async () => {
    vi.mocked(getOpenAIQuotaHistory).mockResolvedValue({
      history: [],
      samples: [],
      has_more: false
    })

    const wrapper = mountModal()
    await flushPromises()

    expect(wrapper.text()).toContain('admin.accounts.openaiQuotaHistory.noCurrent')
    expect(wrapper.text()).toContain('admin.accounts.openaiQuotaHistory.noHistory')
    wrapper.unmount()
  })
})
