<template>
  <section class="flex min-w-0 flex-col overflow-hidden rounded-3xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
    <header class="flex flex-wrap items-center justify-between gap-3 border-b border-gray-200 px-5 py-4 dark:border-dark-700">
      <div class="min-w-0">
        <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('support.aiConsultation') }}</h2>
        <p class="mt-1 text-sm leading-6 text-gray-500 dark:text-gray-400">{{ t('support.aiConversationHint') }}</p>
      </div>
      <span v-if="store.turns.length && !busy && !store.error" class="text-xs text-gray-500 dark:text-gray-400">{{ t('support.saved') }}</span>
    </header>

    <div ref="conversation" role="log" :aria-label="t('support.conversation')" aria-live="polite" :aria-busy="busy" class="max-h-[36rem] min-h-64 space-y-5 overflow-y-auto p-5 sm:p-6">
      <p v-if="restoring" role="status" class="py-12 text-center text-sm text-gray-500 dark:text-gray-400">{{ t('common.loading') }}</p>
      <div v-else-if="!store.turns.length" class="py-10 text-center">
        <Icon name="chatBubble" size="xl" class="mx-auto text-primary-500" />
        <h3 class="mt-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('support.emptyTitle') }}</h3>
        <p class="mx-auto mt-2 max-w-md text-sm leading-6 text-gray-500 dark:text-gray-400">{{ t('support.emptyExample') }}</p>
      </div>
      <template v-else>
        <article
          v-for="(turn, index) in store.turns"
          :key="`${turn.role}-${turn.client_message_id || turn.reply_to || index}`"
          :class="turn.role === 'user' ? 'ml-4 sm:ml-16' : 'mr-4 sm:mr-16'"
        >
          <p class="mb-1 text-xs text-gray-500 dark:text-gray-400">{{ t(turn.role === 'user' ? 'support.you' : 'support.agent') }}</p>
          <div :class="['rounded-xl px-4 py-3 text-sm leading-6 text-gray-800 dark:text-gray-100', turn.role === 'user' ? 'bg-primary-50 dark:bg-primary-950/40' : 'bg-gray-50 dark:bg-dark-800']">
            <p class="whitespace-pre-wrap break-words">{{ turn.content }}</p>
            <div v-if="turn.evidence_action" class="mt-3 text-xs text-gray-600 dark:text-gray-300" data-testid="evidence-decision">
              <p class="font-medium">{{ t(`support.decisions.${turn.evidence_action}`) }}</p>
              <p v-if="turn.evidence_reason_code" class="mt-1">{{ reasonLabel(turn.evidence_reason_code) }}</p>
              <template v-if="turn.missing_information?.length">
                <p class="mt-2 font-medium">{{ t('support.missingInformation') }}</p>
                <ul class="mt-1 list-disc space-y-1 pl-5">
                  <li v-for="item in turn.missing_information" :key="item" class="break-words">{{ item }}</li>
                </ul>
              </template>
            </div>
            <p v-else-if="turn.evidence_status" class="mt-3 text-xs text-gray-500 dark:text-gray-400">{{ t('support.evidenceStatus', { status: evidenceLabel(turn.evidence_status) }) }}</p>
            <div v-if="turn.citations?.length" class="mt-3 border-t border-gray-200 pt-3 text-xs dark:border-dark-600">
              <p class="mb-2 flex items-center gap-2 font-medium">
                <Icon name="book" size="sm" />{{ t('support.retrievedSources') }}
              </p>
              <div v-for="(citation, citationIndex) in turn.citations" :key="citation.chunk_id || citationIndex" class="mb-2 break-words text-gray-500 dark:text-gray-400">
                {{ citation.source || citation.document_id || t('support.document') }}<span v-if="citation.section"> / {{ citation.section }}</span><span v-if="citation.version"> / {{ citation.version }}</span>
                <span v-if="citation.chunk_id && turn.supported_source_ids?.includes(citation.chunk_id)" class="ml-2 text-emerald-700 dark:text-emerald-400">{{ t('support.supportedSource') }}</span>
                <details v-if="citation.excerpt" class="mt-1">
                  <summary class="cursor-pointer text-primary-600 dark:text-primary-400">{{ t('support.viewExcerpt') }}</summary>
                  <blockquote class="mt-2 whitespace-pre-wrap border-l-2 border-gray-300 pl-3 text-gray-600 dark:border-dark-500 dark:text-gray-300">{{ citation.excerpt }}</blockquote>
                </details>
              </div>
            </div>
          </div>
        </article>
      </template>
      <p v-if="store.loading" role="status" class="text-sm text-gray-500 dark:text-gray-400">{{ t('support.processing') }}</p>
    </div>

    <div v-if="store.hasDraft" class="border-t border-amber-200 bg-amber-50 px-5 py-4 dark:border-amber-900 dark:bg-amber-950/30">
      <h3 class="flex items-center gap-2 text-sm font-semibold text-amber-900 dark:text-amber-200"><Icon name="inbox" size="sm" />{{ t('support.ticketDraftTitle') }}</h3>
      <p class="mt-3 break-words text-sm font-medium text-amber-950 dark:text-amber-100">{{ store.draft?.title }}</p>
      <p class="mt-2 whitespace-pre-wrap break-words text-sm leading-6 text-amber-800 dark:text-amber-300">{{ store.draft?.description }}</p>
      <dl class="mt-3 flex flex-wrap gap-x-5 gap-y-2 text-xs text-amber-800 dark:text-amber-300">
        <div class="min-w-0 break-words"><dt class="inline font-medium">{{ t('support.product') }}: </dt><dd class="inline">{{ store.draft?.product }}</dd></div>
        <div><dt class="inline font-medium">{{ t('support.priority') }}: </dt><dd class="inline">{{ store.draft?.priority }}</dd></div>
      </dl>
      <div class="mt-4 flex flex-col gap-2 sm:flex-row">
        <button
          type="button"
          data-testid="confirm-draft"
          class="btn btn-primary justify-center"
          :disabled="busy"
          @click="confirmDraft(true)"
        >
          <Icon name="check" size="sm" class="mr-2" />{{ t('support.confirmTicket') }}
        </button>
        <button type="button" class="btn btn-secondary justify-center" :disabled="busy" @click="confirmDraft(false)">
          {{ t('support.cancelTicket') }}
        </button>
      </div>
    </div>
    <p v-if="store.ticketId" class="break-words border-t border-emerald-200 bg-emerald-50 px-5 py-3 text-sm text-emerald-800 dark:border-emerald-900 dark:bg-emerald-950/30 dark:text-emerald-300">{{ t('support.ticketSubmitted', { id: store.ticketId }) }}</p>
    <div v-if="store.error" role="alert" class="mx-5 mb-3 mt-3 flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-3 text-sm text-red-700 dark:border-red-900 dark:bg-red-950/30 dark:text-red-300">
      <Icon name="exclamationCircle" size="sm" class="mt-0.5 shrink-0" />
      <span>{{ t(store.error) }}</span>
      <button
        v-if="store.error === 'support.errors.loadThread'"
        type="button"
        class="ml-auto shrink-0"
        :title="t('common.refresh')"
        :aria-label="t('common.refresh')"
        :disabled="restoring || store.loading"
        @click="emit('refresh')"
      >
        <Icon name="refresh" size="sm" />
      </button>
    </div>
    <form class="border-t border-gray-200 p-5 dark:border-dark-700" @submit.prevent="submit">
      <label for="support-message" class="sr-only">{{ t('support.formDescription') }}</label>
      <textarea
        id="support-message"
        v-model="message"
        rows="3"
        maxlength="4000"
        :disabled="busy"
        class="input block min-h-28 w-full resize-y"
        :placeholder="t('support.placeholder')"
      />
      <div class="mt-3 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
        <p class="flex min-w-0 items-start gap-2 text-xs leading-5 text-gray-500 dark:text-gray-400">
          <Icon name="shield" size="sm" class="mt-0.5 shrink-0" />{{ t('support.securityDescription') }}
        </p>
        <button type="submit" class="btn btn-primary shrink-0 justify-center" :disabled="busy || !message.trim()">
          <Icon name="arrowRight" size="sm" class="mr-2" />{{ t(store.loading ? 'support.processing' : 'support.send') }}
        </button>
      </div>
    </form>
  </section>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import { useSupportStore } from '@/stores/support'
