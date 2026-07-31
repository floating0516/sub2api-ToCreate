<template>
  <section
    data-testid="official-version-status"
    class="rounded-lg border border-gray-200 bg-gray-50 p-3 dark:border-dark-700 dark:bg-dark-900"
  >
    <div class="flex items-start justify-between gap-3">
      <div class="min-w-0">
        <p class="text-xs font-semibold text-gray-700 dark:text-dark-200">
          {{ t('version.officialProject') }}
        </p>
        <p class="mt-1 flex flex-wrap items-center gap-x-1.5 gap-y-1 text-[11px] text-gray-500 dark:text-dark-400">
          <span>{{ t('version.officialBaseline') }}</span>
          <strong class="font-semibold text-gray-700 dark:text-dark-200">
            {{ currentVersion ? `v${currentVersion}` : '--' }}
          </strong>
          <span aria-hidden="true">·</span>
          <span>{{ t('version.officialLatest') }}</span>
          <strong class="font-semibold text-gray-700 dark:text-dark-200">
            {{ latestVersion ? `v${latestVersion}` : '--' }}
          </strong>
        </p>
      </div>

      <a
        data-testid="official-repository-link"
        :href="OFFICIAL_REPOSITORY_URL"
        target="_blank"
        rel="noopener noreferrer"
        class="inline-flex flex-shrink-0 items-center gap-1 text-xs font-medium text-primary-600 transition-colors hover:text-primary-700 hover:underline dark:text-primary-400 dark:hover:text-primary-300"
      >
        {{ t('version.officialRepository') }}
        <Icon name="externalLink" size="xs" :stroke-width="2" />
      </a>
    </div>

    <a
      v-if="hasUpdate && latestVersion"
      data-testid="official-update-link"
      :href="effectiveReleaseUrl"
      target="_blank"
      rel="noopener noreferrer"
      class="mt-3 flex items-start gap-2 rounded-lg border border-amber-200 bg-amber-50 px-2.5 py-2 text-amber-700 transition-colors hover:bg-amber-100 dark:border-amber-800/50 dark:bg-amber-900/20 dark:text-amber-300 dark:hover:bg-amber-900/30"
    >
      <Icon name="exclamationCircle" size="sm" :stroke-width="2" class="mt-0.5 flex-shrink-0" />
      <span class="min-w-0 flex-1">
        <span class="block text-xs font-semibold">
          {{ t('version.officialUpdateAvailable', { version: latestVersion }) }}
        </span>
        <span class="mt-0.5 block text-[11px] leading-4 opacity-80">
          {{ t('version.officialUpdateCustomHint') }}
        </span>
      </span>
      <Icon name="externalLink" size="xs" :stroke-width="2" class="mt-0.5 flex-shrink-0" />
    </a>

    <div
      v-else-if="latestVersion"
      data-testid="official-up-to-date"
      class="mt-3 flex items-center gap-2 rounded-lg border border-green-200 bg-green-50 px-2.5 py-2 text-xs text-green-700 dark:border-green-800/50 dark:bg-green-900/20 dark:text-green-300"
    >
      <Icon name="checkCircle" size="sm" :stroke-width="2" class="flex-shrink-0" />
      <span>{{ t('version.officialUpToDate') }}</span>
    </div>

    <div
      v-else
      data-testid="official-version-unavailable"
      class="mt-3 flex items-center gap-2 rounded-lg border border-gray-200 bg-white px-2.5 py-2 text-xs text-gray-500 dark:border-dark-700 dark:bg-dark-800 dark:text-dark-400"
    >
      <Icon name="infoCircle" size="sm" :stroke-width="2" class="flex-shrink-0" />
      <span>{{ t('version.officialCheckUnavailable') }}</span>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import {
  OFFICIAL_RELEASES_URL,
  OFFICIAL_REPOSITORY_URL
} from '@/constants/version'

const props = defineProps<{
  currentVersion?: string
  latestVersion?: string
  hasUpdate?: boolean
  releaseUrl?: string
}>()

const { t } = useI18n()

const effectiveReleaseUrl = computed(() => {
  const releaseUrl = props.releaseUrl?.trim()
  return releaseUrl && releaseUrl !== '#' ? releaseUrl : OFFICIAL_RELEASES_URL
})
</script>
