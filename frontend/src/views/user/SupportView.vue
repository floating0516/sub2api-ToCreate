<template>
  <div class="mx-auto flex h-full max-w-5xl flex-col px-4 py-6 sm:px-6">
    <div class="mb-5 flex items-start justify-between gap-4">
      <div>
        <h1 class="text-2xl font-semibold text-gray-900 dark:text-white">{{ t('support.title') }}</h1>
        <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('support.subtitle') }}</p>
      </div>
      <span v-if="store.threadId" class="text-xs text-gray-400">{{ t('support.saved') }}</span>
    </div>

    <div class="min-h-0 flex-1 overflow-y-auto rounded-lg border border-gray-200 bg-white p-4 dark:border-dark-700 dark:bg-dark-800">
      <div v-if="!store.turns.length" class="flex min-h-[280px] items-center justify-center text-center text-sm text-gray-500 dark:text-gray-400">
        <div>
          <p class="font-medium text-gray-700 dark:text-gray-200">{{ t('support.emptyTitle') }}</p>
          <p class="mt-2">{{ t('support.emptyExample') }}</p>
        </div>
      </div>
      <div v-else class="space-y-5">
        <article v-for="(turn, index) in store.turns" :key="`${index}-${turn.created_at || ''}`" :class="turn.role === 'user' ? 'ml-8' : 'mr-8'">
          <div :class="turn.role === 'user' ? 'bg-primary-50 text-gray-900 dark:bg-primary-900/20 dark:text-gray-100' : 'bg-gray-50 text-gray-800 dark:bg-dark-700 dark:text-gray-100'" class="rounded-lg px-4 py-3 text-sm leading-6">
            <div class="whitespace-pre-wrap">{{ turn.content }}</div>
            <div v-if="turn.evidence_status" class="mt-3 text-xs text-gray-500 dark:text-gray-400">{{ t('support.evidenceStatus', { status: evidenceLabel(turn.evidence_status) }) }}</div>
            <div v-if="turn.citations?.length" class="mt-3 border-t border-gray-200 pt-2 text-xs dark:border-dark-600">
              <div class="mb-1 font-medium text-gray-600 dark:text-gray-300">{{ t('support.citations') }}</div>
              <div v-for="(citation, citationIndex) in turn.citations" :key="citationIndex" class="text-gray-500 dark:text-gray-400">
                {{ citation.source || citation.document_id || t('support.document') }}<span v-if="citation.section"> · {{ citation.section }}</span><span v-if="citation.version"> · {{ citation.version }}</span>
              </div>
            </div>
          </div>
        </article>
      </div>
    </div>

    <div v-if="store.hasDraft" class="mt-4 rounded-lg border border-amber-300 bg-amber-50 p-4 text-sm dark:border-amber-700 dark:bg-amber-900/20">
      <div class="font-medium text-amber-900 dark:text-amber-200">{{ t('support.ticketDraftTitle') }}</div>
      <div class="mt-1 text-amber-800 dark:text-amber-300">{{ store.draft?.title }}</div>
      <p class="mt-2 whitespace-pre-wrap text-amber-800 dark:text-amber-300">{{ store.draft?.description }}</p>
      <dl class="mt-3 flex flex-wrap gap-x-6 gap-y-1 text-xs text-amber-800 dark:text-amber-300">
        <div>
          <dt class="inline font-medium">{{ t('support.product') }}:</dt>
          <dd class="ml-1 inline">{{ store.draft?.product }}</dd>
        </div>
        <div>
          <dt class="inline font-medium">{{ t('support.priority') }}:</dt>
          <dd class="ml-1 inline">{{ store.draft?.priority }}</dd>
        </div>
      </dl>
      <div class="mt-3 flex flex-wrap gap-2">
        <button class="rounded-md bg-primary-600 px-3 py-2 text-sm font-medium text-white disabled:opacity-50" :disabled="store.loading" @click="store.confirmTicket(true)">{{ t('support.confirmTicket') }}</button>
        <button class="rounded-md border border-gray-300 px-3 py-2 text-sm disabled:opacity-50 dark:border-dark-500" :disabled="store.loading" @click="store.confirmTicket(false)">{{ t('support.cancelTicket') }}</button>
      </div>
    </div>
    <div v-if="store.ticketId" class="mt-4 text-sm text-green-700 dark:text-green-300">{{ t('support.ticketSubmitted', { id: store.ticketId }) }}</div>
    <div v-if="store.error" class="mt-3 text-sm text-red-600 dark:text-red-400">{{ t(store.error) }}</div>

    <form class="mt-4" @submit.prevent="submit">
      <textarea v-model="message" rows="3" :disabled="store.loading" class="w-full resize-y rounded-lg border border-gray-300 bg-white px-3 py-2 text-sm outline-none focus:border-primary-500 dark:border-dark-600 dark:bg-dark-800 dark:text-white" :placeholder="t('support.placeholder')" />
      <div class="mt-2 flex items-center justify-between gap-3">
        <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('support.inputHint') }}</span>
        <button type="submit" :disabled="store.loading || !message.trim()" class="rounded-md bg-primary-600 px-4 py-2 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-50">{{ store.loading ? t('support.processing') : t('support.send') }}</button>
      </div>
    </form>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useSupportStore } from '@/stores/support'

const store = useSupportStore()
const { t, te } = useI18n()
const message = ref('')

function evidenceLabel(status: string) {
  const key = `support.evidence.${status}`
  return te(key) ? t(key) : status
}

onMounted(() => store.refresh())

async function submit() {
  const value = message.value
  const sent = await store.send(value)
  if (sent) message.value = ''
}
</script>
