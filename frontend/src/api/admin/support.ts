import { apiClient } from '../client'
import type { SupportAttachment, SupportTicketList, SupportTicketRecord, SupportTicketReply, SupportTicketStats } from '../support'

const timeout = 70_000
const base = '/admin/support'

export const adminSupportAPI = {
  listTickets(params: { q?: string; type?: string; status?: string; page: number; page_size: number }) {
    return apiClient.get<SupportTicketList>(`${base}/tickets`, { params }).then((r) => r.data)
  },
  ticketStats() {
    return apiClient.get<SupportTicketStats>(`${base}/tickets/stats`).then((r) => r.data)
  },
  getTicket(id: string) {
    return apiClient.get<SupportTicketRecord>(`${base}/tickets/${encodeURIComponent(id)}`, { timeout }).then((r) => r.data)
  },
  replyTicket(id: string, payload: { content: string; attachment_ids: string[]; client_message_id: string }) {
    return apiClient.post<SupportTicketReply>(`${base}/tickets/${encodeURIComponent(id)}/replies`, payload, { timeout }).then((r) => r.data)
  },
  transitionTicket(id: string, action: 'resolve' | 'close' | 'reopen') {
    return apiClient.post<SupportTicketRecord>(`${base}/tickets/${encodeURIComponent(id)}/${action}`, {}).then((r) => r.data)
  },
  uploadAttachment(file: File, ticketId?: string) {
    const form = new FormData()
    form.append('file', file)
    if (ticketId) form.append('ticket_id', ticketId)
    return apiClient.post<SupportAttachment>(`${base}/uploads`, form, {
      headers: { 'Content-Type': undefined }, timeout,
    }).then((r) => r.data)
  },
  downloadAttachment(id: string) {
    return apiClient.get<Blob>(`${base}/uploads/${encodeURIComponent(id)}`, { responseType: 'blob' }).then((r) => r.data)
  },
}
