import { computed, onUnmounted, ref, watch } from 'vue'
import { supportAPI, type SupportTicketRecord, type SupportTicketStats } from '@/api/support'
import { useAuthStore } from '@/stores/auth'
import type { TicketFormValues } from '@/components/support/ticketPresentation'

type TicketPortalAPI = Pick<typeof supportAPI, 'listTickets' | 'ticketStats' | 'getTicket' | 'replyTicket' | 'transitionTicket'> & Partial<Pick<typeof supportAPI, 'createTicket'>>

export function useTicketPortal(api: TicketPortalAPI = supportAPI, admin = false) {
  const auth = useAuthStore()
  const items = ref<SupportTicketRecord[]>([])
  const total = ref(0)
  const page = ref(1)
  const pageSize = 20
  const search = ref('')
  const type = ref('all')
  const status = ref('all')
  const listState = ref<'unavailable' | 'loading' | 'error' | 'ready'>('loading')
  const available = ref(false)
  const stats = ref<SupportTicketStats | null>(null)
  const statsError = ref(false)
  const detail = ref<SupportTicketRecord | null>(null)
  const detailState = ref<'loading' | 'error' | 'ready'>('loading')
  const actionBusy = ref(false)
  const replyBusy = ref(false)
  const actionError = ref(false)
  const filtered = computed(() => !!search.value.trim() || type.value !== 'all' || status.value !== 'all')
  let accountVersion = 0
  let listVersion = 0
  let statsVersion = 0
  let detailVersion = 0
  let disposed = false
  let searchTimer: ReturnType<typeof setTimeout> | undefined

  const authorized = () => !!auth.user?.id && (!admin || auth.isAdmin)
  const current = (version: number) => !disposed && version === accountVersion && authorized()
  async function loadList() {
    const account = accountVersion
    const version = ++listVersion
    items.value = []
    listState.value = 'loading'
    if (!authorized()) return
    try {
      const result = await api.listTickets({
        q: search.value.trim() || undefined,
        type: type.value === 'all' ? undefined : type.value,
        status: status.value === 'all' ? undefined : status.value,
        page: page.value, page_size: pageSize,
      })
      if (!current(account) || version !== listVersion) return
      if (!Array.isArray(result.items) || !Number.isSafeInteger(result.total) || result.total < 0) throw new Error('Invalid ticket list')
      items.value = result.items
      total.value = result.total
      available.value = true
      listState.value = 'ready'
    } catch (err) {
      if (!current(account) || version !== listVersion) return
      if ((err as { status?: number }).status === 404) {
        available.value = false
        listState.value = 'unavailable'
        statsVersion++
        stats.value = null
        statsError.value = false
      } else listState.value = 'error'
    }
  }
  async function loadStats() {
    const account = accountVersion
    const version = ++statsVersion
    stats.value = null
    statsError.value = false
    if (!authorized()) return
    try {
      const result = await api.ticketStats()
      if (!current(account) || version !== statsVersion) return
      if (!['all', 'pending_agent', 'pending_user', 'resolved', 'closed'].every((key) => Number.isSafeInteger(result[key as keyof SupportTicketStats]) && result[key as keyof SupportTicketStats] >= 0)) throw new Error('Invalid ticket statistics')
      stats.value = result
    } catch {
      if (current(account) && version === statsVersion) statsError.value = true
    }
  }
  async function refresh() {
    const version = accountVersion
    await loadList()
    if (current(version) && available.value) await loadStats()
  }
  async function open(id: string) {
    const account = accountVersion
    const version = ++detailVersion
    detail.value = null
    detailState.value = 'loading'
    actionError.value = false
    if (!authorized()) { detailState.value = 'error'; return }
    try {
      const result = await api.getTicket(id)
      if (!current(account) || version !== detailVersion) return
      if (result.id !== id) throw new Error('Invalid ticket detail')
      detail.value = result
      detailState.value = 'ready'
    } catch {
      if (current(account) && version === detailVersion) detailState.value = 'error'
    }
  }
  async function create(values: TicketFormValues, requestId: string) {
    if (!available.value || !authorized() || !api.createTicket) return false
    const account = accountVersion
    const result = await api.createTicket({
      type: values.type, title: values.title, description: values.description,
      product: values.product, priority: values.priority,
      order_id: values.orderId || undefined, attachment_ids: values.attachmentIds,
      client_request_id: requestId,
    })
    if (!current(account) || !result.id) return false
    return true
  }
  async function reply(text: string, attachmentIds: string[], messageId: string) {
    const record = detail.value
    if (!authorized() || !record?.allowed_actions?.includes('reply') || ['closed', 'resolved'].includes(record.status) || replyBusy.value || actionBusy.value) return false
    const account = accountVersion
    const version = detailVersion
    replyBusy.value = true
    try {
      const result = await api.replyTicket(record.id, { content: text, attachment_ids: attachmentIds, client_message_id: messageId })
      if (!current(account) || !result.id) return false
      void refresh()
      if (version !== detailVersion) return false
      // The write succeeded even if refreshing its timeline subsequently fails.
      await open(record.id)
      return current(account)
    } finally {
      if (current(account)) replyBusy.value = false
    }
  }
  async function transition(action: 'resolve' | 'close' | 'reopen') {
    const record = detail.value
    if (!authorized() || !record?.allowed_actions?.includes(action) || actionBusy.value || replyBusy.value) return false
    const account = accountVersion
    const version = detailVersion
    actionBusy.value = true
    actionError.value = false
    try {
      const result = await api.transitionTicket(record.id, action)
      if (!current(account) || version !== detailVersion || result.id !== record.id) return false
      detail.value = result
      void refresh()
      return true
    } catch {
      if (current(account) && version === detailVersion) actionError.value = true
      return false
    } finally {
      if (current(account)) actionBusy.value = false
    }
  }
  function closeDetail() {
    detailVersion++
    detail.value = null
    actionError.value = false
    actionBusy.value = false
    replyBusy.value = false
  }
  function applyFilters() {
    if (searchTimer) clearTimeout(searchTimer)
    if (page.value !== 1) page.value = 1
    else void loadList()
  }
  function clearFilters() {
    search.value = ''
    type.value = 'all'
    status.value = 'all'
  }
  watch([type, status], applyFilters)
  watch(page, () => { void loadList() })
  watch(search, () => {
    // Invalidate a response immediately, before the next debounced request.
    listVersion++
    items.value = []
    listState.value = 'loading'
    if (searchTimer) clearTimeout(searchTimer)
    searchTimer = setTimeout(applyFilters, 300)
  })
  watch([() => auth.user?.id, () => admin && auth.isAdmin], () => {
    accountVersion++
    listVersion++
    statsVersion++
    closeDetail()
    items.value = []
    total.value = 0
    stats.value = null
    statsError.value = false
    available.value = false
    listState.value = 'loading'
    search.value = ''
    type.value = 'all'
    status.value = 'all'
    page.value = 1
    if (searchTimer) clearTimeout(searchTimer)
  }, { flush: 'sync' })
  onUnmounted(() => {
    disposed = true
    if (searchTimer) clearTimeout(searchTimer)
  })
  return { items, total, page, pageSize, search, type, status, filtered, listState, available, stats, statsError, detail, detailState, actionBusy, replyBusy, actionError, loadList, loadStats, refresh, open, closeDetail, create, reply, transition, clearFilters }
}
