<template>
  <div class="min-h-screen bg-gray-50 px-4 py-10 dark:bg-dark-950 sm:py-16">
    <div class="mx-auto w-full max-w-xl">
      <div class="mb-8 flex items-center justify-center gap-3 text-gray-950 dark:text-white">
        <div class="flex h-10 w-10 items-center justify-center rounded-lg bg-gray-900 text-white dark:bg-white dark:text-gray-900">
          <Icon name="terminal" size="md" :stroke-width="2" />
        </div>
        <span class="text-lg font-semibold">{{ appStore.siteName || 'ToCreate' }}</span>
      </div>

      <main class="rounded-lg border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-600 dark:bg-dark-900 sm:p-8">
        <div v-if="loading" class="py-12 text-center">
          <div class="mx-auto h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
          <p class="mt-4 text-sm text-gray-500 dark:text-dark-400">
            {{ t('installConfirm.loading') }}
          </p>
        </div>

        <div v-else-if="invalidLink || loadError" class="py-4 text-center">
          <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-red-100 text-red-600 dark:bg-red-950/50 dark:text-red-300">
            <Icon name="xCircle" size="lg" :stroke-width="2" />
          </div>
          <h1 class="mt-5 text-xl font-semibold text-gray-950 dark:text-white">
            {{ invalidLink ? t('installConfirm.invalidTitle') : t('installConfirm.expiredTitle') }}
          </h1>
          <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
            {{ invalidLink
              ? t('installConfirm.invalidDescription')
              : t('installConfirm.expiredDescription') }}
          </p>
          <RouterLink to="/custom/codex-claude-import" class="btn btn-primary mt-6">
            <Icon name="arrowLeft" size="sm" class="mr-2" />
            {{ t('installConfirm.back') }}
          </RouterLink>
        </div>

        <template v-else-if="metadata">
          <div class="text-center">
            <div class="mx-auto flex h-12 w-12 items-center justify-center rounded-full bg-emerald-100 text-emerald-700 dark:bg-emerald-950/50 dark:text-emerald-300">
              <Icon name="checkCircle" size="lg" :stroke-width="2" />
            </div>
            <h1 class="mt-5 text-xl font-semibold text-gray-950 dark:text-white">
              {{ t('installConfirm.title') }}
            </h1>
            <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
              {{ t('installConfirm.subtitle') }}
            </p>
          </div>

          <dl class="mt-7 divide-y divide-gray-200 border-y border-gray-200 text-sm dark:divide-dark-700 dark:border-dark-700">
            <div class="grid grid-cols-[110px_minmax(0,1fr)] gap-4 py-3">
              <dt class="text-gray-500 dark:text-dark-400">{{ t('installConfirm.client') }}</dt>
              <dd class="font-medium text-gray-950 dark:text-white">{{ clientLabel }}</dd>
            </div>
            <div class="grid grid-cols-[110px_minmax(0,1fr)] gap-4 py-3">
              <dt class="text-gray-500 dark:text-dark-400">{{ t('installConfirm.key') }}</dt>
              <dd class="min-w-0">
                <span class="block truncate font-medium text-gray-950 dark:text-white">
                  {{ metadata.key.name }}
                </span>
                <code class="mt-0.5 block font-mono text-xs text-gray-500 dark:text-dark-400">
                  {{ metadata.key.prefix }}
                </code>
              </dd>
            </div>
            <div class="grid grid-cols-[110px_minmax(0,1fr)] gap-4 py-3">
              <dt class="text-gray-500 dark:text-dark-400">{{ t('installConfirm.endpoint') }}</dt>
              <dd class="break-all font-mono text-xs text-gray-700 dark:text-dark-200">
                {{ metadata.endpoint }}
              </dd>
            </div>
          </dl>

          <div
            v-if="importError"
            class="mt-5 rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-200"
          >
            {{ importError }}
          </div>

          <div
            v-if="ready"
            class="mt-5 flex items-start gap-3 rounded-lg border border-emerald-200 bg-emerald-50 p-4 text-sm text-emerald-900 dark:border-emerald-900/60 dark:bg-emerald-950/20 dark:text-emerald-100"
          >
            <Icon name="checkCircle" size="sm" class="mt-0.5 flex-none" />
            {{ t('installConfirm.ready') }}
          </div>

          <button
            type="button"
            class="btn btn-primary mt-6 w-full py-3"
            :disabled="importing"
            @click="openImport"
          >
            <Icon :name="ready ? 'externalLink' : 'download'" size="sm" class="mr-2" />
            {{ importing
              ? t('installConfirm.opening')
              : ready
                ? t('installConfirm.retryOpen')
                : t('installConfirm.open') }}
          </button>

          <p class="mt-3 text-center text-xs leading-5 text-gray-500 dark:text-dark-400">
            {{ t('installConfirm.browserHint') }}
          </p>

          <div class="mt-6 flex flex-col items-center gap-3 border-t border-gray-200 pt-5 text-sm dark:border-dark-700 sm:flex-row sm:justify-between">
            <a
              :href="ccSwitchDownloadUrl"
              class="font-medium text-gray-600 hover:text-gray-950 dark:text-dark-300 dark:hover:text-white"
            >
              {{ t('installConfirm.download') }}
            </a>
            <RouterLink
              to="/custom/codex-claude-import"
              class="font-medium text-gray-600 hover:text-gray-950 dark:text-dark-300 dark:hover:text-white"
            >
              {{ t('installConfirm.back') }}
            </RouterLink>
          </div>
        </template>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import type {
  InstallCredentialRequest,
  InstallTokenPeekResult,
  InstallTokenRedeemResult
} from '@/api/installTokens'
import { installTokensAPI } from '@/api'
import { useAppStore } from '@/stores/app'
import Icon from '@/components/icons/Icon.vue'
import {
  getBrowserPlatformMetadata,
  resolveInstallConfirmationAction,
  type InstallConfirmationAction
} from '@/utils/quickstart'

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const appStore = useAppStore()

