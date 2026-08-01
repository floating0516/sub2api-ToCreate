<template>
  <section
    data-testid="custom-update-resolver-settings"
    class="mb-3 border-y border-gray-100 bg-gray-50 px-3 py-3 dark:border-dark-700 dark:bg-dark-900/45"
  >
    <div class="mb-3 flex items-start gap-2">
      <span
        class="flex h-7 w-7 flex-shrink-0 items-center justify-center rounded-lg bg-primary-100 text-primary-600 dark:bg-primary-900/30 dark:text-primary-300"
      >
        <Icon name="sparkles" size="xs" :stroke-width="2" />
      </span>
      <div class="min-w-0 flex-1">
        <p class="text-xs font-semibold text-gray-800 dark:text-dark-100">
          {{ t('version.customUpdateResolverSettingsTitle') }}
        </p>
        <p class="mt-0.5 text-[10px] leading-4 text-gray-500 dark:text-dark-400">
          {{
            configSaved
              ? t('version.customUpdateResolverConfigSaved')
              : t('version.customUpdateResolverUsingDefaults')
          }}
        </p>
      </div>
    </div>

    <div v-if="loading" class="flex min-h-24 items-center justify-center">
      <Icon name="refresh" size="sm" :stroke-width="2" class="animate-spin text-primary-500" />
    </div>

    <div v-else-if="loadError" class="space-y-2">
      <p class="bg-red-50 px-2.5 py-2 text-xs leading-4 text-red-600 dark:bg-red-900/20 dark:text-red-400">
        {{ loadError }}
      </p>
      <button
        type="button"
        class="flex h-8 w-full items-center justify-center gap-1.5 rounded-lg border border-gray-200 bg-white text-xs font-medium text-gray-700 transition-colors hover:bg-gray-50 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-200 dark:hover:bg-dark-700"
        @click="loadConfig"
      >
        <Icon name="refresh" size="xs" :stroke-width="2" />
        {{ t('version.retry') }}
      </button>
    </div>

    <form v-else class="space-y-3" @submit.prevent="saveConfig">
      <label class="block">
        <span class="mb-1 block text-[11px] font-medium text-gray-600 dark:text-dark-300">
          {{ t('version.customUpdateResolverBaseURL') }}
        </span>
        <input
          v-model="baseURL"
          type="url"
          inputmode="url"
          autocomplete="url"
          spellcheck="false"
          class="input h-9 w-full text-xs"
          :placeholder="defaultBaseURL"
          required
          @input="clearFormFeedback"
        />
      </label>

      <label class="block">
        <span class="mb-1 flex items-center justify-between gap-2 text-[11px] font-medium text-gray-600 dark:text-dark-300">
          <span>{{ t('version.customUpdateResolverAPIKey') }}</span>
          <span
            class="flex items-center gap-1 text-[10px] font-normal"
            :class="
              apiKeyConfigured
                ? 'text-green-600 dark:text-green-400'
                : 'text-amber-600 dark:text-amber-400'
            "
          >
            <Icon
              :name="apiKeyConfigured ? 'checkCircle' : 'exclamationCircle'"
              size="xs"
              :stroke-width="2"
            />
            {{
              apiKeyConfigured
                ? t('version.customUpdateResolverAPIKeyConfigured')
                : t('version.customUpdateResolverAPIKeyMissing')
            }}
          </span>
        </span>
        <span class="relative block">
          <input
            v-model="apiKey"
            :type="showAPIKey ? 'text' : 'password'"
            autocomplete="new-password"
            spellcheck="false"
            class="input h-9 w-full pr-9 text-xs"
            :placeholder="
              apiKeyConfigured
                ? t('version.customUpdateResolverAPIKeyKeepPlaceholder')
                : t('version.customUpdateResolverAPIKeyPlaceholder')
            "
            @input="clearFormFeedback"
          />
          <button
            type="button"
            class="absolute inset-y-0 right-0 flex w-9 items-center justify-center text-gray-400 transition-colors hover:text-gray-700 dark:text-dark-400 dark:hover:text-dark-100"
            :title="
              showAPIKey
                ? t('version.customUpdateResolverHideAPIKey')
                : t('version.customUpdateResolverShowAPIKey')
            "
            :aria-label="
              showAPIKey
                ? t('version.customUpdateResolverHideAPIKey')
                : t('version.customUpdateResolverShowAPIKey')
            "
            @click="showAPIKey = !showAPIKey"
          >
            <Icon :name="showAPIKey ? 'eyeOff' : 'eye'" size="xs" :stroke-width="2" />
          </button>
        </span>
      </label>

      <label class="block">
        <span class="mb-1 block text-[11px] font-medium text-gray-600 dark:text-dark-300">
          {{ t('version.customUpdateResolverModel') }}
        </span>
        <select
          v-model="selectedModel"
          class="input h-9 w-full text-xs"
          @change="clearFormFeedback"
        >
          <option value="gpt-5.6-luna">
            {{ t('version.customUpdateResolverModelLuna') }}
          </option>
          <option value="gpt-5.6-terra">
            {{ t('version.customUpdateResolverModelTerra') }}
          </option>
          <option :value="customModelOption">
            {{ t('version.customUpdateResolverModelCustom') }}
          </option>
        </select>
      </label>

      <label v-if="selectedModel === customModelOption" class="block">
        <span class="mb-1 block text-[11px] font-medium text-gray-600 dark:text-dark-300">
          {{ t('version.customUpdateResolverCustomModel') }}
        </span>
        <input
          v-model="customModel"
          type="text"
          autocomplete="off"
          spellcheck="false"
          class="input h-9 w-full font-mono text-xs"
          :placeholder="defaultModel"
          required
          @input="clearFormFeedback"
        />
      </label>

      <div class="flex items-center justify-between gap-3 text-[10px] text-gray-500 dark:text-dark-400">
        <span>{{ t('version.customUpdateResolverReasoningMax') }}</span>
        <span v-if="updatedAt">{{ formatUpdatedAt(updatedAt) }}</span>
      </div>

      <p
        v-if="saveError"
        class="bg-red-50 px-2.5 py-2 text-xs leading-4 text-red-600 dark:bg-red-900/20 dark:text-red-400"
      >
        {{ saveError }}
      </p>
      <p
        v-else-if="saveSuccess"
        class="flex items-center gap-1.5 bg-green-50 px-2.5 py-2 text-xs text-green-700 dark:bg-green-900/20 dark:text-green-300"
      >
        <Icon name="checkCircle" size="xs" :stroke-width="2" />
        {{ t('version.customUpdateResolverSaveSuccess') }}
      </p>

      <p
        v-if="testError"
        class="bg-red-50 px-2.5 py-2 text-xs leading-4 text-red-600 dark:bg-red-900/20 dark:text-red-400"
      >
        {{ testError }}
      </p>
      <p
        v-else-if="testSuccess"
        class="flex items-center gap-1.5 bg-green-50 px-2.5 py-2 text-xs text-green-700 dark:bg-green-900/20 dark:text-green-300"
      >
        <Icon name="checkCircle" size="xs" :stroke-width="2" />
        {{ t('version.customUpdateResolverTestSuccess', { latency: testLatencyMS }) }}
      </p>

      <div class="flex items-stretch gap-2">
        <button
          type="button"
          data-testid="custom-update-resolver-test"
          class="flex h-9 flex-shrink-0 items-center justify-center gap-1.5 whitespace-nowrap rounded-lg border border-gray-200 bg-white px-3 text-xs font-medium text-gray-700 transition-colors hover:bg-gray-50 disabled:cursor-not-allowed disabled:opacity-50 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-200 dark:hover:bg-dark-700"
          :disabled="testDisabled"
          @click="testConnection"
        >
          <Icon
            :name="testingConnection ? 'refresh' : 'beaker'"
            size="xs"
            :stroke-width="2"
            :class="{ 'animate-spin': testingConnection }"
          />
          {{
            testingConnection
              ? t('version.customUpdateResolverTesting')
              : t('version.customUpdateResolverTestConnection')
          }}
        </button>

        <button
          type="submit"
          class="flex h-9 min-w-0 flex-1 items-center justify-center gap-1.5 whitespace-nowrap rounded-lg bg-primary-500 px-3 text-xs font-medium text-white transition-colors hover:bg-primary-600 disabled:cursor-not-allowed disabled:opacity-50"
          :disabled="saveDisabled"
        >
          <Icon
            :name="saving ? 'refresh' : 'check'"
            size="xs"
            :stroke-width="2"
            :class="{ 'animate-spin': saving }"
          />
          {{
            saving
              ? t('common.saving')
              : t('version.customUpdateResolverSaveDefault')
          }}
        </button>
      </div>
    </form>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import {
  getCustomUpdateResolverConfig,
  testCustomUpdateResolverConfig,
  updateCustomUpdateResolverConfig,
  type CustomUpdateResolverConfig,
  type CustomUpdateResolverTestResult,
  type UpdateCustomUpdateResolverConfigRequest
} from '@/api/admin/customBuild'

