import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { reactive, ref } from 'vue'
import SupportView from '../SupportView.vue'
import SupportConversation from '@/components/support/SupportConversation.vue'
import type { SupportTicketRecord } from '@/api/support'

const api = vi.hoisted(() => ({ getTicket: vi.fn(), listTickets: vi.fn(), ticketStats: vi.fn(), createTicket: vi.fn(), replyTicket: vi.fn(), transitionTicket: vi.fn(), uploadAttachment: vi.fn(), downloadAttachment: vi.fn() }))
const auth = vi.hoisted(() => ({ user: { id: 1 } }))
const app = vi.hoisted(() => ({ showSuccess: vi.fn() }))
const orders = vi.hoisted(() => ({ listManagedRechargeOrders: vi.fn() }))
const support = vi.hoisted(() => ({
  ticketId: null as string | null,
  error: '',
  refresh: vi.fn(),
}))
vi.mock('@/api/support', () => ({ supportAPI: api }))
vi.mock('@/api/managedRecharge', () => orders)
vi.mock('@/stores/auth', () => ({ useAuthStore: () => reactive(auth) }))
vi.mock('@/stores/app', () => ({ useAppStore: () => app }))
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
  auth.user = { id: 1 }
  support.error = ''
  support.refresh.mockResolvedValue(undefined)
  api.getTicket.mockResolvedValue(record)
  api.listTickets.mockRejectedValue({ status: 404 })
  orders.listManagedRechargeOrders.mockResolvedValue([])
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
    await flushPromises()
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
    expect(dialog.get('input[type="file"]').attributes('disabled')).toBeDefined()
    expect(dialog.get('fieldset').attributes('disabled')).toBeUndefined()
    expect(dialog.get('[data-testid="submit-ticket"]').attributes('disabled')).toBeDefined()
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

  it('uses real full totals, renders history and filters by a status card', async () => {
    api.listTickets.mockResolvedValue({ items: [{ ...record, type: 'technical' }], total: 53, page: 1, page_size: 20, pages: 3 })
    api.ticketStats.mockResolvedValue({ all: 53, pending_agent: 25, pending_user: 20, resolved: 7, closed: 1 })
    const wrapper = mountView()
    await flushPromises()
    const statistics = wrapper.get('[aria-label="support.statistics"]')
    expect(statistics.text()).toContain('53')
    expect(wrapper.get('#support-panel-tickets').text()).toContain(record.title)
    expect(wrapper.text()).not.toContain('support.listUnavailable')
    expect(wrapper.find('[aria-label="support.currentTicket"]').exists()).toBe(false)
    await statistics.findAll('button')[1]!.trigger('click')
    await flushPromises()
    expect(api.listTickets).toHaveBeenLastCalledWith(expect.objectContaining({ status: 'pending_agent', page: 1 }))
    expect(statistics.findAll('button')[1]!.attributes('aria-pressed')).toBe('true')
  })

  it('does not turn a failed history request into an empty list', async () => {
    api.listTickets.mockRejectedValue({ status: 503 })
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.get('#support-panel-tickets [role="alert"]').text()).toContain('support.errors.loadList')
    expect(wrapper.get('#support-panel-tickets').text()).not.toContain('support.emptyTickets')
  })

  it('keeps failed statistics unavailable instead of showing zero totals', async () => {
    api.listTickets.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })
    api.ticketStats.mockRejectedValue({ status: 503 })
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.get('[aria-label="support.statistics"]').text().match(/--/g)).toHaveLength(5)
    expect(wrapper.text()).toContain('support.errors.loadStats')
    expect(wrapper.get('#support-panel-tickets').text()).toContain('support.emptyTickets')
  })

  it('clears an unsent form when the account changes', async () => {
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-testid="create-ticket"]').trigger('click')
    await wrapper.get('#support-create-title').setValue('Previous account draft')
    reactive(auth).user = { id: 2 }
    await flushPromises()
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
    await wrapper.get('[data-testid="create-ticket"]').trigger('click')
    expect((wrapper.get('#support-create-title').element as HTMLInputElement).value).toBe('')
  })

  it('uses allowed actions and disables resolved replies in the real detail', async () => {
    const resolved = { ...record, status: 'resolved', type: 'technical', allowed_actions: ['close', 'reopen'], attachments: [], replies: [{ id: 'reply-1', actor: 'agent', content: 'Support response', created_at: record.updated_at, attachments: [] }] }
    api.listTickets.mockResolvedValue({ items: [resolved], total: 1, page: 1, page_size: 20, pages: 1 })
    api.ticketStats.mockResolvedValue({ all: 1, pending_agent: 0, pending_user: 0, resolved: 1, closed: 0 })
    api.getTicket.mockResolvedValue(resolved)
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-testid="ticket-row"]').trigger('click')
    await flushPromises()
    const dialog = wrapper.get('[role="dialog"]')
    expect(dialog.text()).toContain('Support response')
    expect(dialog.text()).toContain('support.messageRoles.staff')
    expect(dialog.get('[data-testid="send-reply"]').attributes('disabled')).toBeDefined()
    expect(dialog.text()).toContain('support.actions.reopen')
    expect(dialog.text()).not.toContain('support.actions.resolve')
  })

  it('keeps AI consultation usable while a portal read is slow', async () => {
    let finish!: (value: unknown) => void
    api.listTickets.mockImplementationOnce(() => new Promise((resolve) => { finish = resolve }))
    const wrapper = mountView()
    await flushPromises()
    expect(wrapper.getComponent(SupportConversation).props('restoring')).toBe(false)
    expect(wrapper.get('#support-panel-tickets [aria-busy]').attributes('aria-busy')).toBe('true')
    finish({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })
    await flushPromises()
  })

  it('submits a real form and refreshes the history only after creation succeeds', async () => {
    api.listTickets.mockResolvedValueOnce({ items: [], total: 0, page: 1, page_size: 20, pages: 0 }).mockResolvedValue({ items: [record], total: 1, page: 1, page_size: 20, pages: 1 })
    api.ticketStats.mockResolvedValue({ all: 1, pending_agent: 1, pending_user: 0, resolved: 0, closed: 0 })
    api.createTicket.mockResolvedValue(record)
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-testid="create-ticket"]').trigger('click')
    await flushPromises()
    expect(orders.listManagedRechargeOrders).toHaveBeenCalledWith(100)
    await wrapper.get('#support-create-title').setValue('Test issue')
    await wrapper.get('#support-create-description').setValue('Detailed reproduction steps')
    await wrapper.get('#support-create-form').trigger('submit')
    await flushPromises()
    expect(api.createTicket).toHaveBeenCalledWith({ type: 'pre-sale', title: 'Test issue', description: 'Detailed reproduction steps', priority: 'normal', product: 'tocreate', order_id: undefined, attachment_ids: [], client_request_id: expect.any(String) })
    expect(app.showSuccess).toHaveBeenCalledWith('support.created')
    expect(wrapper.find('[role="dialog"]').exists()).toBe(false)
    expect(wrapper.get('#support-panel-tickets').text()).toContain(record.title)
  })

  it('does not announce success or close the form on an incomplete creation response', async () => {
    api.listTickets.mockResolvedValue({ items: [], total: 0, page: 1, page_size: 20, pages: 0 })
    api.ticketStats.mockResolvedValue({ all: 0, pending_agent: 0, pending_user: 0, resolved: 0, closed: 0 })
    api.createTicket.mockResolvedValue({})
    const wrapper = mountView()
    await flushPromises()
    await wrapper.get('[data-testid="create-ticket"]').trigger('click')
    await wrapper.get('#support-create-title').setValue('Test issue')
    await wrapper.get('#support-create-description').setValue('Detailed reproduction steps')
    await wrapper.get('#support-create-form').trigger('submit')
    await flushPromises()
    expect(app.showSuccess).not.toHaveBeenCalled()
    expect(wrapper.get('[role="dialog"]').text()).toContain('support.errors.createTicket')
    expect((wrapper.get('#support-create-title').element as HTMLInputElement).value).toBe('Test issue')
  })
})
