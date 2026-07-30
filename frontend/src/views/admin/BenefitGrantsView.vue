<template>
  <AppLayout>
    <div class="mx-auto max-w-7xl">
      <div class="mb-5 border-b border-gray-200 dark:border-dark-700">
        <div class="flex gap-6">
          <button
            type="button"
            :class="tabClass('create')"
            @click="switchTab('create')"
          >
            <Icon name="gift" size="sm" />
            {{ t('admin.benefitGrants.tabs.create') }}
          </button>
          <button
            type="button"
            :class="tabClass('history')"
            @click="switchTab('history')"
          >
            <Icon name="clipboard" size="sm" />
            {{ t('admin.benefitGrants.tabs.history') }}
          </button>
        </div>
      </div>

      <section
        v-if="activeTab === 'create'"
        class="overflow-hidden border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900"
      >
        <div class="grid lg:grid-cols-[minmax(0,1.15fr)_minmax(340px,0.85fr)]">
          <form class="divide-y divide-gray-200 dark:divide-dark-700" @submit.prevent="loadPreview">
            <div class="p-5 sm:p-6">
              <div class="mb-4 flex items-center gap-2">
                <span class="flex h-7 w-7 items-center justify-center rounded-full bg-gray-100 text-xs font-semibold text-gray-700 dark:bg-dark-700 dark:text-gray-200">1</span>
                <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                  {{ t('admin.benefitGrants.audience.section') }}
                </h2>
              </div>

              <div class="grid gap-4 sm:grid-cols-2">
                <div>
                  <label class="input-label">{{ t('admin.benefitGrants.audience.type') }}</label>
                  <Select
                    :model-value="form.audience_type"
                    :options="audienceOptions"
                    @update:model-value="setAudienceType"
                  />
                </div>
                <div v-if="form.audience_type !== 'today_active'">
                  <label class="input-label">{{ t('admin.benefitGrants.audience.days') }}</label>
                  <div class="relative">
                    <input
                      v-model.number="form.audience_days"
                      type="number"
                      min="1"
                      max="365"
                      class="input pr-14"
                    />
                    <span class="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-sm text-gray-400">
                      {{ t('admin.benefitGrants.units.days') }}
                    </span>
                  </div>
                </div>
                <div v-else class="flex items-end">
                  <div class="w-full border-l-2 border-emerald-500 bg-emerald-50 px-3 py-2.5 text-sm text-emerald-800 dark:bg-emerald-950/30 dark:text-emerald-300">
                    {{ currentAudienceDate }} / {{ browserTimezone }}
                  </div>
                </div>
              </div>
            </div>

            <div class="p-5 sm:p-6">
              <div class="mb-4 flex items-center gap-2">
                <span class="flex h-7 w-7 items-center justify-center rounded-full bg-gray-100 text-xs font-semibold text-gray-700 dark:bg-dark-700 dark:text-gray-200">2</span>
                <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                  {{ t('admin.benefitGrants.benefit.section') }}
                </h2>
              </div>

              <div class="grid gap-4 sm:grid-cols-2">
                <div>
                  <label class="input-label">{{ t('admin.benefitGrants.benefit.type') }}</label>
                  <Select
                    :model-value="form.benefit_type"
                    :options="benefitOptions"
                    @update:model-value="setBenefitType"
                  />
                </div>

                <template v-if="form.benefit_type === 'subscription'">
                  <div>
                    <label class="input-label">{{ t('admin.benefitGrants.benefit.group') }}</label>
                    <Select
                      :model-value="form.group_id"
                      :options="subscriptionGroupOptions"
                      :placeholder="t('admin.benefitGrants.benefit.selectGroup')"
                      @update:model-value="setGroupID"
                    />
                  </div>
                  <div>
                    <label class="input-label">{{ t('admin.benefitGrants.benefit.validity') }}</label>
                    <div class="relative">
                      <input
                        v-model.number="form.validity_days"
                        type="number"
                        min="1"
                        max="36500"
                        class="input pr-14"
                      />
                      <span class="pointer-events-none absolute right-3 top-1/2 -translate-y-1/2 text-sm text-gray-400">
                        {{ t('admin.benefitGrants.units.days') }}
                      </span>
                    </div>
                  </div>
                  <div>
                    <label class="input-label">{{ t('admin.benefitGrants.benefit.conflictPolicy') }}</label>
                    <Select
                      :model-value="form.conflict_policy"
                      :options="conflictPolicyOptions"
                      @update:model-value="setConflictPolicy"
                    />
                  </div>
                </template>

                <template v-else>
                  <div>
                    <label class="input-label">{{ t('admin.benefitGrants.benefit.balanceAmount') }}</label>
                    <div class="relative">
                      <span class="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-gray-400">$</span>
                      <input
                        v-model.number="form.balance_amount"
                        type="number"
                        min="0.01"
                        max="1000000"
                        step="0.01"
                        class="input pl-7"
                      />
                    </div>
                    <p class="mt-1.5 text-xs text-gray-500 dark:text-gray-400">
                      {{ t('admin.benefitGrants.benefit.balanceHint') }}
                    </p>
                  </div>
                </template>
              </div>
            </div>

            <div class="p-5 sm:p-6">
              <div class="mb-4 flex items-center gap-2">
                <span class="flex h-7 w-7 items-center justify-center rounded-full bg-gray-100 text-xs font-semibold text-gray-700 dark:bg-dark-700 dark:text-gray-200">3</span>
                <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                  {{ t('admin.benefitGrants.notes.section') }}
                </h2>
              </div>
              <label class="input-label">{{ t('admin.benefitGrants.notes.label') }}</label>
              <textarea
                v-model="form.notes"
                maxlength="1800"
                rows="3"
                class="input min-h-[84px] resize-y"
                :placeholder="t('admin.benefitGrants.notes.placeholder')"
              ></textarea>
            </div>

            <div class="flex justify-end gap-3 bg-gray-50 px-5 py-4 dark:bg-dark-800/60 sm:px-6">
              <button type="button" class="btn btn-secondary" @click="resetForm">
                {{ t('common.reset') }}
              </button>
              <button type="submit" class="btn btn-primary" :disabled="!canPreview || previewLoading">
                <Icon
                  name="refresh"
                  size="sm"
                  :class="['mr-2', previewLoading ? 'animate-spin' : '']"
                />
                {{ previewLoading ? t('admin.benefitGrants.preview.loading') : t('admin.benefitGrants.preview.action') }}
              </button>
            </div>
          </form>

          <aside class="border-t border-gray-200 bg-gray-50/70 p-5 dark:border-dark-700 dark:bg-dark-950/40 sm:p-6 lg:border-l lg:border-t-0">
            <div class="sticky top-24">
              <div class="mb-4 flex items-center justify-between gap-3">
                <div>
                  <h2 class="text-base font-semibold text-gray-900 dark:text-white">
                    {{ t('admin.benefitGrants.preview.title') }}
                  </h2>
                  <p v-if="preview" class="mt-1 text-xs text-gray-500 dark:text-gray-400">
                    {{ formatDateTime(preview.window_start) }} / {{ formatDateTime(preview.window_end) }}
                  </p>
                </div>
                <button
                  v-if="preview"
                  type="button"
                  class="btn btn-secondary px-2"
                  :disabled="previewLoading"
                  :title="t('common.refresh')"
                  @click="loadPreview"
                >
                  <Icon name="refresh" size="sm" :class="previewLoading ? 'animate-spin' : ''" />
                </button>
              </div>

              <div v-if="previewLoading" class="flex min-h-[260px] items-center justify-center">
                <div class="text-center text-sm text-gray-500 dark:text-gray-400">
                  <Icon name="refresh" size="lg" class="mx-auto mb-3 animate-spin" />
                  {{ t('admin.benefitGrants.preview.loading') }}
                </div>
              </div>

              <div v-else-if="preview" class="space-y-5">
                <div class="grid grid-cols-2 gap-px overflow-hidden border border-gray-200 bg-gray-200 dark:border-dark-700 dark:bg-dark-700">
                  <div class="bg-white p-4 dark:bg-dark-900">
                    <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.benefitGrants.preview.matched') }}</p>
                    <p class="mt-1 text-2xl font-semibold tabular-nums text-gray-900 dark:text-white">{{ preview.matched_count }}</p>
                  </div>
                  <div class="bg-white p-4 dark:bg-dark-900">
                    <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.benefitGrants.preview.eligible') }}</p>
                    <p class="mt-1 text-2xl font-semibold tabular-nums text-emerald-600 dark:text-emerald-400">{{ preview.eligible_count }}</p>
                  </div>
                  <div class="bg-white p-4 dark:bg-dark-900">
                    <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.benefitGrants.preview.alreadyGranted') }}</p>
                    <p class="mt-1 text-xl font-semibold tabular-nums text-gray-700 dark:text-gray-300">{{ preview.already_granted_count }}</p>
                  </div>
                  <div class="bg-white p-4 dark:bg-dark-900">
                    <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.benefitGrants.preview.conflicts') }}</p>
                    <p :class="['mt-1 text-xl font-semibold tabular-nums', preview.conflict_count ? 'text-amber-600 dark:text-amber-400' : 'text-gray-700 dark:text-gray-300']">
                      {{ preview.conflict_count }}
                    </p>
                  </div>
                </div>

                <dl class="divide-y divide-gray-200 border-y border-gray-200 text-sm dark:divide-dark-700 dark:border-dark-700">
                  <div class="flex items-start justify-between gap-4 py-3">
                    <dt class="text-gray-500 dark:text-gray-400">{{ t('admin.benefitGrants.preview.audience') }}</dt>
                    <dd class="text-right font-medium text-gray-900 dark:text-white">{{ audienceLabel(preview) }}</dd>
                  </div>
                  <div class="flex items-start justify-between gap-4 py-3">
                    <dt class="text-gray-500 dark:text-gray-400">{{ t('admin.benefitGrants.preview.benefit') }}</dt>
                    <dd class="text-right font-medium text-gray-900 dark:text-white">{{ previewBenefitLabel(preview) }}</dd>
                  </div>
                  <div v-if="preview.benefit_type === 'balance'" class="flex items-start justify-between gap-4 py-3">
                    <dt class="text-gray-500 dark:text-gray-400">{{ t('admin.benefitGrants.preview.totalBalance') }}</dt>
                    <dd class="text-right font-semibold text-gray-900 dark:text-white">{{ previewTotalBalance(preview) }}</dd>
                  </div>
                  <div v-if="preview.benefit_type === 'subscription'" class="flex items-start justify-between gap-4 py-3">
                    <dt class="text-gray-500 dark:text-gray-400">{{ t('admin.benefitGrants.preview.policy') }}</dt>
                    <dd class="text-right font-medium text-gray-900 dark:text-white">{{ conflictPolicyLabel(preview.conflict_policy) }}</dd>
                  </div>
                </dl>

                <div
                  v-if="preview.conflict_count > 0"
                  class="border-l-2 border-amber-500 bg-amber-50 px-3 py-2.5 text-xs leading-5 text-amber-800 dark:bg-amber-950/30 dark:text-amber-300"
                >
                  {{ t('admin.benefitGrants.preview.conflictHint') }}
                </div>

                <button
                  type="button"
                  class="btn btn-primary w-full justify-center"
                  :disabled="preview.eligible_count <= 0 || executing"
                  @click="confirmVisible = true"
                >
                  <Icon name="gift" size="sm" class="mr-2" />
                  {{ t('admin.benefitGrants.execute.action', { count: preview.eligible_count }) }}
                </button>
              </div>

              <div
                v-else
                class="flex min-h-[260px] flex-col items-center justify-center border border-dashed border-gray-300 px-6 text-center dark:border-dark-600"
              >
                <Icon name="users" size="xl" class="mb-3 text-gray-300 dark:text-dark-600" />
                <p class="text-sm font-medium text-gray-600 dark:text-gray-300">
                  {{ previewError || t('admin.benefitGrants.preview.empty') }}
                </p>
              </div>
            </div>
          </aside>
        </div>
      </section>

      <section v-else class="space-y-4">
        <div class="flex items-center justify-end">
          <button
            type="button"
            class="btn btn-secondary"
            :disabled="historyLoading"
            :title="t('common.refresh')"
            @click="loadCampaigns"
          >
            <Icon name="refresh" size="sm" :class="historyLoading ? 'animate-spin' : ''" />
          </button>
        </div>

        <div class="overflow-hidden border border-gray-200 bg-white shadow-sm dark:border-dark-700 dark:bg-dark-900">
          <DataTable :columns="campaignColumns" :data="campaigns" :loading="historyLoading" row-key="id">
            <template #cell-created_at="{ value }">
              <span class="whitespace-nowrap text-sm text-gray-600 dark:text-gray-300">{{ formatDateTime(value) }}</span>
            </template>
            <template #cell-audience="{ row }">
              <div class="min-w-0">
                <p class="font-medium text-gray-900 dark:text-white">{{ audienceLabel(row) }}</p>
                <p class="mt-0.5 text-xs text-gray-400">{{ row.audience_date }} / {{ row.timezone }}</p>
              </div>
            </template>
            <template #cell-benefit="{ row }">
              <div class="min-w-0">
                <p class="font-medium text-gray-900 dark:text-white">{{ campaignBenefitLabel(row) }}</p>
                <p v-if="row.notes" class="mt-0.5 max-w-[260px] truncate text-xs text-gray-400" :title="row.notes">{{ row.notes }}</p>
              </div>
            </template>
            <template #cell-result="{ row }">
              <div class="whitespace-nowrap text-sm">
                <span class="font-semibold text-emerald-600 dark:text-emerald-400">{{ row.granted_count }}</span>
                <span class="text-gray-400"> / {{ row.eligible_count }}</span>
                <span v-if="row.failed_count" class="ml-2 text-red-600 dark:text-red-400">
                  {{ t('admin.benefitGrants.history.failedCount', { count: row.failed_count }) }}
                </span>
              </div>
            </template>
            <template #cell-status="{ row }">
              <span :class="campaignStatusClass(row.status)">
                {{ campaignStatusLabel(row.status) }}
              </span>
            </template>
            <template #cell-actions="{ row }">
              <div class="flex items-center justify-end gap-2">
                <button
                  type="button"
                  class="btn btn-secondary px-2"
                  :title="t('admin.benefitGrants.history.detail')"
                  @click="openCampaignDetail(row)"
                >
                  <Icon name="eye" size="sm" />
                </button>
                <button
                  v-if="campaignCanRetry(row)"
                  type="button"
                  class="btn btn-secondary px-2"
                  :disabled="retryingCampaignID === row.id"
                  :title="t('admin.benefitGrants.history.retry')"
                  @click="retryCampaign(row)"
                >
                  <Icon name="refresh" size="sm" :class="retryingCampaignID === row.id ? 'animate-spin' : ''" />
                </button>
              </div>
            </template>
            <template #empty>
              <div class="flex flex-col items-center py-12">
                <Icon name="clipboard" size="xl" class="mb-3 text-gray-300 dark:text-dark-600" />
                <p class="text-sm font-medium text-gray-500 dark:text-gray-400">{{ t('admin.benefitGrants.history.empty') }}</p>
              </div>
            </template>
          </DataTable>
        </div>

        <Pagination
          v-if="historyTotal > 0"
          :total="historyTotal"
          :page="historyPage"
          :page-size="historyPageSize"
          @update:page="handleHistoryPage"
          @update:pageSize="handleHistoryPageSize"
        />
      </section>
    </div>

    <BaseDialog
      :show="confirmVisible"
      :title="t('admin.benefitGrants.confirm.title')"
      width="normal"
      @close="confirmVisible = false"
    >
      <div v-if="preview" class="space-y-4">
        <div class="border-l-2 border-amber-500 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:bg-amber-950/30 dark:text-amber-200">
          {{ t('admin.benefitGrants.confirm.message', { count: preview.eligible_count }) }}
        </div>
        <dl class="divide-y divide-gray-200 border-y border-gray-200 text-sm dark:divide-dark-700 dark:border-dark-700">
          <div class="flex justify-between gap-4 py-3">
            <dt class="text-gray-500 dark:text-gray-400">{{ t('admin.benefitGrants.preview.audience') }}</dt>
            <dd class="text-right font-medium text-gray-900 dark:text-white">{{ audienceLabel(preview) }}</dd>
          </div>
          <div class="flex justify-between gap-4 py-3">
            <dt class="text-gray-500 dark:text-gray-400">{{ t('admin.benefitGrants.preview.benefit') }}</dt>
            <dd class="text-right font-medium text-gray-900 dark:text-white">{{ previewBenefitLabel(preview) }}</dd>
          </div>
          <div v-if="preview.benefit_type === 'balance'" class="flex justify-between gap-4 py-3">
            <dt class="text-gray-500 dark:text-gray-400">{{ t('admin.benefitGrants.preview.totalBalance') }}</dt>
            <dd class="text-right font-semibold text-gray-900 dark:text-white">{{ previewTotalBalance(preview) }}</dd>
          </div>
        </dl>
      </div>
      <template #footer>
        <button type="button" class="btn btn-secondary" :disabled="executing" @click="confirmVisible = false">
          {{ t('common.cancel') }}
        </button>
        <button type="button" class="btn btn-primary" :disabled="executing" @click="executeGrant">
          <Icon name="gift" size="sm" :class="['mr-2', executing ? 'animate-pulse' : '']" />
          {{ executing ? t('admin.benefitGrants.execute.running') : t('admin.benefitGrants.confirm.submit') }}
        </button>
      </template>
    </BaseDialog>

    <BaseDialog
      :show="detailVisible"
      :title="t('admin.benefitGrants.detail.title')"
      width="wide"
      @close="closeCampaignDetail"
    >
      <div v-if="selectedCampaign" class="space-y-5">
        <div class="grid grid-cols-2 gap-px overflow-hidden border border-gray-200 bg-gray-200 sm:grid-cols-4 dark:border-dark-700 dark:bg-dark-700">
          <div class="bg-gray-50 p-3 dark:bg-dark-900">
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.benefitGrants.preview.matched') }}</p>
            <p class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ selectedCampaign.matched_count }}</p>
          </div>
          <div class="bg-gray-50 p-3 dark:bg-dark-900">
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.benefitGrants.preview.eligible') }}</p>
            <p class="mt-1 text-lg font-semibold text-gray-900 dark:text-white">{{ selectedCampaign.eligible_count }}</p>
          </div>
          <div class="bg-gray-50 p-3 dark:bg-dark-900">
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.benefitGrants.detail.granted') }}</p>
            <p class="mt-1 text-lg font-semibold text-emerald-600 dark:text-emerald-400">{{ selectedCampaign.granted_count }}</p>
          </div>
          <div class="bg-gray-50 p-3 dark:bg-dark-900">
            <p class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.benefitGrants.detail.failed') }}</p>
            <p class="mt-1 text-lg font-semibold text-red-600 dark:text-red-400">{{ selectedCampaign.failed_count }}</p>
          </div>
        </div>

        <dl class="grid gap-x-6 gap-y-3 border-y border-gray-200 py-4 text-sm sm:grid-cols-2 dark:border-dark-700">
          <div class="flex items-start justify-between gap-4">
            <dt class="text-gray-500 dark:text-gray-400">{{ t('admin.benefitGrants.preview.audience') }}</dt>
            <dd class="text-right font-medium text-gray-900 dark:text-white">{{ audienceLabel(selectedCampaign) }}</dd>
          </div>
          <div class="flex items-start justify-between gap-4">
            <dt class="text-gray-500 dark:text-gray-400">{{ t('admin.benefitGrants.preview.benefit') }}</dt>
            <dd class="text-right font-medium text-gray-900 dark:text-white">{{ campaignBenefitLabel(selectedCampaign) }}</dd>
          </div>
          <div v-if="selectedCampaign.benefit_type === 'subscription'" class="flex items-start justify-between gap-4">
            <dt class="text-gray-500 dark:text-gray-400">{{ t('admin.benefitGrants.preview.policy') }}</dt>
            <dd class="text-right font-medium text-gray-900 dark:text-white">{{ conflictPolicyLabel(selectedCampaign.conflict_policy) }}</dd>
          </div>
          <div class="flex items-start justify-between gap-4">
            <dt class="text-gray-500 dark:text-gray-400">{{ t('admin.benefitGrants.detail.window') }}</dt>
            <dd class="text-right font-medium text-gray-900 dark:text-white">
              {{ formatDateTime(selectedCampaign.window_start) }} / {{ formatDateTime(selectedCampaign.window_end) }}
            </dd>
          </div>
          <div v-if="selectedCampaign.notes" class="flex items-start justify-between gap-4 sm:col-span-2">
            <dt class="text-gray-500 dark:text-gray-400">{{ t('admin.benefitGrants.notes.label') }}</dt>
            <dd class="max-w-2xl whitespace-pre-wrap text-right text-gray-900 dark:text-white">{{ selectedCampaign.notes }}</dd>
          </div>
        </dl>

        <div class="flex flex-wrap items-end justify-between gap-3">
          <div class="w-full sm:w-48">
            <label class="input-label">{{ t('admin.benefitGrants.detail.statusFilter') }}</label>
            <Select
              :model-value="recipientStatusFilter"
              :options="recipientStatusOptions"
              @update:model-value="setRecipientStatusFilter"
              @change="handleRecipientFilter"
            />
          </div>
          <button
            v-if="campaignCanRetry(selectedCampaign)"
            type="button"
            class="btn btn-secondary"
            :disabled="retryingCampaignID === selectedCampaign.id"
            @click="retryCampaign(selectedCampaign)"
          >
            <Icon name="refresh" size="sm" :class="['mr-2', retryingCampaignID === selectedCampaign.id ? 'animate-spin' : '']" />
            {{ t('admin.benefitGrants.history.retry') }}
          </button>
        </div>

        <div class="overflow-hidden border border-gray-200 dark:border-dark-700">
          <DataTable :columns="recipientColumns" :data="recipients" :loading="recipientsLoading" row-key="id">
            <template #cell-user="{ row }">
              <div class="min-w-0 max-w-[260px]">
                <p class="truncate font-medium text-gray-900 dark:text-white" :title="row.email">{{ row.email }}</p>
                <p class="mt-0.5 text-xs text-gray-400">#{{ row.user_id }}<span v-if="row.username"> / {{ row.username }}</span></p>
              </div>
            </template>
            <template #cell-status="{ row }">
              <span :class="recipientStatusClass(row.status)">{{ recipientStatusLabel(row.status) }}</span>
            </template>
            <template #cell-result="{ row }">
              <span class="text-sm text-gray-600 dark:text-gray-300">{{ recipientResultLabel(row) }}</span>
            </template>
            <template #cell-attempt_count="{ value }">
              <span class="tabular-nums text-gray-600 dark:text-gray-300">{{ value }}</span>
            </template>
            <template #cell-error="{ value }">
              <span class="block max-w-[320px] truncate text-xs text-red-600 dark:text-red-400" :title="value">{{ value || '-' }}</span>
            </template>
            <template #empty>
              <div class="py-10 text-center text-sm text-gray-500 dark:text-gray-400">{{ t('admin.benefitGrants.detail.empty') }}</div>
            </template>
          </DataTable>
        </div>

        <Pagination
          v-if="recipientsTotal > 0"
          :total="recipientsTotal"
          :page="recipientsPage"
          :page-size="recipientsPageSize"
          @update:page="handleRecipientPage"
          @update:pageSize="handleRecipientPageSize"
        />
      </div>
    </BaseDialog>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { adminAPI } from '@/api/admin'
