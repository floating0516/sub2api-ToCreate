import { apiClient } from './client'

const SUPPORT_REQUEST_TIMEOUT_MS = 70_000

export interface SupportCitation {
  source?: string
  section?: string
  version?: string
  document_id?: string
  chunk_id?: string
  score?: number
  excerpt?: string
  url?: string
  fetched_at?: string
  provider?: string
  scope?: string
  knowledge_release?: string
}

export interface SupportEvidenceDecision {
  generation?: { mode: 'local' | 'model' | 'fallback'; model?: string; reason?: string; usage?: { input_tokens?: number; output_tokens?: number } }
  evidence_action?: 'answer' | 'clarify' | 'refuse' | null
  evidence_reason_code?: string
  supported_source_ids?: string[]
  missing_information?: string[]
}

export interface SupportTicketDraft {
  title: string
  description: string
  priority: string
  product: string
}

export interface SupportChatResult extends SupportEvidenceDecision {
  thread_id: string
  answer: string
  citations: SupportCitation[]
  evidence_status?: string
  next_action?: string
  ticket_draft?: SupportTicketDraft | null
  ticket_validation?: { ok: boolean; errors: string[] } | null
  ticket_id?: string | null
  status?: string
  replayed?: boolean
}

export interface SupportThread {
  thread_id: string
  turns: SupportThreadTurn[]
  ticket_draft?: SupportTicketDraft | null
  ticket_id?: string | null
  status?: string
  updated_at?: string
}

export interface SupportThreadTurn extends SupportEvidenceDecision {
  role?: string
  content?: string
  citations?: SupportCitation[]
  evidence_status?: string
  created_at?: string
  client_message_id?: string
  reply_to?: string
}

export interface SupportTicketRecord {
  id: string
  title: string
  description: string
  priority: string
  status: string
  product: string
  created_at: string
  updated_at: string
  type?: string
  order_id?: string | null
  requester_id?: string
  last_reply_at?: string | null
  attachments?: SupportAttachment[]
  replies?: SupportTicketReply[]
  allowed_actions?: string[]
}

export interface SupportAttachment {
  id: string
  filename: string
  size: number
  content_type: string
  url?: string
}

export interface SupportTicketReply {
  id: string
  actor: string
  content: string
  created_at: string
  attachments: SupportAttachment[]
}

export interface SupportTicketList {
  items: SupportTicketRecord[]
  total: number
  page: number
  page_size: number
  pages: number
}

export interface SupportTicketStats {
  all: number
  pending_agent: number
  pending_user: number
  resolved: number
  closed: number
}

export interface SupportCreateTicket {
  type: string
  title: string
  description: string
  priority: string
  product: string
  order_id?: string
  attachment_ids: string[]
  client_request_id: string
}

export const supportAPI = {
  chat(payload: { message: string; client_message_id: string; thread_id?: string }) {
    return apiClient.post<SupportChatResult>('/support/chat', payload, { timeout: SUPPORT_REQUEST_TIMEOUT_MS }).then((r) => r.data)
  },
  confirmTicket(payload: { thread_id: string; confirm: boolean }) {
    return apiClient.post<{ status: string; ticket_id?: string | null; reused?: boolean }>('/support/tickets/confirm', payload, { timeout: SUPPORT_REQUEST_TIMEOUT_MS }).then((r) => r.data)
  },
  getThread(threadId: string) {
    return apiClient.get<SupportThread>(`/support/threads/${encodeURIComponent(threadId)}`, { timeout: SUPPORT_REQUEST_TIMEOUT_MS }).then((r) => r.data)
  },
  getTicket(ticketId: string) {
    return apiClient.get<SupportTicketRecord>(`/support/tickets/${encodeURIComponent(ticketId)}`, { timeout: SUPPORT_REQUEST_TIMEOUT_MS }).then((r) => r.data)
  },
  listTickets(params: { q?: string; type?: string; status?: string; page: number; page_size: number }) {
    return apiClient.get<SupportTicketList>('/support/tickets', { params }).then((r) => r.data)
  },
  ticketStats() {
    return apiClient.get<SupportTicketStats>('/support/tickets/stats').then((r) => r.data)
  },
  createTicket(payload: SupportCreateTicket) {
    return apiClient.post<SupportTicketRecord>('/support/tickets', payload, { timeout: SUPPORT_REQUEST_TIMEOUT_MS }).then((r) => r.data)
  },
  replyTicket(ticketId: string, payload: { content: string; attachment_ids: string[]; client_message_id: string }) {
    return apiClient.post<SupportTicketReply>(`/support/tickets/${encodeURIComponent(ticketId)}/replies`, payload, { timeout: SUPPORT_REQUEST_TIMEOUT_MS }).then((r) => r.data)
  },
  transitionTicket(ticketId: string, action: 'resolve' | 'close' | 'reopen') {
    return apiClient.post<SupportTicketRecord>(`/support/tickets/${encodeURIComponent(ticketId)}/${action}`, {}).then((r) => r.data)
  },
  uploadAttachment(file: File, ticketId?: string) {
    const form = new FormData()
    form.append('file', file)
    if (ticketId) form.append('ticket_id', ticketId)
    return apiClient.post<SupportAttachment>('/support/uploads', form, {
      headers: { 'Content-Type': undefined }, timeout: SUPPORT_REQUEST_TIMEOUT_MS,
    }).then((r) => r.data)
  },
  downloadAttachment(id: string) {
    return apiClient.get<Blob>(`/support/uploads/${encodeURIComponent(id)}`, { responseType: 'blob' }).then((r) => r.data)
  },
}
