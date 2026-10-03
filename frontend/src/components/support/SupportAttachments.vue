<template>
  <ul v-if="attachments.length" :aria-label="t('support.attachments')" class="mt-3 space-y-2">
    <li v-for="attachment in attachments" :key="attachment.id" class="min-w-0">
      <button type="button" :disabled="!!downloading || !downloadFile" class="inline-flex max-w-full items-start gap-2 text-left text-sm text-primary-600 hover:underline disabled:opacity-50 dark:text-primary-400" @click="download(attachment)">
        <Icon name="download" size="sm" class="mt-0.5 shrink-0" /><span class="break-all">{{ attachment.filename }}</span>
        <span v-if="downloading === attachment.id" role="status">{{ t('common.loading') }}</span>
      </button>
    </li>
    <li v-if="error" role="alert" class="text-sm text-red-600 dark:text-red-300">{{ t('support.errors.downloadAttachment') }}</li>
  </ul>
</template>

<script setup lang="ts">
import { onUnmounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import type { SupportAttachment } from '@/api/support'

const props = defineProps<{ attachments: SupportAttachment[]; downloadFile?: (id: string) => Promise<Blob> }>()
const { t } = useI18n()
const downloading = ref('')
const error = ref(false)
let disposed = false
async function download(attachment: SupportAttachment) {
  if (!props.downloadFile || downloading.value) return
  downloading.value = attachment.id
  error.value = false
  try {
    const blob = await props.downloadFile(attachment.id)
    if (disposed) return
    if (!(blob instanceof Blob)) throw new Error('Invalid attachment response')
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = attachment.filename
    document.body.appendChild(link)
    link.click()
    link.remove()
    setTimeout(() => URL.revokeObjectURL(url), 1000)
  } catch {
    if (!disposed) error.value = true
  } finally {
    if (!disposed) downloading.value = ''
  }
}
onUnmounted(() => { disposed = true })
</script>
