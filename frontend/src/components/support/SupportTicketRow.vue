<template>
  <button
    type="button"
    data-testid="ticket-row"
    class="group flex w-full min-w-0 flex-col gap-4 px-5 py-6 text-left transition hover:bg-gray-50 focus-visible:outline focus-visible:outline-2 focus-visible:outline-primary-500 sm:px-6 lg:flex-row lg:items-center dark:hover:bg-dark-800"
    @click="emit('open', ticket.id)"
  >
    <div class="min-w-0 flex-1">
      <div class="flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-gray-400">
        <span class="break-all font-mono">{{ ticket.id }}</span>
        <span class="max-w-full break-words rounded-full bg-gray-100 px-2 py-1 dark:bg-dark-800">{{ ticket.typeLabel || t('support.typeUnavailable') }}</span>
        <span :class="ticket.priority === 'high' ? 'text-red-600 dark:text-red-300' : ''">{{ t(priorityKey(ticket.priority)) }}</span>
      </div>
      <h3 class="mt-2 break-words text-sm font-semibold text-gray-900 dark:text-white">{{ ticket.title }}</h3>
      <p class="mt-1 line-clamp-2 break-words text-sm leading-6 text-gray-500 dark:text-gray-400">{{ ticket.latestMessage || ticket.description }}</p>
      <p v-if="ticket.product" class="mt-2 break-words text-xs text-gray-500 dark:text-gray-400">{{ t('support.product') }}: {{ ticket.product }}</p>
      <p v-if="ticket.relatedOrderLabel" class="mt-2 break-words text-xs text-gray-500 dark:text-gray-400">{{ t('support.relatedOrder') }}: {{ ticket.relatedOrderLabel }}</p>
    </div>
    <div class="flex min-w-0 flex-wrap items-center gap-3 lg:max-w-sm lg:justify-end">
      <span :class="['max-w-full break-words rounded-full px-2.5 py-1 text-xs font-medium', status.color]">{{ t(status.key) }}</span>
      <time :datetime="date ? ticket.updated_at : undefined" class="text-xs text-gray-500 dark:text-gray-400">{{ date || t('support.notProvided') }}</time>
      <Icon name="chevronRight" size="sm" class="shrink-0 text-gray-400 group-hover:text-primary-600" />
    </div>
  </button>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { priorityKey, ticketDate, ticketStatus, type TicketListItem } from './ticketPresentation'

const props = defineProps<{ ticket: TicketListItem }>()
const emit = defineEmits<{ open: [id: string] }>()
const { t, locale } = useI18n()
const status = computed(() => ticketStatus(props.ticket.status))
const date = computed(() => ticketDate(props.ticket.updated_at, locale.value))
</script>
