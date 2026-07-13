<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl">
      <article class="rounded-lg border border-gray-200 bg-white p-5 shadow-sm dark:border-dark-700 dark:bg-dark-900 sm:p-6">
        <header class="mb-6 border-b border-gray-200 pb-5 dark:border-dark-700 sm:flex sm:items-start sm:justify-between sm:gap-4">
          <div class="min-w-0">
            <p class="text-sm font-medium text-primary-700 dark:text-primary-300">
              {{ t('admin.customBuild.badge') }}
            </p>
            <h1 class="mt-2 text-2xl font-bold tracking-normal text-gray-950 dark:text-white">
              {{ t('admin.customBuild.title') }}
            </h1>
            <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
              {{ t('admin.customBuild.description') }}
            </p>
            <p v-if="notes?.updated_at" class="mt-2 text-xs text-gray-500 dark:text-dark-400">
              {{ t('admin.customBuild.updatedAt', { time: formatUpdatedAt(notes.updated_at) }) }}
            </p>
          </div>
          <button
            type="button"
            class="btn btn-secondary mt-4 flex-shrink-0 sm:mt-0"
            :disabled="loading"
            @click="loadNotes"
          >
            <Icon name="refresh" size="sm" :class="loading ? 'animate-spin' : ''" />
            {{ t('common.refresh') }}
          </button>
        </header>

        <div v-if="loading && !notes" class="flex min-h-[240px] items-center justify-center">
          <div class="h-8 w-8 animate-spin rounded-full border-b-2 border-primary-600"></div>
        </div>

        <div
          v-else-if="loadError"
          class="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-500/30 dark:bg-red-500/10 dark:text-red-200"
        >
          {{ loadError }}
        </div>

        <div v-else-if="renderedHtml" class="custom-build-shell">
          <aside v-if="tocItems.length > 0" class="custom-build-toc">
            <div class="custom-build-toc-title">{{ t('admin.customBuild.toc') }}</div>
            <nav class="custom-build-toc-nav">
              <a
                v-for="item in tocItems"
                :key="item.id"
                :href="'#' + item.id"
                class="custom-build-toc-item"
                :class="[
                  `custom-build-toc-level-${item.level}`,
                  { 'custom-build-toc-active': activeHeadingId === item.id }
                ]"
                @click.prevent="scrollToHeading(item.id)"
              >
                {{ item.text }}
              </a>
            </nav>
          </aside>
          <div
            ref="contentRef"
            class="custom-build-content"
            v-html="renderedHtml"
          ></div>
        </div>

        <div
          v-else
          class="rounded-lg border border-dashed border-gray-300 bg-gray-50 px-6 py-14 text-center text-sm text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-400"
        >
          {{ t('admin.customBuild.empty') }}
        </div>
      </article>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import Icon from '@/components/icons/Icon.vue'
import { customBuildAPI, type CustomBuildNotes } from '@/api/admin/customBuild'

const { t } = useI18n()
const loading = ref(false)
const loadError = ref('')
const notes = ref<CustomBuildNotes | null>(null)
const contentRef = ref<HTMLElement | null>(null)
const activeHeadingId = ref('')
let scrollRafId = 0

interface TocItem {
  id: string
  text: string
  level: number
}

marked.setOptions({
  breaks: true,
  gfm: true,
})

const renderedContent = computed(() => {
  const content = notes.value?.content?.trim() || ''
  if (!content) {
    return { html: '', toc: [] as TocItem[] }
  }
  const html = marked.parse(content) as string
  const sanitized = DOMPurify.sanitize(html)
  return injectHeadingIds(sanitized)
})

const renderedHtml = computed(() => renderedContent.value.html)
const tocItems = computed(() => renderedContent.value.toc)

function generateHeadingId(text: string, index: number): string {
  const base = text
    .toLowerCase()
    .replace(/[^\w一-鿿]+/g, '-')
    .replace(/^-+|-+$/g, '')
  return base ? `${base}-${index}` : `heading-${index}`
}

function injectHeadingIds(html: string): { html: string; toc: TocItem[] } {
  const toc: TocItem[] = []
  let headingIndex = 0
  const withIds = html.replace(
    /<(h[1-4])[^>]*>(.*?)<\/h[1-4]>/gi,
    (_, tag: string, content: string) => {
      const level = Number.parseInt(tag[1], 10)
      const text = content.replace(/<[^>]+>/g, '').trim()
      if (!text) return `<${tag}>${content}</${tag}>`
      const id = generateHeadingId(text, headingIndex++)
      toc.push({ id, text, level })
      return `<${tag} id="${id}">${content}</${tag}>`
    },
  )
  return { html: withIds, toc }
}

function scrollToHeading(id: string) {
  const container = contentRef.value
  if (!container) return
  const el = container.querySelector(`#${CSS.escape(id)}`)
  if (!el) return
  el.scrollIntoView({ behavior: 'smooth', block: 'start' })
  activeHeadingId.value = id
}

function updateActiveHeading() {
  if (scrollRafId) return
  scrollRafId = requestAnimationFrame(() => {
    scrollRafId = 0
    const container = contentRef.value
    if (!container || tocItems.value.length === 0) return

    let current = tocItems.value[0]?.id || ''
    for (const item of tocItems.value) {
      const el = container.querySelector(`#${CSS.escape(item.id)}`) as HTMLElement | null
      if (!el) continue
      if (el.getBoundingClientRect().top <= 120) {
        current = item.id
      }
    }
    activeHeadingId.value = current
  })
}

function formatUpdatedAt(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleString()
}

async function loadNotes() {
  loading.value = true
  loadError.value = ''
  try {
    notes.value = await customBuildAPI.getNotes()
  } catch (error: any) {
    loadError.value = error?.message || t('admin.customBuild.loadFailed')
  } finally {
    loading.value = false
  }
}

watch(renderedHtml, async () => {
  await nextTick()
  activeHeadingId.value = tocItems.value[0]?.id || ''
})

onMounted(() => {
  loadNotes()
  window.addEventListener('scroll', updateActiveHeading, { passive: true })
})

onUnmounted(() => {
  window.removeEventListener('scroll', updateActiveHeading)
  if (scrollRafId) {
    cancelAnimationFrame(scrollRafId)
    scrollRafId = 0
  }
})
</script>

<style scoped>
.custom-build-shell {
  @apply grid gap-6 lg:grid-cols-[220px_minmax(0,1fr)];
}

.custom-build-toc {
  @apply hidden lg:block;
  position: sticky;
  top: 88px;
  max-height: calc(100vh - 120px);
  overflow: auto;
}

.custom-build-toc-title {
  @apply mb-2 px-2 text-xs font-semibold uppercase tracking-wide text-gray-500 dark:text-dark-400;
}

.custom-build-toc-nav {
  @apply space-y-1 border-l border-gray-200 pl-2 dark:border-dark-700;
}

.custom-build-toc-item {
  @apply block truncate rounded px-2 py-1.5 text-sm text-gray-600 transition-colors hover:bg-gray-100 hover:text-gray-950 dark:text-dark-300 dark:hover:bg-dark-800 dark:hover:text-white;
}

.custom-build-toc-active {
  @apply bg-primary-50 font-medium text-primary-700 dark:bg-primary-900/20 dark:text-primary-300;
}

.custom-build-toc-level-1 {
  padding-left: 0.5rem;
}

.custom-build-toc-level-2 {
  padding-left: 1rem;
}

.custom-build-toc-level-3 {
  padding-left: 1.5rem;
}

.custom-build-toc-level-4 {
  padding-left: 2rem;
}

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