import type {
  BenefitGrantAudienceType,
  BenefitGrantCampaign,
  BenefitGrantCampaignStatus,
  BenefitGrantConflictPolicy,
  BenefitGrantPreview,
  BenefitGrantRecipient,
  BenefitGrantRecipientStatus,
  BenefitGrantRequest,
  BenefitGrantType
} from '@/api/admin'
import type { Group } from '@/types'
import type { Column } from '@/components/common/types'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import DataTable from '@/components/common/DataTable.vue'
import Pagination from '@/components/common/Pagination.vue'
import Select from '@/components/common/Select.vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores'
import { formatCurrency, formatDateTime } from '@/utils/format'

type Tab = 'create' | 'history'
type SelectValue = string | number | boolean | null

const { t } = useI18n()
const appStore = useAppStore()
const activeTab = ref<Tab>('create')
const groups = ref<Group[]>([])
const preview = ref<BenefitGrantPreview | null>(null)
const previewLoading = ref(false)
const previewError = ref('')
const executing = ref(false)
const confirmVisible = ref(false)
let previewSequence = 0

const browserTimezone = getBrowserTimezone()
const currentAudienceDate = ref(getDateInTimezone(browserTimezone))

const form = reactive({
  operation_key: createOperationKey('benefit-grant'),
  audience_type: 'today_active' as BenefitGrantAudienceType,
  audience_days: 7,
  benefit_type: 'subscription' as BenefitGrantType,
  conflict_policy: 'skip_active' as BenefitGrantConflictPolicy,
  group_id: null as number | null,
  validity_days: 1,
  balance_amount: 1,
  notes: ''
})

