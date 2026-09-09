<template>
  <div class="space-y-4">
    <div v-if="loading" data-testid="channels-loading" class="py-16 text-center">
      <Icon name="refresh" size="lg" class="inline-block animate-spin text-gray-400" />
    </div>

    <div v-else-if="rows.length === 0" data-testid="channels-empty" class="py-16 text-center">
      <Icon name="inbox" size="xl" class="mx-auto mb-3 h-12 w-12 text-gray-400" />
      <p class="text-sm text-gray-500 dark:text-gray-400">{{ emptyLabel }}</p>
    </div>

    <article
      v-else
      v-for="(channel, chIdx) in rows"
      :key="`${channel.name}-${chIdx}`"
      data-testid="channel-card"
      class="overflow-visible rounded-2xl border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800"
    >
      <header class="flex flex-wrap items-start justify-between gap-3 border-b border-gray-100 px-5 py-4 dark:border-dark-700">
        <div class="min-w-0">
          <h2 class="text-base font-semibold text-gray-900 dark:text-white">
            {{ channel.name }}
          </h2>
          <p
            v-if="channel.description"
            class="mt-1 max-w-3xl text-sm leading-6 text-gray-500 dark:text-gray-400"
          >
            {{ channel.description }}
          </p>
        </div>
        <p class="text-xs text-gray-400 dark:text-gray-500">
          {{ t('availableChannels.summary', {
            groups: groupCount(channel),
            models: modelCount(channel),
          }) }}
        </p>
      </header>

      <div
        v-for="section in channel.platforms"
        :key="`${channel.name}-${section.platform}`"
        class="space-y-4 px-5 py-4 [&:not(:first-of-type)]:border-t [&:not(:first-of-type)]:border-gray-100 dark:[&:not(:first-of-type)]:border-dark-700"
      >
        <div class="flex flex-wrap items-center gap-2">
          <span
            :class="[
              'inline-flex items-center gap-1 rounded-md border px-2 py-0.5 text-[11px] font-medium uppercase',
              platformBadgeClass(section.platform),
            ]"
          >
            <PlatformIcon :platform="section.platform as GroupPlatform" size="xs" />
            {{ section.platform }}
          </span>
        </div>

        <section class="min-w-0">
          <h3 class="mb-2 text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
            {{ columns.groups }}
          </h3>
          <div class="flex flex-col gap-2">
            <div
              v-if="exclusiveGroups(section).length > 0"
              class="flex min-w-0 flex-wrap items-center gap-1.5"
            >
              <span
                class="inline-flex items-center gap-0.5 text-[10px] font-medium uppercase text-purple-600 dark:text-purple-400"
                :title="t('availableChannels.exclusiveTooltip')"
              >
                <Icon name="shield" size="xs" class="h-3 w-3" />
                {{ t('availableChannels.exclusive') }}
              </span>
              <div
                v-for="g in visibleGroups(exclusiveGroups(section), sectionKey(channel.name, section.platform, 'ex'))"
                :key="`ex-${g.id}`"
                class="inline-flex max-w-full min-w-0 flex-wrap items-center gap-1"
              >
                <GroupBadge
                  class="max-w-full"
                  :name="g.name"
                  :platform="g.platform as GroupPlatform"
                  :subscription-type="(g.subscription_type || 'standard') as SubscriptionType"
                  :rate-multiplier="g.rate_multiplier"
                  :user-rate-multiplier="userGroupRates[g.id] ?? null"
                  always-show-rate
                />
                <span
                  v-if="hasPeakRate(g)"
                  class="inline-flex items-center gap-1 rounded-md bg-amber-50 px-1.5 py-0.5 text-[10px] font-medium text-amber-700 dark:bg-amber-900/20 dark:text-amber-300"
                  :title="peakRateTitle(g)"
                >
                  <Icon name="clock" size="xs" class="h-3 w-3" />
                  {{ peakRateLabel(g) }}
                </span>
              </div>
              <button
                v-if="canToggleGroups(exclusiveGroups(section))"
                type="button"
                class="text-[11px] text-primary-600 hover:underline dark:text-primary-400"
                @click="toggleGroups(sectionKey(channel.name, section.platform, 'ex'))"
              >
                {{ groupToggleLabel(exclusiveGroups(section), sectionKey(channel.name, section.platform, 'ex')) }}
              </button>
            </div>

            <div
              v-if="publicGroups(section).length > 0"
              class="flex min-w-0 flex-wrap items-center gap-1.5"
            >
              <span
                class="inline-flex items-center gap-0.5 text-[10px] font-medium uppercase text-gray-500 dark:text-gray-400"
                :title="t('availableChannels.publicTooltip')"
              >
                <Icon name="globe" size="xs" class="h-3 w-3" />
                {{ t('availableChannels.public') }}
              </span>
              <div
                v-for="g in visibleGroups(publicGroups(section), sectionKey(channel.name, section.platform, 'pub'))"
                :key="`pub-${g.id}`"
                class="inline-flex max-w-full min-w-0 flex-wrap items-center gap-1"
              >
                <GroupBadge
                  class="max-w-full"
                  :name="g.name"
                  :platform="g.platform as GroupPlatform"
                  :subscription-type="(g.subscription_type || 'standard') as SubscriptionType"
                  :rate-multiplier="g.rate_multiplier"
                  :user-rate-multiplier="userGroupRates[g.id] ?? null"
                  always-show-rate
                />
                <span
                  v-if="hasPeakRate(g)"
                  class="inline-flex items-center gap-1 rounded-md bg-amber-50 px-1.5 py-0.5 text-[10px] font-medium text-amber-700 dark:bg-amber-900/20 dark:text-amber-300"
                  :title="peakRateTitle(g)"
                >
                  <Icon name="clock" size="xs" class="h-3 w-3" />
                  {{ peakRateLabel(g) }}
                </span>
              </div>
              <button
                v-if="canToggleGroups(publicGroups(section))"
                type="button"
                class="text-[11px] text-primary-600 hover:underline dark:text-primary-400"
                @click="toggleGroups(sectionKey(channel.name, section.platform, 'pub'))"
              >
                {{ groupToggleLabel(publicGroups(section), sectionKey(channel.name, section.platform, 'pub')) }}
              </button>
            </div>

            <span v-if="section.groups.length === 0" class="text-xs text-gray-400">-</span>
          </div>
        </section>

        <section class="min-w-0">
          <div class="mb-2 flex flex-wrap items-baseline justify-between gap-2">
            <h3 class="text-[11px] font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
              {{ columns.supportedModels }}
            </h3>
            <p class="text-[11px] text-gray-400 dark:text-gray-500">
              {{ t('availableChannels.modelHint') }}
            </p>
          </div>
          <div
            v-if="section.supported_models.length > 0"
            class="grid grid-cols-1 gap-2 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4"
          >
            <button
              v-for="m in section.supported_models"
              :key="`${section.platform}-${m.name}`"
              type="button"
              class="flex min-w-0 items-center justify-start rounded-lg border border-gray-100 bg-gray-50/70 px-2.5 py-2 text-left transition-colors hover:border-gray-200 hover:bg-white dark:border-dark-700 dark:bg-dark-900/40 dark:hover:border-dark-600 dark:hover:bg-dark-800"
              :title="t('availableChannels.copyModel')"
              @click="copyModel(m.name)"
            >
              <SupportedModelChip
                class="max-w-full min-w-0 [&>span]:max-w-full [&>span]:truncate"
                :model="m"
                :pricing-key-prefix="pricingKeyPrefix"
                :no-pricing-label="noPricingLabel"
                :show-platform="false"
                :platform-hint="section.platform"
              />
            </button>
          </div>
          <span v-else class="text-xs text-gray-400">{{ noModelsLabel }}</span>
        </section>
      </div>
    </article>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import Icon from '@/components/icons/Icon.vue'
