import { beforeEach, describe, expect, it, vi } from 'vitest'
import { supportAPI } from '../support'
import { adminSupportAPI } from '../admin/support'

const client = vi.hoisted(() => ({ get: vi.fn(), post: vi.fn() }))
vi.mock('../client', () => ({ apiClient: client }))
beforeEach(() => { vi.resetAllMocks(); client.get.mockResolvedValue({ data: {} }); client.post.mockResolvedValue({ data: {} }) })

describe('Support portal API contract', () => {
  it('uses same-origin list and full-statistics paths', async () => {
    await supportAPI.listTickets({ q: '401', type: 'technical', status: 'pending_agent', page: 2, page_size: 20 })
    expect(client.get).toHaveBeenCalledWith('/support/tickets', { params: { q: '401', type: 'technical', status: 'pending_agent', page: 2, page_size: 20 } })
    await supportAPI.ticketStats()
    expect(client.get).toHaveBeenLastCalledWith('/support/tickets/stats')
  })

  it('sends the backend idempotency fields for creation and replies', async () => {
    const payload = { type: 'technical', title: 'Test issue', description: 'Reproduction details', priority: 'normal', product: 'tocreate', attachment_ids: [], client_request_id: 'request-key' }
    await supportAPI.createTicket(payload)
    expect(client.post).toHaveBeenCalledWith('/support/tickets', payload, expect.objectContaining({ timeout: 70000 }))
    await supportAPI.replyTicket('ticket-1', { content: 'A reply', attachment_ids: ['file-1'], client_message_id: 'message-key' })
    expect(client.post).toHaveBeenLastCalledWith('/support/tickets/ticket-1/replies', { content: 'A reply', attachment_ids: ['file-1'], client_message_id: 'message-key' }, expect.objectContaining({ timeout: 70000 }))
  })

  it.each(['resolve', 'close', 'reopen'] as const)('uses the explicit %s transition path', async (action) => {
    await supportAPI.transitionTicket('ticket-1', action)
    expect(client.post).toHaveBeenCalledWith(`/support/tickets/ticket-1/${action}`, {})
  })

  it('uploads multipart through the shared client and downloads authenticated blobs', async () => {
    const file = new File(['hello'], 'example.txt', { type: 'text/plain' })
    await supportAPI.uploadAttachment(file, 'ticket-1')
    const [, form, options] = client.post.mock.calls[0]!
    expect(form).toBeInstanceOf(FormData)
    expect(form.get('file')).toBe(file)
    expect(form.get('ticket_id')).toBe('ticket-1')
    expect(options.headers['Content-Type']).toBeUndefined()
    await supportAPI.downloadAttachment('file-1')
    expect(client.get).toHaveBeenLastCalledWith('/support/uploads/file-1', { responseType: 'blob' })
  })

  it('keeps administrator ticket traffic on the admin same-origin namespace', async () => {
    await adminSupportAPI.listTickets({ page: 1, page_size: 20 })
    expect(client.get).toHaveBeenLastCalledWith('/admin/support/tickets', { params: { page: 1, page_size: 20 } })
    await adminSupportAPI.ticketStats()
    expect(client.get).toHaveBeenLastCalledWith('/admin/support/tickets/stats')
    await adminSupportAPI.replyTicket('ticket-1', { content: 'Reply', attachment_ids: [], client_message_id: 'admin-message' })
    expect(client.post).toHaveBeenLastCalledWith('/admin/support/tickets/ticket-1/replies', { content: 'Reply', attachment_ids: [], client_message_id: 'admin-message' }, expect.objectContaining({ timeout: 70000 }))
  })
})