const campaigns = ref<BenefitGrantCampaign[]>([])
const historyLoading = ref(false)
const historyPage = ref(1)
const historyPageSize = ref(20)
const historyTotal = ref(0)
const retryingCampaignID = ref<number | null>(null)

const detailVisible = ref(false)
const selectedCampaign = ref<BenefitGrantCampaign | null>(null)
const recipients = ref<BenefitGrantRecipient[]>([])
const recipientsLoading = ref(false)
const recipientsPage = ref(1)
const recipientsPageSize = ref(50)
const recipientsTotal = ref(0)
const recipientStatusFilter = ref<BenefitGrantRecipientStatus | ''>('')

const audienceOptions = computed<Array<{ value: BenefitGrantAudienceType; label: string }>>(() => [
  { value: 'today_active', label: t('admin.benefitGrants.audience.todayActive') },
  { value: 'recent_active', label: t('admin.benefitGrants.audience.recentActive') },
  { value: 'recent_registered', label: t('admin.benefitGrants.audience.recentRegistered') }
])

const benefitOptions = computed<Array<{ value: BenefitGrantType; label: string }>>(() => [
  { value: 'subscription', label: t('admin.benefitGrants.benefit.subscription') },
  { value: 'balance', label: t('admin.benefitGrants.benefit.balance') }
])

