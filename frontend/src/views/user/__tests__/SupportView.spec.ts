import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { reactive, ref } from 'vue'
import SupportView from '../SupportView.vue'
import type { SupportTicketRecord } from '@/api/support'

const api = vi.hoisted(() => ({ getTicket: vi.fn() }))
const support = vi.hoisted(() => ({
  ticketId: null as string | null,
  error: '',
  refresh: vi.fn(),
}))
vi.mock('@/api/support', () => ({ supportAPI: api }))
vi.mock('@/stores/support', () => ({ useSupportStore: () => reactive(support) }))
vi.mock('vue-i18n', async (importOriginal) => ({
  ...await importOriginal<typeof import('vue-i18n')>(),
  useI18n: () => ({ t: (key: string) => key, locale: ref('zh-CN') }),
}))
enableAutoUnmount(afterEach)

const record: SupportTicketRecord = {
  id: 'ticket-123', title: 'A real issue', description: 'Original issue description',
  product: 'Codex', priority: 'normal', status: 'submitted',
  created_at: '2026-09-30T00:00:00Z', updated_at: '2026-09-30T00:00:00Z',
}
const mountView = () => mount(SupportView, {
  global: { stubs: {
    AppLayout: { template: '<main><slot /></main>' },
    BaseDialog: { props: ['show', 'title'], template: '<div v-if="show" role="dialog"><slot /><slot name="footer" /></div>' },
    SupportConversation: true,
    Icon: true,
    Select: true,
  } },
})

beforeEach(() => {
  vi.resetAllMocks()
  support.ticketId = null
  support.error = ''
  support.refresh.mockResolvedValue(undefined)
  api.getTicket.mockResolvedValue(record)
})

describe('SupportView API boundaries', () => {
  it('shows unavailable totals and does not claim an empty history without a list API', async () => {
    const wrapper = mountView()
    await flushPromises()
    const statistics = wrapper.get('[aria-label="support.statistics"]')
    expect(statistics.findAll('button')).toHaveLength(5)
    expect(statistics.findAll('button').every((button) => button.attributes('disabled') !== undefined)).toBe(true)
    expect(statistics.text().match(/--/g)).toHaveLength(5)
    expect(wrapper.get('#support-panel-tickets').text()).toContain('support.listNotConnected')
    expect(wrapper.text()).toContain('support.noLinkedTicket')
    expect(api.getTicket).not.toHaveBeenCalled()
  })

  it('shows the linked ticket separately from the history and preserves the unavailable notice', async () => {
    support.ticketId = record.id
    const wrapper = mountView()
    await flushPromises()
    expect(api.getTicket).toHaveBeenCalledWith(record.id)
    expect(wrapper.get('[data-testid="ticket-row"]').text()).toContain(record.title)
    expect(wrapper.get('[data-testid="ticket-row"]').text()).toContain('support.status.submitted')
    expect(wrapper.get('#support-panel-tickets').text()).not.toContain(record.id)
    expect(wrapper.text()).toContain('support.listUnavailable')
    expect(wrapper.text()).toContain('support.typeUnavailable')
    await wrapper.get('[data-testid="ticket-row"]').trigger('click')
    expect(wrapper.get('[role="dialog"]').text()).toContain('support.replyUnavailable')
  })

  it('shows a load error instead of an empty result and supports retry', async () => {
    support.ticketId = record.id
    api.getTicket.mockRejectedValueOnce({ status: 503 })
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.get('[role="alert"]').text()).toContain('support.errors.loadTicket')
    expect(wrapper.text()).not.toContain('support.noLinkedTicket')
    await wrapper.get('[role="alert"] button').trigger('click')
    await flushPromises()
    expect(wrapper.find('[role="alert"]').exists()).toBe(false)
    expect(wrapper.find('[data-testid="ticket-row"]').exists()).toBe(true)
  })

  it('does not invent types or fail rendering unknown metadata', async () => {
    support.ticketId = record.id
    api.getTicket.mockResolvedValueOnce({ ...record, status: 'new-status', priority: 'urgent', updated_at: 'invalid-date', product: 'technical' })
    const wrapper = mountView()
    await flushPromises()
    const row = wrapper.get('[data-testid="ticket-row"]')
    expect(row.text()).toContain('support.status.unknown')
    expect(row.text()).toContain('support.priorityUnknown')
    expect(row.text()).toContain('support.notProvided')
    expect(row.text()).toContain('support.typeUnavailable')
    expect(row.text()).not.toContain('support.types.technical')
  })

  it('marks direct creation as unavailable and offers the separate AI conversation', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-testid="create-ticket"]').trigger('click')
    const dialog = wrapper.get('[role="dialog"]')
    expect(dialog.text()).toContain('support.directCreateUnavailable')
    expect(dialog.find('input[type="file"]').exists()).toBe(false)
    expect(dialog.get('fieldset').attributes('disabled')).toBeDefined()
    await dialog.findAll('button').find((button) => button.text().includes('support.aiConsultation'))!.trigger('click')
    expect(wrapper.get('#support-tab-chat').attributes('aria-selected')).toBe('true')
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
  })

  it('discards late ticket responses when the linked ticket changes', async () => {
    support.ticketId = record.id
    const wrapper = mountView()
    await flushPromises()
    let finishOlderRequest!: (value: SupportTicketRecord) => void
    api.getTicket.mockImplementationOnce(() => new Promise<SupportTicketRecord>((resolve) => { finishOlderRequest = resolve }))
    const state = reactive(support)
    state.ticketId = 'ticket-older'
    await flushPromises()
    api.getTicket.mockResolvedValueOnce({ ...record, id: 'ticket-newer', title: 'Newer issue' })
    state.ticketId = 'ticket-newer'
    await flushPromises()
    finishOlderRequest({ ...record, id: 'ticket-older', title: 'Older issue' })
    await flushPromises()
    expect(wrapper.get('[data-testid="ticket-row"]').text()).toContain('Newer issue')
    expect(wrapper.get('[data-testid="ticket-row"]').text()).not.toContain('Older issue')
  })
})
