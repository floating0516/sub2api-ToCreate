import type { SupportAttachment, SupportTicketRecord } from '@/api/support'

// Presentation models are separate from the Agent/BFF wire format.
export interface TicketListItem extends SupportTicketRecord {
  typeLabel?: string
  latestMessage?: string
  relatedOrderLabel?: string
}

export interface TicketFormValues {
  type: string
  title: string
  description: string
  product: string
  priority: string
  orderId: string
  attachmentIds: string[]
}

export interface TicketMessage {
  id: string
  role: 'user' | 'staff' | 'system' | 'unknown'
  content: string
  createdAt: string
  attachments?: SupportAttachment[]
}

export function priorityKey(priority: string) {
  const keys: Record<string, string> = { high: 'priorityHigh', normal: 'priorityNormal', low: 'priorityLow' }
  return `support.${Object.prototype.hasOwnProperty.call(keys, priority) ? keys[priority] : 'priorityUnknown'}`
}

export function ticketStatus(status: string) {
  const gray = 'bg-gray-100 text-gray-700 dark:bg-dark-800 dark:text-gray-300'
  const statuses: Record<string, { key: string; color: string }> = {
    submitted: { key: 'submitted', color: gray },
    pending_agent: { key: 'pendingAgent', color: 'bg-orange-50 text-orange-700 dark:bg-orange-950/30 dark:text-orange-300' },
    pending_user: { key: 'pendingUser', color: 'bg-blue-50 text-blue-700 dark:bg-blue-950/30 dark:text-blue-300' },
    resolved: { key: 'resolved', color: 'bg-emerald-50 text-emerald-700 dark:bg-emerald-950/30 dark:text-emerald-300' },
    closed: { key: 'closed', color: gray },
  }
  const entry = Object.prototype.hasOwnProperty.call(statuses, status) ? statuses[status]! : { key: 'unknown', color: gray }
  return { key: `support.status.${entry.key}`, color: entry.color }
}

export function ticketDate(value: string, locale: string): string | null {
  const date = new Date(value)
  if (!Number.isFinite(date.getTime())) return null
  return new Intl.DateTimeFormat(locale, { dateStyle: 'medium', timeStyle: 'short' }).format(date)
}

export function ticketTypeKey(type?: string) {
  const types: Record<string, string> = { 'pre-sale': 'preSale', order: 'order', 'after-sale': 'afterSale', technical: 'technical', other: 'other' }
  return type && Object.prototype.hasOwnProperty.call(types, type) ? `support.types.${types[type]}` : 'support.typeUnavailable'
}

export function ticketMessages(ticket: SupportTicketRecord): TicketMessage[] {
  return (ticket.replies || []).map((reply) => ({
    id: reply.id,
    role: reply.actor === 'user' ? 'user' : reply.actor === 'agent' ? 'staff' : reply.actor === 'system' ? 'system' : 'unknown',
    content: reply.content,
    createdAt: reply.created_at,
    attachments: reply.attachments,
  }))
}