const conflictPolicyOptions = computed<Array<{
  value: BenefitGrantConflictPolicy
  label: string
}>>(() => [
  { value: 'skip_active', label: t('admin.benefitGrants.policy.skipActive') },
  { value: 'extend_active', label: t('admin.benefitGrants.policy.extendActive') }
])

const subscriptionGroupOptions = computed(() =>
  groups.value
    .filter(group => group.subscription_type === 'subscription' && group.status === 'active')
    .map(group => ({ value: group.id, label: group.name }))
)

const recipientStatusOptions = computed<Array<{
  value: BenefitGrantRecipientStatus | ''
  label: string
}>>(() => [
  { value: '', label: t('admin.benefitGrants.detail.allStatuses') },
  { value: 'granted', label: t('admin.benefitGrants.recipientStatus.granted') },
  { value: 'failed', label: t('admin.benefitGrants.recipientStatus.failed') },
  { value: 'skipped', label: t('admin.benefitGrants.recipientStatus.skipped') },
  { value: 'pending', label: t('admin.benefitGrants.recipientStatus.pending') },
  { value: 'processing', label: t('admin.benefitGrants.recipientStatus.processing') }
])

const canPreview = computed(() => {
  const days = form.audience_type === 'today_active' ? 1 : Number(form.audience_days)
  if (!Number.isInteger(days) || days < 1 || days > 365) return false
  if (form.benefit_type === 'subscription') {
    return Boolean(
      form.group_id &&
      Number.isInteger(Number(form.validity_days)) &&
      form.validity_days >= 1 &&
      form.validity_days <= 36500
    )
  }
  return Number(form.balance_amount) > 0 && Number(form.balance_amount) <= 1000000
})