const emit = defineEmits<{
  saved: [config: CustomUpdateResolverConfig]
}>()

const { t } = useI18n()
const customModelOption = '__custom__'
const presetModels = new Set(['gpt-5.6-luna', 'gpt-5.6-terra'])

const loading = ref(true)
const saving = ref(false)
const testingConnection = ref(false)
const loadError = ref('')
const saveError = ref('')
const saveSuccess = ref(false)
const testError = ref('')
const testSuccess = ref(false)
const testLatencyMS = ref(0)
const configSaved = ref(false)
const apiKeyConfigured = ref(false)
const defaultBaseURL = ref('https://api.lihe.chat')
const defaultModel = ref('gpt-5.6-luna')
const baseURL = ref('')
const apiKey = ref('')
const showAPIKey = ref(false)
const selectedModel = ref('gpt-5.6-luna')
const customModel = ref('')
const updatedAt = ref('')

const resolvedModel = computed(() =>
  selectedModel.value === customModelOption ? customModel.value.trim() : selectedModel.value
)

const saveDisabled = computed(
  () =>
    saving.value ||
    testingConnection.value ||
    baseURL.value.trim() === '' ||
    resolvedModel.value === ''
)

const testDisabled = computed(
  () =>
    testingConnection.value ||
    saving.value ||
    baseURL.value.trim() === '' ||
    resolvedModel.value === '' ||
    (apiKey.value.trim() === '' && !apiKeyConfigured.value)
)

