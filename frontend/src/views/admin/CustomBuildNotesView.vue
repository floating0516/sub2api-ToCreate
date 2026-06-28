<template>
  <AppLayout>
    <div class="mx-auto max-w-5xl">
      <article class="rounded-lg border border-gray-200 bg-white p-6 shadow-sm dark:border-dark-700 dark:bg-dark-900 sm:p-8">
        <header class="mb-6 border-b border-gray-200 pb-5 dark:border-dark-700">
          <p class="text-sm font-medium text-primary-700 dark:text-primary-300">
            {{ t('admin.customBuild.badge') }}
          </p>
          <h1 class="mt-2 text-2xl font-bold tracking-normal text-gray-950 dark:text-white">
            {{ t('admin.customBuild.title') }}
          </h1>
          <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
            {{ t('admin.customBuild.description') }}
          </p>
        </header>

        <div class="custom-build-content" v-html="renderedHtml"></div>
      </article>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import { getLocale } from '@/i18n'
import zhNotes from './custom-build-notes.zh.md?raw'
import enNotes from './custom-build-notes.en.md?raw'

const { t } = useI18n()

marked.setOptions({
  breaks: true,
  gfm: true,
})

const notesMarkdown = computed(() => getLocale() === 'zh' ? zhNotes : enNotes)

const renderedHtml = computed(() => {
  const html = marked.parse(notesMarkdown.value.trim()) as string
  return DOMPurify.sanitize(html)
})
</script>

<style scoped>
.custom-build-content {
  line-height: 1.75;
  overflow-wrap: anywhere;
  color: inherit;
}

.custom-build-content :deep(h1) {
  @apply mb-4 mt-8 border-b border-gray-200 pb-3 text-3xl font-bold dark:border-dark-700;
}

.custom-build-content :deep(h1:first-child) {
  @apply mt-0;
}

.custom-build-content :deep(h2) {
  @apply mb-3 mt-7 text-2xl font-bold;
}

.custom-build-content :deep(h3) {
  @apply mb-2 mt-6 text-xl font-semibold;
}

.custom-build-content :deep(p) {
  @apply mb-4 text-gray-700 dark:text-dark-200;
}

.custom-build-content :deep(ul) {
  @apply mb-4 list-disc pl-6;
}

.custom-build-content :deep(ol) {
  @apply mb-4 list-decimal pl-6;
}

.custom-build-content :deep(li) {
  @apply mb-1 text-gray-700 dark:text-dark-200;
}

.custom-build-content :deep(code) {
  @apply rounded bg-gray-100 px-1.5 py-0.5 font-mono text-sm dark:bg-dark-800;
}

.custom-build-content :deep(pre) {
  @apply my-5 overflow-x-auto rounded-lg bg-gray-950 p-4 text-gray-100;
}

.custom-build-content :deep(pre code) {
  @apply bg-transparent p-0 text-inherit;
}

.custom-build-content :deep(blockquote) {
  @apply my-5 border-l-4 border-gray-300 pl-4 text-gray-600 dark:border-dark-600 dark:text-dark-300;
}

.custom-build-content :deep(hr) {
  @apply my-7 border-gray-200 dark:border-dark-700;
}
</style>