const campaignColumns = computed<Column[]>(() => [
  { key: 'created_at', label: t('admin.benefitGrants.history.createdAt'), sortable: false },
  { key: 'audience', label: t('admin.benefitGrants.history.audience'), sortable: false },
  { key: 'benefit', label: t('admin.benefitGrants.history.benefit'), sortable: false },
  { key: 'result', label: t('admin.benefitGrants.history.result'), sortable: false },
  { key: 'status', label: t('admin.benefitGrants.history.status'), sortable: false },
  { key: 'actions', label: t('admin.benefitGrants.history.actions'), sortable: false, class: 'text-right' }
])

const recipientColumns = computed<Column[]>(() => [
  { key: 'user', label: t('admin.benefitGrants.detail.user'), sortable: false },
  { key: 'status', label: t('admin.benefitGrants.history.status'), sortable: false },
  { key: 'result', label: t('admin.benefitGrants.detail.result'), sortable: false },
  { key: 'attempt_count', label: t('admin.benefitGrants.detail.attempts'), sortable: false },
  { key: 'error', label: t('admin.benefitGrants.detail.error'), sortable: false }
])

watch(
  [
    () => form.audience_type,
    () => form.audience_days,
    () => form.benefit_type,
    () => form.conflict_policy,
    () => form.group_id,
    () => form.validity_days,
    () => form.balance_amount,
    () => form.notes
  ],
  () => {
    invalidatePreview()
  }
)

