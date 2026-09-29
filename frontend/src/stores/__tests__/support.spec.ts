import { beforeEach, describe, expect, it, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useSupportStore } from '@/stores/support'

const authStore = vi.hoisted(() => ({
  user: { id: 42 },
}))

const supportMocks = vi.hoisted(() => ({
  chat: vi.fn(),
  confirmTicket: vi.fn(),
  getThread: vi.fn(),
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authStore,
}))

vi.mock('@/api/support', () => ({
  supportAPI: supportMocks,
}))

const draft = {
  title: 'Codex returns 401',
  description: 'The request fails with 401.',
  priority: 'normal',
  product: 'Codex',
}

describe('useSupportStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    localStorage.clear()
    vi.clearAllMocks()
  })

  it('restores an unconfirmed ticket draft after refresh', async () => {
    localStorage.setItem('support.thread.42', '7d5584bd-22d4-45af-9971-d1019f36a45d')
    supportMocks.getThread.mockResolvedValue({
      thread_id: '7d5584bd-22d4-45af-9971-d1019f36a45d',
      turns: [],
      ticket_draft: draft,
      ticket_id: null,
      status: 'ticket_drafted',
    })

    const store = useSupportStore()
    await store.refresh()

    expect(store.threadId).toBe('7d5584bd-22d4-45af-9971-d1019f36a45d')
    expect(store.draft).toEqual(draft)
    expect(store.hasDraft).toBe(true)
  })

  it('removes a stale stored thread after a confirmed 404', async () => {
    localStorage.setItem('support.thread.42', '7d5584bd-22d4-45af-9971-d1019f36a45d')
    supportMocks.getThread.mockRejectedValue({ status: 404 })

    const store = useSupportStore()
    await store.refresh()

    expect(store.threadId).toBeNull()
    expect(localStorage.getItem('support.thread.42')).toBeNull()
  })

  it('reuses the message and thread IDs when retrying after a network error', async () => {
    supportMocks.chat
      .mockRejectedValueOnce({ status: 0 })
      .mockResolvedValueOnce({
        thread_id: 'server-thread',
        answer: 'Recovered answer',
        citations: [],
        status: 'answered',
      })
    supportMocks.getThread.mockRejectedValue({ status: 404 })

    const store = useSupportStore()
    await store.send('Codex returns 401')
    const firstPayload = supportMocks.chat.mock.calls[0][0]

    await store.send('Codex returns 401')
    const retryPayload = supportMocks.chat.mock.calls[1][0]

    expect(supportMocks.getThread).toHaveBeenCalledWith(firstPayload.thread_id)
    expect(retryPayload.client_message_id).toBe(firstPayload.client_message_id)
    expect(retryPayload.thread_id).toBe(firstPayload.thread_id)
    expect(store.pendingMessage).toBeNull()
  })

  it('reconciles a timed-out request from the server before offering a retry', async () => {
    supportMocks.chat.mockImplementation(async (payload: { client_message_id: string; thread_id: string }) => {
      supportMocks.getThread.mockResolvedValue({
        thread_id: payload.thread_id,
        turns: [
          {
            role: 'user',
            content: 'Codex returns 401',
            client_message_id: payload.client_message_id,
          },
          {
            role: 'assistant',
            content: 'Check the configured token.',
            reply_to: payload.client_message_id,
          },
        ],
        status: 'answered',
      })
      throw { status: 504 }
    })

    const store = useSupportStore()
    await store.send('Codex returns 401')

    expect(supportMocks.getThread).toHaveBeenCalledOnce()
    expect(store.turns).toHaveLength(2)
    expect(store.turns[1].content).toBe('Check the configured token.')
    expect(store.pendingMessage).toBeNull()
    expect(store.error).toBe('')
  })

  it.each([
    [true, 'ticket_submitted', 'ticket-123'] as const,
    [false, 'ticket_cancelled', null] as const,
  ])('clears the draft after confirm=%s', async (confirm, status, ticketId) => {
    supportMocks.confirmTicket.mockResolvedValue({
      status,
      ticket_id: ticketId,
    })
    const store = useSupportStore()
    store.threadId = '7d5584bd-22d4-45af-9971-d1019f36a45d'
    store.draft = draft
    store.status = 'ticket_drafted'

    await store.confirmTicket(confirm)

    expect(supportMocks.confirmTicket).toHaveBeenCalledWith({
      thread_id: '7d5584bd-22d4-45af-9971-d1019f36a45d',
      confirm,
    })
    expect(store.draft).toBeNull()
    expect(store.status).toBe(status)
    expect(store.ticketId).toBe(ticketId)
  })
})