import PlatformIcon from '@/components/common/PlatformIcon.vue'
import GroupBadge from '@/components/common/GroupBadge.vue'
import SupportedModelChip from './SupportedModelChip.vue'
import type { UserAvailableChannel, UserAvailableGroup, UserChannelPlatformSection } from '@/api/channels'
import type { GroupPlatform, SubscriptionType } from '@/types'
import { platformBadgeClass } from '@/utils/platformColors'
import { useAppStore } from '@/stores/app'
import { useClipboard } from '@/composables/useClipboard'
import { hasPeakRate as groupHasPeakRate, formatPeakRateWindow, serverTimezoneLabel } from '@/utils/peak-rate'

const GROUP_PREVIEW_LIMIT = 8

const props = defineProps<{
  columns: {
    name: string
    description: string
    platform: string
    groups: string
    supportedModels: string
  }
  rows: UserAvailableChannel[]
  loading: boolean
  pricingKeyPrefix: string
  noPricingLabel: string
  noModelsLabel: string
  emptyLabel: string
  /** 用户专属倍率（group_id → multiplier）；无专属时由 GroupBadge 仅显示默认倍率。 */
  userGroupRates: Record<number, number>
}>()

void props.userGroupRates
void props.columns

const { t } = useI18n()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()
const expanded = ref<Record<string, boolean>>({})