function invalidatePreview() {
  previewSequence++
  preview.value = null
  previewError.value = ''
  previewLoading.value = false
}

function setAudienceType(value: SelectValue) {
  if (
    value === 'today_active' ||
    value === 'recent_active' ||
    value === 'recent_registered'
  ) {
    form.audience_type = value
  }
}

function setBenefitType(value: SelectValue) {
  if (value === 'subscription' || value === 'balance') {
    form.benefit_type = value
  }
}

function setConflictPolicy(value: SelectValue) {
  if (value === 'skip_active' || value === 'extend_active') {
    form.conflict_policy = value
  }
}

function setGroupID(value: SelectValue) {
  const groupID = Number(value)
  form.group_id = Number.isInteger(groupID) && groupID > 0 ? groupID : null
}

function setRecipientStatusFilter(value: SelectValue) {
  if (
    value === '' ||
    value === 'pending' ||
    value === 'processing' ||
    value === 'granted' ||
    value === 'skipped' ||
    value === 'failed'
  ) {
    recipientStatusFilter.value = value
  }
}

function syncCurrentAudienceDate(): boolean {
  const nextDate = getDateInTimezone(browserTimezone)
  if (nextDate === currentAudienceDate.value) return false
  currentAudienceDate.value = nextDate
  invalidatePreview()
  return true
}

function switchTab(tab: Tab) {
  activeTab.value = tab
  if (tab === 'history') loadCampaigns()
}

function tabClass(tab: Tab): string[] {
  return [
    'inline-flex items-center gap-2 border-b-2 px-1 pb-3 text-sm font-medium transition-colors',
    activeTab.value === tab
      ? 'border-primary-500 text-primary-600 dark:text-primary-400'
      : 'border-transparent text-gray-500 hover:text-gray-800 dark:text-gray-400 dark:hover:text-gray-200'
  ]
}

function buildRequest(): BenefitGrantRequest | null {
  if (!canPreview.value) return null
  const request: BenefitGrantRequest = {
    operation_key: form.operation_key,
    audience_type: form.audience_type,
    audience_date: currentAudienceDate.value,
    audience_days: form.audience_type === 'today_active' ? 1 : Number(form.audience_days),
    timezone: browserTimezone,
    benefit_type: form.benefit_type,
    notes: form.notes.trim() || undefined
  }
  if (form.benefit_type === 'subscription') {
    request.conflict_policy = form.conflict_policy
    request.group_id = Number(form.group_id)
    request.validity_days = Number(form.validity_days)
  } else {
    request.conflict_policy = 'none'
    request.balance_amount = Number(form.balance_amount)
  }
  return request
}

