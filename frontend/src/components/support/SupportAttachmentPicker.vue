<template>
  <div>
    <input ref="input" type="file" multiple accept=".png,.jpg,.jpeg,.webp,.pdf,.txt" class="sr-only" tabindex="-1" :disabled="!uploadFile || disabled || busy" @change="selectFiles" />
    <button type="button" class="btn btn-secondary" :disabled="!uploadFile || disabled || busy || entries.length >= maxFiles" @click="input?.click()"><Icon name="upload" size="sm" class="mr-2" />{{ t('support.chooseAttachment') }}</button>
    <p class="mt-2 text-xs leading-5 text-gray-500 dark:text-gray-400">{{ t(uploadFile ? 'support.attachmentLimits' : 'support.attachmentUnavailable') }}</p>
    <p v-if="selectionError" role="alert" class="mt-2 text-sm text-red-600 dark:text-red-300">{{ t(selectionError) }}</p>
    <ul v-if="entries.length" class="mt-3 space-y-2">
      <li v-for="entry in entries" :key="entry.key" class="flex min-w-0 flex-wrap items-center gap-2 border-b border-gray-200 py-2 text-sm dark:border-dark-700">
        <span class="min-w-0 flex-1 break-all">{{ entry.file.name }}</span>
        <span v-if="entry.state === 'uploading'" role="status" class="text-xs text-gray-500">{{ t('support.uploading') }}</span>
        <span v-else-if="entry.state === 'ready'" class="text-xs text-emerald-700 dark:text-emerald-300">{{ t('support.uploaded') }}</span>
        <template v-else>
          <span role="alert" class="text-xs text-red-600 dark:text-red-300">{{ t('support.errors.uploadAttachment') }}</span>
          <button type="button" class="btn btn-ghost btn-sm" :disabled="disabled || busy" :title="t('common.refresh')" :aria-label="t('common.refresh')" @click="upload(entry)"><Icon name="refresh" size="sm" /></button>
        </template>
        <button type="button" class="btn btn-ghost btn-sm" :disabled="disabled || busy" :title="t('common.remove')" :aria-label="t('common.remove')" @click="remove(entry.key)"><Icon name="x" size="sm" /></button>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
import { computed, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { SupportAttachment } from '@/api/support'

interface Entry { key: number; file: File; state: 'uploading' | 'ready' | 'error'; attachment?: SupportAttachment }
const props = withDefaults(defineProps<{
  uploadFile?: (file: File) => Promise<SupportAttachment>
  maxFiles?: number
  disabled?: boolean
  resetKey?: number
}>(), { maxFiles: 5, disabled: false, resetKey: 0 })
const emit = defineEmits<{ change: [ids: string[]]; pending: [value: boolean]; busy: [value: boolean] }>()
const { t } = useI18n()
const input = ref<HTMLInputElement | null>(null)
const entries = ref<Entry[]>([])
const selectionError = ref('')
const busy = computed(() => entries.value.some((entry) => entry.state === 'uploading'))
let nextKey = 0
let version = 0
let disposed = false

function publish() {
  emit('change', entries.value.flatMap((entry) => entry.attachment ? [entry.attachment.id] : []))
  emit('pending', entries.value.some((entry) => entry.state !== 'ready'))
  emit('busy', busy.value)
}
async function upload(entry: Entry) {
  if (!props.uploadFile || props.disabled || busy.value && entry.state !== 'uploading') return
  const requestVersion = version
  entry.state = 'uploading'
  publish()
  try {
    const result = await props.uploadFile(entry.file)
    if (disposed || version !== requestVersion) return
    if (!result.id) throw new Error('Invalid upload response')
    entry.attachment = result
    entry.state = 'ready'
  } catch {
    if (!disposed && version === requestVersion) entry.state = 'error'
  }
  if (!disposed && version === requestVersion) publish()
}
async function selectFiles(event: Event) {
  const target = event.target as HTMLInputElement
  const files = Array.from(target.files || [])
  target.value = ''
  if (!props.uploadFile || props.disabled || busy.value) return
  selectionError.value = ''
  const requestVersion = version
  for (const file of files) {
    if (disposed || version !== requestVersion) return
    if (entries.value.length >= props.maxFiles) { selectionError.value = 'support.validation.attachmentCount'; break }
    if (!file.size || file.size > 10 * 1024 * 1024) { selectionError.value = 'support.validation.attachmentSize'; continue }
    if (!/\.(png|jpe?g|webp|pdf|txt)$/i.test(file.name) || file.type && !['image/png', 'image/jpeg', 'image/webp', 'application/pdf', 'text/plain'].includes(file.type)) { selectionError.value = 'support.validation.attachmentType'; continue }
    entries.value.push({ key: ++nextKey, file, state: 'uploading' })
    await upload(entries.value[entries.value.length - 1]!)
  }
}
function remove(key: number) {
  entries.value = entries.value.filter((entry) => entry.key !== key)
  selectionError.value = ''
  publish()
}
watch(() => props.resetKey, () => {
  version++
  entries.value = []
  selectionError.value = ''
  publish()
})
onUnmounted(() => { disposed = true })
</script>
