<template>
  <div :aria-busy="state === 'loading'">
    <div v-if="state === 'unavailable'" class="px-6 py-12 text-center" role="status">
      <Icon name="inbox" size="xl" class="mx-auto text-gray-400" />
      <h3 class="mt-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('support.listNotConnected') }}</h3>
      <p class="mx-auto mt-2 max-w-lg text-sm leading-6 text-gray-500 dark:text-gray-400">{{ t('support.noTicketsHint') }}</p>
      <button type="button" class="btn btn-primary mt-5" @click="emit('consultation')"><Icon name="chat" size="sm" class="mr-2" />{{ t('support.aiConsultation') }}</button>
    </div>
    <div v-else-if="state === 'loading'" role="status" class="space-y-4 px-5 py-6 sm:px-6">
      <span class="sr-only">{{ t('common.loading') }}</span>
      <div v-for="index in 3" :key="index" aria-hidden="true" class="h-28 animate-pulse rounded-lg bg-gray-100 dark:bg-dark-800" />
    </div>
    <div v-else-if="state === 'error'" role="alert" class="px-6 py-12 text-center">
      <Icon name="exclamationCircle" size="lg" class="mx-auto text-red-500" />
      <p class="mt-3 text-sm text-gray-600 dark:text-gray-300">{{ t('support.errors.loadList') }}</p>
      <button type="button" class="btn btn-secondary mt-4" @click="emit('retry')"><Icon name="refresh" size="sm" class="mr-2" />{{ t('common.refresh') }}</button>
    </div>
    <template v-else>
      <div v-if="items.length" class="divide-y divide-gray-200 dark:divide-dark-700">
        <SupportTicketRow v-for="item in items" :key="item.id" :ticket="item" @open="emit('open', $event)" />
      </div>
      <div v-else role="status" class="px-6 py-12 text-center">
        <Icon name="inbox" size="xl" class="mx-auto text-gray-400" />
        <h3 class="mt-4 text-base font-semibold text-gray-900 dark:text-white">{{ t(total > 0 ? 'support.emptyTicketPage' : filtered ? 'support.noMatchingTickets' : 'support.emptyTickets') }}</h3>
        <button v-if="total > 0" type="button" class="btn btn-secondary mt-4" @click="emit('update:page', 1)">{{ t('support.firstPage') }}</button>
        <button v-if="filtered" type="button" class="btn btn-secondary mt-4" @click="emit('clearFilters')">{{ t('support.clearFilters') }}</button>
      </div>
      <Pagination
        v-if="total > 0"
        :total="total"
        :page="page"
        :page-size="pageSize"
        :show-page-size-selector="false"
        @update:page="emit('update:page', $event)"
      />
    </template>
    <footer v-if="state === 'unavailable'" class="border-t border-gray-200 px-5 py-4 text-xs text-gray-500 sm:px-6 dark:border-dark-700 dark:text-gray-400">{{ t('support.paginationUnavailable') }}</footer>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import Pagination from '@/components/common/Pagination.vue'
import SupportTicketRow from './SupportTicketRow.vue'
import type { TicketListItem } from './ticketPresentation'

withDefaults(defineProps<{
  state: 'unavailable' | 'loading' | 'error' | 'ready'
  items?: TicketListItem[]
  filtered?: boolean
  total?: number
  page?: number
  pageSize?: number
}>(), { items: () => [], filtered: false, total: 0, page: 1, pageSize: 20 })
const emit = defineEmits<{
  open: [id: string]
  retry: []
  consultation: []
  clearFilters: []
  'update:page': [page: number]
}>()
const { t } = useI18n()
</script>