async function loadPreview() {
  syncCurrentAudienceDate()
  const request = buildRequest()
  if (!request) return
  const sequence = ++previewSequence
  previewLoading.value = true
  previewError.value = ''
  try {
    const result = await adminAPI.benefitGrants.preview(request)
    if (sequence !== previewSequence) return
    preview.value = result
  } catch (error: any) {
    if (sequence !== previewSequence) return
    preview.value = null
    previewError.value = error?.message || t('admin.benefitGrants.preview.failed')
  } finally {
    if (sequence === previewSequence) {
      previewLoading.value = false
    }
  }
}

async function executeGrant() {
  if (syncCurrentAudienceDate()) {
    confirmVisible.value = false
    appStore.showError(t('admin.benefitGrants.execute.audienceChanged'))
    await loadPreview()
    return
  }
  const request = buildRequest()
  if (!request || !preview.value) return
  executing.value = true
  try {
    const result = await adminAPI.benefitGrants.execute({
      ...request,
      expected_matched_count: preview.value.matched_count,
      expected_eligible_count: preview.value.eligible_count,
      expected_snapshot: preview.value.snapshot_token
    })
    if (result.failed_count > 0) {
      appStore.showError(t('admin.benefitGrants.execute.partial', {
        granted: result.granted_count,
        failed: result.failed_count
      }))
    } else {
      appStore.showSuccess(t('admin.benefitGrants.execute.success', {
        granted: result.granted_count,
        skipped: result.skipped_count
      }))
    }
    confirmVisible.value = false
    resetForm()
    activeTab.value = 'history'
    historyPage.value = 1
    await loadCampaigns()
  } catch (error: any) {
    const reason = error?.reason || error?.code
    if (reason === 'BENEFIT_GRANT_AUDIENCE_CHANGED') {
      confirmVisible.value = false
      appStore.showError(t('admin.benefitGrants.execute.audienceChanged'))
      await loadPreview()
    } else {
      appStore.showError(error?.message || t('admin.benefitGrants.execute.failed'))
    }
  } finally {
    executing.value = false
  }
}

function resetForm() {
  form.operation_key = createOperationKey('benefit-grant')
  form.audience_type = 'today_active'
  form.audience_days = 7
  form.benefit_type = 'subscription'
  form.conflict_policy = 'skip_active'
  form.group_id = subscriptionGroupOptions.value.length === 1
    ? Number(subscriptionGroupOptions.value[0].value)
    : null
  form.validity_days = 1
  form.balance_amount = 1
  form.notes = ''
  invalidatePreview()
}

async function loadGroups() {
  try {
    groups.value = await adminAPI.groups.getAll()
    if (!form.group_id && subscriptionGroupOptions.value.length === 1) {
      form.group_id = Number(subscriptionGroupOptions.value[0].value)
    }
  } catch {
    appStore.showError(t('admin.benefitGrants.benefit.groupsFailed'))
  }
}

async function loadCampaigns() {
  historyLoading.value = true
  try {
    const response = await adminAPI.benefitGrants.list(historyPage.value, historyPageSize.value)
    campaigns.value = response.items
    historyTotal.value = response.total
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.benefitGrants.history.loadFailed'))
  } finally {
    historyLoading.value = false
  }
}

function handleHistoryPage(page: number) {
  historyPage.value = page
  loadCampaigns()
}

function handleHistoryPageSize(pageSize: number) {
  historyPageSize.value = pageSize
  historyPage.value = 1
  loadCampaigns()
}

async function openCampaignDetail(campaign: BenefitGrantCampaign) {
  selectedCampaign.value = campaign
  recipientsPage.value = 1
  recipientStatusFilter.value = ''
  detailVisible.value = true
  const [freshCampaign] = await Promise.all([
    adminAPI.benefitGrants.get(campaign.id).catch((error: any) => {
      appStore.showError(error?.message || t('admin.benefitGrants.detail.loadFailed'))
      return null
    }),
    loadRecipients()
  ])
  if (freshCampaign && selectedCampaign.value?.id === campaign.id) {
    selectedCampaign.value = freshCampaign
  }
}

function closeCampaignDetail() {
  detailVisible.value = false
  selectedCampaign.value = null
  recipients.value = []
  recipientsTotal.value = 0
}

async function loadRecipients() {
  if (!selectedCampaign.value) return
  recipientsLoading.value = true
  try {
    const response = await adminAPI.benefitGrants.listRecipients(
      selectedCampaign.value.id,
      recipientsPage.value,
      recipientsPageSize.value,
      recipientStatusFilter.value || undefined
    )
    recipients.value = response.items
    recipientsTotal.value = response.total
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.benefitGrants.detail.loadFailed'))
  } finally {
    recipientsLoading.value = false
  }
}

function handleRecipientFilter() {
  recipientsPage.value = 1
  loadRecipients()
}

function handleRecipientPage(page: number) {
  recipientsPage.value = page
  loadRecipients()
}

function handleRecipientPageSize(pageSize: number) {
  recipientsPageSize.value = pageSize
  recipientsPage.value = 1
  loadRecipients()
}

