import { apiClient } from './client'

const SUPPORT_REQUEST_TIMEOUT_MS = 70_000

export interface SupportCitation {
  source?: string
  section?: string
  version?: string
  document_id?: string
  chunk_id?: string
  score?: number
}

export interface SupportTicketDraft {
  title: string
  description: string
  priority: string
  product: string
}

export interface SupportChatResult {
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

export interface SupportThreadTurn {
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
}
