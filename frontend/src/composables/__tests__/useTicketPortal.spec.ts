import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { enableAutoUnmount, flushPromises, mount } from '@vue/test-utils'
import { defineComponent, reactive } from 'vue'
import { useTicketPortal } from '../useTicketPortal'

const api = vi.hoisted(() => ({ listTickets: vi.fn(), ticketStats: vi.fn(), getTicket: vi.fn(), createTicket: vi.fn(), replyTicket: vi.fn(), transitionTicket: vi.fn() }))
const auth = vi.hoisted(() => ({ user: { id: 1 } }))
vi.mock('@/api/support', () => ({ supportAPI: api }))
vi.mock('@/stores/auth', () => ({ useAuthStore: () => reactive(auth) }))
enableAutoUnmount(afterEach)
afterEach(() => vi.useRealTimers())
const record = { id: 'ticket-1', title: 'Test issue', description: 'Details about the issue', product: 'tocreate', priority: 'normal', status: 'pending_agent', created_at: '2026-09-30T00:00:00Z', updated_at: '2026-09-30T00:00:00Z', allowed_actions: ['reply', 'resolve', 'close'], replies: [], attachments: [] }
const pageResult = { items: [record], total: 73, page: 1, page_size: 20, pages: 4 }
const totals = { all: 73, pending_agent: 30, pending_user: 20, resolved: 13, closed: 10 }
function harness() {
  let portal!: ReturnType<typeof useTicketPortal>
  mount(defineComponent({ setup() { portal = useTicketPortal(); return {} }, template: '<div />' }))
  return portal
}
function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((done) => { resolve = done })
  return { promise, resolve }
}
beforeEach(() => {
  vi.resetAllMocks()
  auth.user = { id: 1 }
  api.listTickets.mockResolvedValue(pageResult)
  api.ticketStats.mockResolvedValue(totals)
  api.getTicket.mockResolvedValue(record)
})

describe('Ticket portal data boundaries', () => {
  it('uses the full statistics endpoint, not the number of displayed items', async () => {
    const portal = harness()
    await portal.refresh()
    expect(portal.items.value).toHaveLength(1)
    expect(portal.total.value).toBe(73)
    expect(portal.stats.value).toEqual(totals)
    expect(portal.listState.value).toBe('ready')
  })

  it('distinguishes an absent portal API from service errors', async () => {
    const portal = harness()
    api.listTickets.mockRejectedValueOnce({ status: 404 })
    await portal.refresh()
    expect(portal.listState.value).toBe('unavailable')
    expect(portal.available.value).toBe(false)
    expect(portal.stats.value).toBeNull()
    api.listTickets.mockRejectedValueOnce({ status: 503 })
    await portal.refresh()
    expect(portal.listState.value).toBe('error')
  })

  it('does not interpret malformed successful responses as zero tickets', async () => {
    const portal = harness()
    api.listTickets.mockResolvedValueOnce({ total: 0 })
    await portal.refresh()
    expect(portal.listState.value).toBe('error')
    expect(portal.available.value).toBe(false)
  })

  it('clears statistics on failure and rejects incomplete or negative counts', async () => {
    const portal = harness()
    await portal.refresh()
    api.ticketStats.mockRejectedValueOnce({ status: 503 })
    await portal.loadStats()
    expect(portal.stats.value).toBeNull()
    expect(portal.statsError.value).toBe(true)
    api.ticketStats.mockResolvedValueOnce({ ...totals, closed: -1 })
    await portal.loadStats()
    expect(portal.stats.value).toBeNull()
  })

  it('resets pagination on status and type changes without sending user_id', async () => {
    const portal = harness()
    portal.page.value = 3
    await flushPromises()
    portal.type.value = 'technical'
    portal.status.value = 'pending_user'
    await flushPromises()
    expect(portal.page.value).toBe(1)
    expect(api.listTickets).toHaveBeenLastCalledWith({ q: undefined, type: 'technical', status: 'pending_user', page: 1, page_size: 20 })
  })

  it('debounces search and discards responses from the earlier query', async () => {
    vi.useFakeTimers({ toFake: ['setTimeout', 'clearTimeout'] })
    const portal = harness()
    const older = deferred<typeof pageResult>()
    api.listTickets.mockReturnValueOnce(older.promise)
    const request = portal.loadList()
    portal.search.value = 'new query'
    await flushPromises()
    older.resolve(pageResult)
    await request
    expect(portal.items.value).toEqual([])
    await vi.advanceTimersByTimeAsync(300)
    expect(api.listTickets).toHaveBeenLastCalledWith(expect.objectContaining({ q: 'new query', page: 1 }))
  })

  it('clears account-owned data and rejects late list, stats, and detail responses', async () => {
    const portal = harness()
    const list = deferred<typeof pageResult>()
    const stats = deferred<typeof totals>()
    const detail = deferred<typeof record>()
    api.listTickets.mockReturnValueOnce(list.promise)
    api.ticketStats.mockReturnValueOnce(stats.promise)
    api.getTicket.mockReturnValueOnce(detail.promise)
    const requests = [portal.loadList(), portal.loadStats(), portal.open(record.id)]
    reactive(auth).user = { id: 2 }
    list.resolve(pageResult)
    stats.resolve(totals)
    detail.resolve(record)
    await Promise.all(requests)
    expect(portal.items.value).toEqual([])
    expect(portal.stats.value).toBeNull()
    expect(portal.detail.value).toBeNull()
    expect(portal.available.value).toBe(false)
  })

  it('does not let a late response overwrite a newer selected detail', async () => {
    const portal = harness()
    const older = deferred<typeof record>()
    api.getTicket.mockReturnValueOnce(older.promise)
    const request = portal.open(record.id)
    api.getTicket.mockResolvedValueOnce({ ...record, id: 'ticket-new' })
    await portal.open('ticket-new')
    older.resolve(record)
    await request
    expect(portal.detail.value?.id).toBe('ticket-new')
  })

  it.each(['resolved', 'closed'])('blocks replies for %s even with an erroneous allowed_actions response', async (status) => {
    const portal = harness()
    api.getTicket.mockResolvedValueOnce({ ...record, status })
    await portal.open(record.id)
    expect(await portal.reply('A reply', [], 'message-key')).toBe(false)
    expect(api.replyTicket).not.toHaveBeenCalled()
  })

  it('never marks a failed transition successful or changes the cached status', async () => {
    const portal = harness()
    await portal.open(record.id)
    api.transitionTicket.mockRejectedValueOnce({ status: 409 })
    expect(await portal.transition('close')).toBe(false)
    expect(portal.actionError.value).toBe(true)
    expect(portal.detail.value?.status).toBe('pending_agent')
  })

  it('only performs state transitions advertised by the backend', async () => {
    const portal = harness()
    await portal.open(record.id)
    expect(await portal.transition('reopen')).toBe(false)
    expect(api.transitionTicket).not.toHaveBeenCalled()
    api.transitionTicket.mockResolvedValueOnce({ ...record, status: 'closed', allowed_actions: ['reopen'] })
    expect(await portal.transition('close')).toBe(true)
    expect(portal.detail.value?.allowed_actions).toEqual(['reopen'])
  })
})