function exclusiveGroups(section: UserChannelPlatformSection): UserAvailableGroup[] {
  return section.groups.filter((g) => g.is_exclusive)
}

function publicGroups(section: UserChannelPlatformSection): UserAvailableGroup[] {
  return section.groups.filter((g) => !g.is_exclusive)
}

function sectionKey(channelName: string, platform: string, kind: 'ex' | 'pub'): string {
  return `${channelName}::${platform}::${kind}`
}

function isExpanded(key: string): boolean {
  return Boolean(expanded.value[key])
}

function visibleGroups(groups: UserAvailableGroup[], key: string): UserAvailableGroup[] {
  if (isExpanded(key) || groups.length <= GROUP_PREVIEW_LIMIT) return groups
  return groups.slice(0, GROUP_PREVIEW_LIMIT)
}

function hiddenCount(groups: UserAvailableGroup[], key: string): number {
  if (isExpanded(key) || groups.length <= GROUP_PREVIEW_LIMIT) return 0
  return groups.length - GROUP_PREVIEW_LIMIT
}

function canToggleGroups(groups: UserAvailableGroup[]): boolean {
  return groups.length > GROUP_PREVIEW_LIMIT
}

function groupToggleLabel(groups: UserAvailableGroup[], key: string): string {
  if (isExpanded(key)) return t('availableChannels.showLessGroups')
  return t('availableChannels.showMoreGroups', { count: hiddenCount(groups, key) })
}

function toggleGroups(key: string) {
  expanded.value = { ...expanded.value, [key]: !expanded.value[key] }
}

function groupCount(channel: UserAvailableChannel): number {
  return channel.platforms.reduce((sum, section) => sum + section.groups.length, 0)
}

function modelCount(channel: UserAvailableChannel): number {
  const names = new Set<string>()
  for (const section of channel.platforms) {
    for (const model of section.supported_models) {
      if (model.name) names.add(model.name)
    }
  }
  return names.size
}

function hasPeakRate(group: UserAvailableGroup): boolean {
  return groupHasPeakRate(group)
}

function peakRateLabel(group: UserAvailableGroup): string {
  return formatPeakRateWindow(group, serverTimezoneLabel(appStore.cachedPublicSettings?.server_utc_offset))
}

function peakRateTitle(group: UserAvailableGroup): string {
  return t('common.peakRateTooltip', { window: peakRateLabel(group) }) + t('common.peakRateImageNote')
}

async function copyModel(name: string) {
  await copyToClipboard(name, t('availableChannels.copied'))
}
</script>