async function retryCampaign(campaign: BenefitGrantCampaign) {
  if (!window.confirm(t('admin.benefitGrants.history.retryConfirm'))) return
  retryingCampaignID.value = campaign.id
  try {
    const result = await adminAPI.benefitGrants.retry(
      campaign.id,
      createOperationKey(`benefit-grant-retry-${campaign.id}`)
    )
    appStore.showSuccess(t('admin.benefitGrants.history.retryDone', {
      granted: result.granted_count,
      failed: result.failed_count
    }))
    selectedCampaign.value = selectedCampaign.value?.id === campaign.id
      ? result.campaign
      : selectedCampaign.value
    await Promise.all([
      loadCampaigns(),
      selectedCampaign.value?.id === campaign.id ? loadRecipients() : Promise.resolve()
    ])
  } catch (error: any) {
    appStore.showError(error?.message || t('admin.benefitGrants.history.retryFailed'))
  } finally {
    retryingCampaignID.value = null
  }
}

function audienceLabel(item: Pick<BenefitGrantPreview | BenefitGrantCampaign, 'audience_type' | 'audience_days'>): string {
  if (item.audience_type === 'today_active') return t('admin.benefitGrants.audience.todayActive')
  if (item.audience_type === 'recent_active') {
    return t('admin.benefitGrants.audience.recentActiveDays', { days: item.audience_days })
  }
  return t('admin.benefitGrants.audience.recentRegisteredDays', { days: item.audience_days })
}

function previewBenefitLabel(item: BenefitGrantPreview): string {
  if (item.benefit_type === 'balance') {
    return t('admin.benefitGrants.benefit.balanceValue', { amount: formatCurrency(item.balance_amount) })
  }
  const group = groups.value.find(candidate => candidate.id === item.group_id)
  return t('admin.benefitGrants.benefit.subscriptionValue', {
    group: group?.name || `#${item.group_id}`,
    days: item.validity_days
  })
}

function previewTotalBalance(item: BenefitGrantPreview): string {
  return formatCurrency(item.balance_amount * item.eligible_count)
}

function campaignBenefitLabel(item: BenefitGrantCampaign): string {
  if (item.benefit_type === 'balance') {
    return t('admin.benefitGrants.benefit.balanceValue', {
      amount: formatCurrency(item.balance_amount || 0)
    })
  }
  return t('admin.benefitGrants.benefit.subscriptionValue', {
    group: item.group_name || `#${item.group_id || 0}`,
    days: item.validity_days || 0
  })
}

function conflictPolicyLabel(policy: BenefitGrantConflictPolicy): string {
  return policy === 'extend_active'
    ? t('admin.benefitGrants.policy.extendActive')
    : t('admin.benefitGrants.policy.skipActive')
}

function campaignStatusLabel(status: BenefitGrantCampaignStatus): string {
  return t(`admin.benefitGrants.campaignStatus.${status}`)
}

function campaignStatusClass(status: BenefitGrantCampaignStatus): string[] {
  const classes: Record<BenefitGrantCampaignStatus, string> = {
    running: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300',
    completed: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300',
    partial: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300',
    failed: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  }
  return ['inline-flex rounded px-2 py-1 text-xs font-medium', classes[status]]
}

function recipientStatusLabel(status: BenefitGrantRecipientStatus): string {
  return t(`admin.benefitGrants.recipientStatus.${status}`)
}

function recipientStatusClass(status: BenefitGrantRecipientStatus): string[] {
  const classes: Record<BenefitGrantRecipientStatus, string> = {
    pending: 'bg-gray-100 text-gray-700 dark:bg-dark-700 dark:text-gray-300',
    processing: 'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-300',
    granted: 'bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-300',
    skipped: 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300',
    failed: 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-300'
  }
  return ['inline-flex rounded px-2 py-1 text-xs font-medium', classes[status]]
}

function recipientResultLabel(recipient: BenefitGrantRecipient): string {
  if (recipient.result_type === 'balance_added' && recipient.balance_before != null && recipient.balance_after != null) {
    return `${formatCurrency(recipient.balance_before)} -> ${formatCurrency(recipient.balance_after)}`
  }
  if (recipient.result_type) {
    return t(`admin.benefitGrants.resultType.${recipient.result_type}`)
  }
  return '-'
}

function campaignCanRetry(campaign: BenefitGrantCampaign): boolean {
  return campaign.failed_count > 0 || campaign.status === 'running'
}

function createOperationKey(prefix: string): string {
  const suffix = globalThis.crypto?.randomUUID?.()
    ?? `${Date.now()}-${Math.random().toString(36).slice(2)}`
  return `${prefix}-${suffix}`
}

function getBrowserTimezone(): string {
  try {
    return Intl.DateTimeFormat().resolvedOptions().timeZone || 'Asia/Shanghai'
  } catch {
    return 'Asia/Shanghai'
  }
}

function getDateInTimezone(timezone: string): string {
  try {
    const parts = new Intl.DateTimeFormat('en-US', {
      timeZone: timezone,
      year: 'numeric',
      month: '2-digit',
      day: '2-digit'
    }).formatToParts(new Date())
    const values = Object.fromEntries(parts.map(part => [part.type, part.value]))
    return `${values.year}-${values.month}-${values.day}`
  } catch {
    const now = new Date()
    const year = now.getFullYear()
    const month = String(now.getMonth() + 1).padStart(2, '0')
    const day = String(now.getDate()).padStart(2, '0')
    return `${year}-${month}-${day}`
  }
}

onMounted(() => {
  loadGroups()
  loadCampaigns()
})
</script>
