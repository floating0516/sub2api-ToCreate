<template>
  <AppLayout>
    <div class="mx-auto flex min-h-[calc(100vh-8rem)] max-w-7xl flex-col gap-6">
      <header class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
        <div class="flex min-w-0 items-start gap-3">
          <span class="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-primary-100 text-primary-700 dark:bg-primary-950/50 dark:text-primary-300">
            <Icon name="chat" size="lg" />
          </span>
          <div class="min-w-0">
            <h1 class="text-2xl font-semibold tracking-tight text-gray-900 dark:text-white">{{ t('support.title') }}</h1>
            <p class="mt-1 max-w-2xl text-sm text-gray-500 dark:text-gray-400">{{ t('support.subtitle') }}</p>
          </div>
        </div>
        <span v-if="store.threadId" class="inline-flex shrink-0 items-center gap-1.5 text-xs text-gray-500 dark:text-gray-400">
          <span class="h-1.5 w-1.5 rounded-full bg-emerald-500" />
          {{ t('support.saved') }}
        </span>
      </header>

      <div class="grid min-h-0 flex-1 gap-6 lg:grid-cols-[minmax(0,1fr)_18rem]">
        <section class="flex min-h-[34rem] min-w-0 flex-col overflow-hidden rounded-xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
          <div class="flex items-center justify-between border-b border-gray-200 px-5 py-4 dark:border-dark-700">
            <div>
              <h2 class="text-sm font-semibold text-gray-900 dark:text-white">{{ t('support.conversation') }}</h2>
              <p class="mt-0.5 text-xs text-gray-500 dark:text-gray-400">{{ t('support.conversationHint') }}</p>
            </div>
            <Icon name="shield" size="sm" class="text-gray-400" :title="t('support.securityTitle')" />
          </div>

          <div class="min-h-0 flex-1 overflow-y-auto px-5 py-5">
            <div v-if="!store.turns.length" class="flex min-h-[22rem] items-center justify-center text-center">
              <div class="max-w-md">
                <span class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-gray-100 text-gray-400 dark:bg-dark-800 dark:text-dark-400">
                  <Icon name="chatBubble" size="lg" />
                </span>
                <h3 class="mt-4 text-base font-semibold text-gray-900 dark:text-white">{{ t('support.emptyTitle') }}</h3>
                <p class="mt-2 text-sm text-gray-500 dark:text-gray-400">{{ t('support.emptyExample') }}</p>
                <div class="mt-5 flex flex-wrap justify-center gap-2">
                  <button
                    v-for="example in examples"
                    :key="example"
                    type="button"
                    class="rounded-full border border-gray-200 px-3 py-1.5 text-xs text-gray-600 transition hover:border-primary-300 hover:text-primary-600 dark:border-dark-600 dark:text-gray-300 dark:hover:border-primary-700 dark:hover:text-primary-300"
                    @click="message = example"
                  >
                    {{ example }}
                  </button>
                </div>
              </div>
            </div>

            <div v-else class="space-y-5">
              <article v-for="(turn, index) in store.turns" :key="`${index}-${turn.created_at || ''}`" :class="turn.role === 'user' ? 'ml-8 sm:ml-16' : 'mr-8 sm:mr-16'">
                <div class="mb-1 text-[11px] font-medium uppercase tracking-wide text-gray-400">
                  {{ turn.role === 'user' ? t('support.you') : t('support.agent') }}
                </div>
                <div :class="turn.role === 'user' ? 'bg-primary-50 text-gray-900 dark:bg-primary-950/40 dark:text-gray-100' : 'bg-gray-50 text-gray-800 dark:bg-dark-800 dark:text-gray-100'" class="rounded-xl px-4 py-3 text-sm leading-6">
                  <div class="whitespace-pre-wrap">{{ turn.content }}</div>
                  <div v-if="turn.evidence_status" class="mt-3 inline-flex rounded-full bg-white/70 px-2 py-0.5 text-xs text-gray-500 dark:bg-black/10 dark:text-gray-400">
                    {{ t('support.evidenceStatus', { status: evidenceLabel(turn.evidence_status) }) }}
                  </div>
                  <div v-if="turn.citations?.length" class="mt-3 border-t border-gray-200/80 pt-3 text-xs dark:border-dark-600">
                    <div class="mb-2 flex items-center gap-1.5 font-medium text-gray-600 dark:text-gray-300">
                      <Icon name="book" size="sm" />
                      {{ t('support.citations') }}
                    </div>
                    <div class="space-y-1.5">
                      <div v-for="(citation, citationIndex) in turn.citations" :key="citationIndex" class="flex flex-wrap gap-x-1 text-gray-500 dark:text-gray-400">
                        <span class="font-medium text-gray-700 dark:text-gray-300">{{ citation.source || citation.document_id || t('support.document') }}</span>
                        <span v-if="citation.section">· {{ citation.section }}</span>
                        <span v-if="citation.version">· {{ citation.version }}</span>
                      </div>
                    </div>
                  </div>
                </div>
              </article>
            </div>
          </div>

          <div v-if="store.error" role="alert" class="mx-5 mb-3 flex items-start gap-2 rounded-lg border border-red-200 bg-red-50 px-3 py-2.5 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300">
            <Icon name="exclamationCircle" size="sm" class="mt-0.5 shrink-0" />
            <span>{{ t(store.error) }}</span>
          </div>

          <form class="border-t border-gray-200 p-4 dark:border-dark-700" @submit.prevent="submit">
            <textarea v-model="message" rows="3" :disabled="store.loading" class="block w-full resize-none rounded-lg border border-gray-300 bg-gray-50 px-3 py-2.5 text-sm outline-none transition placeholder:text-gray-400 focus:border-primary-500 focus:bg-white dark:border-dark-600 dark:bg-dark-800 dark:text-white dark:focus:bg-dark-900" :placeholder="t('support.placeholder')" />
            <div class="mt-2 flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
              <span class="text-xs text-gray-500 dark:text-gray-400">{{ t('support.inputHint') }}</span>
              <button type="submit" :disabled="store.loading || !message.trim()" class="btn btn-primary inline-flex shrink-0 items-center justify-center disabled:cursor-not-allowed disabled:opacity-50">
                <Icon name="arrowRight" size="sm" class="mr-2" />
                {{ store.loading ? t('support.processing') : t('support.send') }}
              </button>
            </div>
          </form>
        </section>

        <aside class="flex min-w-0 flex-col gap-4">
          <section class="rounded-xl border border-gray-200 bg-white p-5 dark:border-dark-700 dark:bg-dark-900">
            <div class="flex items-center gap-2 text-sm font-semibold text-gray-900 dark:text-white">
              <Icon name="shield" size="sm" class="text-primary-600 dark:text-primary-400" />
              {{ t('support.securityTitle') }}
            </div>
            <p class="mt-3 text-sm leading-6 text-gray-600 dark:text-gray-300">{{ t('support.securityDescription') }}</p>
          </section>

          <section v-if="store.hasDraft" class="rounded-xl border border-amber-300 bg-amber-50 p-5 dark:border-amber-700 dark:bg-amber-950/30">
            <div class="flex items-center gap-2 text-sm font-semibold text-amber-900 dark:text-amber-200">
              <Icon name="inbox" size="sm" />
              {{ t('support.ticketDraftTitle') }}
            </div>
            <h3 class="mt-3 text-sm font-medium text-amber-950 dark:text-amber-100">{{ store.draft?.title }}</h3>
            <p class="mt-2 whitespace-pre-wrap text-sm leading-5 text-amber-800 dark:text-amber-300">{{ store.draft?.description }}</p>
            <dl class="mt-3 space-y-1 text-xs text-amber-800 dark:text-amber-300">
              <div><dt class="inline font-medium">{{ t('support.product') }}:</dt> <dd class="inline">{{ store.draft?.product }}</dd></div>
              <div><dt class="inline font-medium">{{ t('support.priority') }}:</dt> <dd class="inline">{{ store.draft?.priority }}</dd></div>
            </dl>
            <div class="mt-4 flex flex-col gap-2">
              <button class="btn btn-primary w-full justify-center" :disabled="store.loading" @click="store.confirmTicket(true)">{{ t('support.confirmTicket') }}</button>
              <button class="btn btn-secondary w-full justify-center" :disabled="store.loading" @click="store.confirmTicket(false)">{{ t('support.cancelTicket') }}</button>
            </div>
          </section>

          <div v-if="store.ticketId" class="rounded-xl border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-800 dark:border-emerald-900/60 dark:bg-emerald-950/30 dark:text-emerald-300">
            <div class="flex items-center gap-2 font-medium"><Icon name="checkCircle" size="sm" />{{ t('support.ticketSubmitted', { id: store.ticketId }) }}</div>
          </div>
        </aside>
      </div>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { useSupportStore } from '@/stores/support'

const store = useSupportStore()
const { t, te } = useI18n()
const message = ref('')
const examples = [
  'Codex 怎么配？',
  '它返回 401 了怎么办？',
]

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
