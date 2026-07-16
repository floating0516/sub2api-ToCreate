<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl space-y-6">
      <section class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900">
        <div class="flex flex-col gap-4 p-5 sm:flex-row sm:items-center sm:justify-between">
          <div class="flex min-w-0 items-center gap-3">
            <span
              class="flex h-10 w-10 shrink-0 items-center justify-center rounded-full"
              :class="connected ? 'bg-emerald-100 text-emerald-700 dark:bg-emerald-950/50 dark:text-emerald-400' : 'bg-gray-100 text-gray-500 dark:bg-dark-800 dark:text-dark-400'"
            >
              <Icon :name="connected ? 'checkCircle' : 'chat'" size="lg" />
            </span>
            <div class="min-w-0">
              <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                {{ connected ? t('liheOAuth.connected') : t('liheOAuth.notConnected') }}
              </h2>
              <p v-if="connected" class="mt-0.5 text-sm text-gray-500 dark:text-dark-400">
                {{ t('liheOAuth.activeConnections', { count: tokens.length }) }}
              </p>
            </div>
          </div>

          <button
            type="button"
            class="btn btn-primary inline-flex shrink-0 items-center justify-center"
            :disabled="loading || !integration?.enabled || !integration.connect_url"
            @click="openLiheChat"
          >
            <Icon name="externalLink" size="sm" class="mr-2" />
            {{ t('liheOAuth.importChat') }}
          </button>
        </div>
      </section>

      <section class="overflow-hidden rounded-lg border border-gray-200 bg-white dark:border-dark-700 dark:bg-dark-900">
        <div class="flex items-center justify-between border-b border-gray-200 px-5 py-4 dark:border-dark-700">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">{{ t('liheOAuth.connections') }}</h2>
          <button
            type="button"
            class="btn btn-secondary flex h-9 w-9 items-center justify-center p-0"
            :disabled="loading"
            :title="t('common.refresh')"
            @click="loadIntegration"
          >
            <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
          </button>
        </div>

        <div v-if="loading && tokens.length === 0" class="flex min-h-44 items-center justify-center text-gray-400">
          <Icon name="refresh" size="lg" class="animate-spin" />
        </div>

        <div v-else-if="tokens.length === 0" class="flex min-h-44 flex-col items-center justify-center px-5 text-center">
          <Icon name="link" size="xl" class="text-gray-300 dark:text-dark-600" />
          <p class="mt-3 text-sm text-gray-500 dark:text-dark-400">{{ t('liheOAuth.noConnections') }}</p>
        </div>

        <div v-else class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-200 dark:divide-dark-700">
            <thead class="bg-gray-50 dark:bg-dark-800/70">
              <tr>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('liheOAuth.name') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('liheOAuth.providers') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('liheOAuth.createdAt') }}</th>
                <th class="px-5 py-3 text-left text-xs font-medium uppercase text-gray-500 dark:text-dark-400">{{ t('liheOAuth.lastUsedAt') }}</th>
                <th class="w-16 px-5 py-3"><span class="sr-only">{{ t('liheOAuth.revoke') }}</span></th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-800">
              <tr v-for="token in tokens" :key="token.id">
                <td class="whitespace-nowrap px-5 py-4 text-sm font-medium text-gray-900 dark:text-white">{{ token.name }}</td>
                <td class="px-5 py-4">
                  <div class="flex min-w-52 flex-wrap gap-1.5">
                    <span
                      v-for="provider in token.providers"
                      :key="provider"
                      class="rounded-md bg-gray-100 px-2 py-1 text-xs font-medium text-gray-700 dark:bg-dark-800 dark:text-dark-300"
                    >
                      {{ provider }}
                    </span>
                  </div>
                </td>
                <td class="whitespace-nowrap px-5 py-4 text-sm text-gray-600 dark:text-dark-300">{{ formatDateTime(token.created_at) }}</td>
                <td class="whitespace-nowrap px-5 py-4 text-sm text-gray-600 dark:text-dark-300">
                  {{ token.last_used_at ? formatDateTime(token.last_used_at) : t('liheOAuth.neverUsed') }}
                </td>
                <td class="px-5 py-4 text-right">
                  <button
                    type="button"
                    class="inline-flex h-8 w-8 items-center justify-center rounded-md text-gray-400 hover:bg-red-50 hover:text-red-600 disabled:opacity-50 dark:hover:bg-red-950/30 dark:hover:text-red-400"
                    :disabled="revokingId === token.id"
                    :title="t('liheOAuth.revoke')"
                    @click="requestRevoke(token)"
                  >
                    <Icon :name="revokingId === token.id ? 'refresh' : 'trash'" size="sm" :class="revokingId === token.id ? 'animate-spin' : ''" />
                  </button>
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>
    </div>

    <ConfirmDialog
      :show="selectedToken !== null"
      :title="t('liheOAuth.revokeTitle')"
      :message="t('liheOAuth.revokeConfirm')"
      :confirm-text="t('liheOAuth.revoke')"
      :cancel-text="t('common.cancel')"
      :danger="true"
      :confirm-disabled="revokingId !== null"
      @confirm="confirmRevoke"
      @cancel="selectedToken = null"
    />
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import ConfirmDialog from '@/components/common/ConfirmDialog.vue'
import Icon from '@/components/icons/Icon.vue'
import liheOAuthAPI, { type LiheAccessTokenRecord, type LiheIntegration } from '@/api/liheOAuth'
import { useAppStore } from '@/stores'
import { extractApiErrorMessage } from '@/utils/apiError'
import { formatDateTime } from '@/utils/format'

const { t } = useI18n()
const appStore = useAppStore()
const integration = ref<LiheIntegration | null>(null)
const loading = ref(false)
const selectedToken = ref<LiheAccessTokenRecord | null>(null)
const revokingId = ref<number | null>(null)

const tokens = computed(() => integration.value?.tokens ?? [])
const connected = computed(() => tokens.value.length > 0)

async function loadIntegration() {
  loading.value = true
  try {
    integration.value = await liheOAuthAPI.getIntegration()
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('liheOAuth.loadFailed')))
  } finally {
    loading.value = false
  }
}

function openLiheChat() {
  const target = integration.value?.connect_url
  if (target) window.location.assign(target)
}

function requestRevoke(token: LiheAccessTokenRecord) {
  selectedToken.value = token
}

async function confirmRevoke() {
  if (!selectedToken.value || revokingId.value !== null) return
  revokingId.value = selectedToken.value.id
  try {
    await liheOAuthAPI.revokeToken(selectedToken.value.id)
    selectedToken.value = null
    appStore.showSuccess(t('liheOAuth.revoked'))
    await loadIntegration()
  } catch (error: unknown) {
    appStore.showError(extractApiErrorMessage(error, t('liheOAuth.revokeFailed')))
  } finally {
    revokingId.value = null
  }
}

onMounted(loadIntegration)
</script>