function applyConfig(config: CustomUpdateResolverConfig) {
  configSaved.value = config.saved
  apiKeyConfigured.value = config.api_key_configured
  defaultBaseURL.value = config.default_base_url
  defaultModel.value = config.default_model
  baseURL.value = config.base_url
  updatedAt.value = config.updated_at || ''
  apiKey.value = ''
  showAPIKey.value = false

  if (presetModels.has(config.model)) {
    selectedModel.value = config.model
    customModel.value = ''
  } else {
    selectedModel.value = customModelOption
    customModel.value = config.model
  }
}

function clearSaveFeedback() {
  saveError.value = ''
  saveSuccess.value = false
}

function clearTestFeedback() {
  testError.value = ''
  testSuccess.value = false
  testLatencyMS.value = 0
}

function clearFormFeedback() {
  clearSaveFeedback()
  clearTestFeedback()
}

function errorMessage(error: unknown, fallback: string): string {
  const value = error as { response?: { data?: { message?: string } }; message?: string }
  return value.response?.data?.message || value.message || fallback
}

function formatUpdatedAt(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat(undefined, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  }).format(date)
}

async function loadConfig() {
  loading.value = true
  loadError.value = ''
  try {
    applyConfig(await getCustomUpdateResolverConfig())
  } catch (error: unknown) {
    loadError.value = errorMessage(
      error,
      t('version.customUpdateResolverLoadFailed')
    )
  } finally {
    loading.value = false
  }
}

async function saveConfig() {
  if (saveDisabled.value) return

  saving.value = true
  clearFormFeedback()
  const payload: UpdateCustomUpdateResolverConfigRequest = {
    base_url: baseURL.value.trim(),
    model: resolvedModel.value
  }
  if (apiKey.value.trim() !== '') {
    payload.api_key = apiKey.value
  }

  try {
    const config = await updateCustomUpdateResolverConfig(payload)
    applyConfig(config)
    saveSuccess.value = true
    emit('saved', config)
  } catch (error: unknown) {
    saveError.value = errorMessage(
      error,
      t('version.customUpdateResolverSaveFailed')
    )
  } finally {
    saving.value = false
  }
}

async function testConnection() {
  if (testDisabled.value) return

  testingConnection.value = true
  clearFormFeedback()
  const payload: UpdateCustomUpdateResolverConfigRequest = {
    base_url: baseURL.value.trim(),
    model: resolvedModel.value
  }
  if (apiKey.value.trim() !== '') {
    payload.api_key = apiKey.value
  }

  try {
    const result: CustomUpdateResolverTestResult = await testCustomUpdateResolverConfig(payload)
    testLatencyMS.value = result.latency_ms
    testSuccess.value = true
  } catch (error: unknown) {
    testError.value = errorMessage(
      error,
      t('version.customUpdateResolverTestFailed')
    )
  } finally {
    testingConnection.value = false
  }
}

onMounted(loadConfig)
</script>