const action = ref<InstallConfirmationAction | null>(null)
const metadata = ref<InstallTokenPeekResult | null>(null)
const loading = ref(true)
const loadError = ref('')
const importError = ref('')
const importing = ref(false)
const deeplink = ref('')
const ready = ref(false)

const invalidLink = computed(() => !action.value)

const clientLabel = computed(() => {
  switch (metadata.value?.client) {
    case 'claude-code':
      return t('quickStart.clientStep.claude')
    case 'codex':
      return t('quickStart.clientStep.codex')
    case 'gemini-cli':
      return t('quickStart.clientStep.gemini')
    default:
      return ''
  }
})

const ccSwitchDownloadUrl = computed(() => {
  const platform = getBrowserPlatformMetadata()
  if (platform.os === 'darwin') return '/download/cc-switch/macos'
  if (platform.os === 'linux') {
    return platform.arch === 'arm64'
      ? '/download/cc-switch/linux-arm64'
      : '/download/cc-switch/linux-x86_64'
  }
  return platform.arch === 'arm64'
    ? '/download/cc-switch/windows-arm64'
    : '/download/cc-switch/windows'
})

function credentialWithPlatform(request: InstallCredentialRequest): InstallCredentialRequest {
  return {
    ...request,
    ...getBrowserPlatformMetadata()
  }
}

async function loadMetadata() {
  if (!action.value) {
    loading.value = false
    return
  }
  loading.value = true
  loadError.value = ''
  try {
    metadata.value = await installTokensAPI.peek(action.value.request)
  } catch (error) {
    loadError.value = (error as { message?: string }).message || 'invalid'
  } finally {
    loading.value = false
  }
}

function launchDeeplink(value: string): boolean {
  try {
    window.location.assign(value)
    ready.value = true
    return true
  } catch {
    ready.value = false
    return false
  }
}

function receiptFromConfirmUrl(confirmUrl: string | undefined): string {
  if (!confirmUrl) return ''
  try {
    return new URL(confirmUrl, window.location.origin).searchParams.get('receipt') || ''
  } catch {
    return ''
  }
}

async function redeemCredential(): Promise<InstallTokenRedeemResult> {
  if (!action.value) {
    throw new Error(t('installConfirm.invalidDescription'))
  }
  const request = credentialWithPlatform(action.value.request)
  if (action.value.kind === 'receipt') {
    return installTokensAPI.confirm(request)
  }
  return installTokensAPI.redeem(request)
}

async function openImport() {
  if (deeplink.value) {
    if (!launchDeeplink(deeplink.value)) {
      importError.value = t('installConfirm.openFailed')
    }
    return
  }
  importing.value = true
  importError.value = ''
  try {
    const result = await redeemCredential()
    deeplink.value = result.deeplink
    if (action.value?.kind === 'token') {
      const receipt = receiptFromConfirmUrl(result.confirm_url)
      if (receipt) {
        action.value = {
          kind: 'receipt',
          request: { receipt }
        }
        await router.replace({
          path: '/custom/install-confirm',
          query: { receipt }
        })
      }
    }
    if (!launchDeeplink(result.deeplink)) {
      throw new Error(t('installConfirm.openFailed'))
    }
  } catch (error) {
    const candidate = error as { message?: string }
    importError.value = candidate.message || t('installConfirm.expiredDescription')
  } finally {
    importing.value = false
  }
}

onMounted(async () => {
  action.value = resolveInstallConfirmationAction(route.query)
  if (!appStore.publicSettingsLoaded) {
    await appStore.fetchPublicSettings().catch(() => null)
  }
  await loadMetadata()
})
</script>