import { useAppStore } from '@/stores/app'
import { useAuthStore } from '@/stores/auth'

const props = defineProps<{ restoring: boolean }>()
const emit = defineEmits<{ (event: 'refresh'): void }>()
const store = useSupportStore()
const appStore = useAppStore()
const authStore = useAuthStore()
const { t, te } = useI18n()
const message = ref('')
const conversation = ref<HTMLElement | null>(null)
const busy = computed(() => props.restoring || store.loading || store.error === 'support.errors.loadThread')

watch(() => authStore.user?.id, () => { message.value = '' }, { flush: 'sync' })

function evidenceLabel(status: string) {
  const key = `support.evidence.${status}`
  return te(key) ? t(key) : status
}
function reasonLabel(reason: string) {
  const key = `support.reasons.${reason}`
  return te(key) ? t(key) : t('support.reasons.unknown')
}
async function submit() {
  if (busy.value) return
  const sent = await store.send(message.value)
  if (sent) message.value = ''
}
async function confirmDraft(confirm: boolean) {
  if (busy.value) return
  const succeeded = await store.confirmTicket(confirm)
  if (succeeded) appStore.showSuccess(t(confirm ? 'support.created' : 'support.draftCancelled'))
}
watch(() => store.turns.length, async () => {
  await nextTick()
  if (conversation.value) conversation.value.scrollTop = conversation.value.scrollHeight
})
</script>
