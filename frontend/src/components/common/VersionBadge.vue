<template>
  <div class="relative">
    <!-- Admin: Full version badge with dropdown -->
    <template v-if="isAdmin">
      <button
        @click="toggleDropdown"
        class="flex items-center gap-1.5 rounded-lg px-2 py-1 text-xs transition-colors"
        :class="[
          hasUpdate
            ? 'bg-amber-100 text-amber-700 hover:bg-amber-200 dark:bg-amber-900/30 dark:text-amber-400 dark:hover:bg-amber-900/50'
            : 'bg-gray-100 text-gray-600 hover:bg-gray-200 dark:bg-dark-800 dark:text-dark-400 dark:hover:bg-dark-700'
        ]"
        :title="hasUpdate ? t('version.updateAvailable') : t('version.upToDate')"
      >
        <span v-if="currentVersion" class="font-medium">v{{ currentVersion }}</span>
        <span
          v-else
          class="h-3 w-12 animate-pulse rounded bg-gray-200 font-medium dark:bg-dark-600"
        ></span>
        <!-- Update indicator -->
        <span v-if="hasUpdate" class="relative flex h-2 w-2">
          <span
            class="absolute inline-flex h-full w-full animate-ping rounded-full bg-amber-400 opacity-75"
          ></span>
          <span class="relative inline-flex h-2 w-2 rounded-full bg-amber-500"></span>
        </span>
      </button>

      <!-- Dropdown -->
      <transition name="dropdown">
        <div
          v-if="dropdownOpen"
          ref="dropdownRef"
          class="absolute left-0 z-50 mt-2 max-h-[calc(100vh-5rem)] w-80 max-w-[calc(100vw-2rem)] overflow-x-hidden overflow-y-auto whitespace-normal rounded-xl border border-gray-200 bg-white shadow-lg transition-all duration-200 dark:border-dark-700 dark:bg-dark-800"
        >
          <!-- Header with refresh button -->
          <div
            class="flex items-center justify-between border-b border-gray-100 px-4 py-3 dark:border-dark-700"
          >
            <span class="text-sm font-medium text-gray-700 dark:text-dark-300">{{
              t('version.currentVersion')
            }}</span>
            <button
              @click="refreshVersion(true)"
              class="rounded-lg p-1.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600 dark:hover:bg-dark-700 dark:hover:text-dark-200"
              :disabled="loading"
              :title="t('version.refresh')"
            >
              <Icon
                name="refresh"
                size="sm"
                :stroke-width="2"
                :class="{ 'animate-spin': loading }"
              />
            </button>
          </div>

          <div class="p-4">
            <!-- Loading state -->
            <div v-if="loading" class="flex items-center justify-center py-6">
              <svg class="h-6 w-6 animate-spin text-primary-500" fill="none" viewBox="0 0 24 24">
                <circle
                  class="opacity-25"
                  cx="12"
                  cy="12"
                  r="10"
                  stroke="currentColor"
                  stroke-width="4"
                ></circle>
                <path
                  class="opacity-75"
                  fill="currentColor"
                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                ></path>
              </svg>
            </div>

            <!-- Content -->
            <template v-else>
              <!-- Version display - centered and prominent -->
              <div class="mb-4 text-center">
                <div class="inline-flex items-center gap-2">
                  <span
                    v-if="currentVersion"
                    class="text-2xl font-bold text-gray-900 dark:text-white"
                    >v{{ currentVersion }}</span
                  >
                  <span v-else class="text-2xl font-bold text-gray-400 dark:text-dark-500">--</span>
                  <!-- Show check mark when up to date -->
                  <span
                    v-if="!hasUpdate"
                    class="flex h-5 w-5 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/30"
                  >
                    <svg
                      class="h-3 w-3 text-green-600 dark:text-green-400"
                      fill="currentColor"
                      viewBox="0 0 20 20"
                    >
                      <path
                        fill-rule="evenodd"
                        d="M16.707 5.293a1 1 0 010 1.414l-8 8a1 1 0 01-1.414 0l-4-4a1 1 0 011.414-1.414L8 12.586l7.293-7.293a1 1 0 011.414 0z"
                        clip-rule="evenodd"
                      />
                    </svg>
                  </span>
                </div>
                <p class="mt-1 text-xs text-gray-500 dark:text-dark-400">
                  {{
                    hasUpdate
                      ? t('version.latestVersion') + ': v' + latestVersion
                      : t('version.upToDate')
                  }}
                </p>
              </div>

              <div class="-mx-4 mb-3 px-4">
                <OfficialVersionStatus
                  :current-version="currentVersion"
                  :latest-version="latestVersion"
                  :has-update="hasUpdate"
                  :release-url="releaseInfo?.html_url"
                />
              </div>

              <div
                v-if="customUpdateAvailable"
                class="-mx-4 mb-3 border-y border-gray-100 px-4 py-3 dark:border-dark-700"
              >
                <div class="mb-2 flex items-center justify-between gap-2">
                  <span
                    class="flex min-w-0 items-center gap-2 text-sm font-medium text-gray-700 dark:text-dark-200"
                  >
                    <Icon name="sync" size="sm" :stroke-width="2" />
                    <span>{{ t('version.customUpdateTitle') }}</span>
                  </span>
                  <span class="flex flex-shrink-0 items-center gap-2">
                    <button
                      type="button"
                      class="flex h-7 items-center gap-1 rounded-lg px-2 text-[10px] font-medium transition-colors disabled:cursor-not-allowed disabled:opacity-50"
                      :class="
                        resolverSettingsVisible
                          ? 'bg-primary-100 text-primary-700 dark:bg-primary-900/30 dark:text-primary-300'
                          : 'text-gray-500 hover:bg-gray-100 hover:text-gray-800 dark:text-dark-400 dark:hover:bg-dark-700 dark:hover:text-dark-100'
                      "
                      :disabled="customUpdateDemoActive"
                      :title="t('version.customUpdateResolverSettings')"
                      :aria-label="t('version.customUpdateResolverSettings')"
                      :aria-expanded="resolverSettingsVisible"
                      @click="resolverSettingsVisible = !resolverSettingsVisible"
                    >
                      <Icon name="cog" size="xs" :stroke-width="2" />
                      <span>{{ t('version.customUpdateResolverSettings') }}</span>
                    </button>
                    <span
                      class="h-2 w-2 rounded-full"
                      :class="customUpdateStatusDotClass"
                    ></span>
                  </span>
                </div>

                <div
                  v-if="customUpdateDemoActive"
                  data-testid="custom-update-demo-banner"
                  class="mb-3 border border-blue-200 bg-blue-50 p-2.5 text-blue-900 dark:border-blue-800/60 dark:bg-blue-900/20 dark:text-blue-100"
                >
                  <div class="flex items-start gap-2">
                    <Icon
                      name="infoCircle"
                      size="xs"
                      :stroke-width="2"
                      class="mt-0.5 flex-shrink-0"
                    />
                    <div class="min-w-0 flex-1">
                      <p class="text-xs font-semibold">{{ t('version.customUpdateDemoTitle') }}</p>
                      <p class="mt-0.5 text-[10px] leading-4 text-blue-700 dark:text-blue-300">
                        {{ t('version.customUpdateDemoHint') }}
                      </p>
                    </div>
                    <button
                      type="button"
                      class="flex h-6 w-6 flex-shrink-0 items-center justify-center text-blue-500 transition-colors hover:bg-blue-100 hover:text-blue-700 dark:text-blue-300 dark:hover:bg-blue-900/40 dark:hover:text-blue-100"
                      :title="t('version.customUpdateDemoExit')"
                      @click="exitCustomUpdateDemo"
                    >
                      <Icon name="x" size="xs" :stroke-width="2" />
                    </button>
                  </div>
                  <select
                    :value="customUpdateDemoScenario"
                    class="mt-2 h-8 w-full border border-blue-200 bg-white px-2 text-xs text-gray-700 outline-none focus:border-primary-500 focus:ring-1 focus:ring-primary-500 dark:border-blue-800 dark:bg-dark-800 dark:text-dark-200"
                    :aria-label="t('version.customUpdateDemoTitle')"
                    @change="handleCustomUpdateDemoScenarioChange"
                  >
                    <option
                      v-for="option in customUpdateDemoOptions"
                      :key="option.value"
                      :value="option.value"
                    >
                      {{ option.label }}
                    </option>
                  </select>
                </div>

                <CustomUpdateResolverSettings
                  v-if="resolverSettingsVisible && !customUpdateDemoActive"
                />

                <div
                  v-else-if="!customUpdateControllerOnline"
                  class="flex items-start gap-2 bg-amber-50 px-2.5 py-2 text-xs text-amber-700 dark:bg-amber-900/20 dark:text-amber-300"
                >
                  <Icon
                    name="exclamationTriangle"
                    size="xs"
                    :stroke-width="2"
                    class="mt-0.5 flex-shrink-0"
                  />
                  <span>{{ t('version.customUpdateOffline') }}</span>
                </div>

                <template v-else>
                  <div class="flex items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
                    <svg
                      v-if="customUpdateBusy"
                      class="h-3.5 w-3.5 flex-shrink-0 animate-spin text-primary-500"
                      fill="none"
                      viewBox="0 0 24 24"
                    >
                      <circle
                        class="opacity-25"
                        cx="12"
                        cy="12"
                        r="10"
                        stroke="currentColor"
                        stroke-width="4"
                      ></circle>
                      <path
                        class="opacity-75"
                        fill="currentColor"
                        d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                      ></path>
                    </svg>
                    <Icon
                      v-else
                      :name="customUpdateStatusIcon"
                      size="xs"
                      :stroke-width="2"
                      :class="customUpdateStatusIconClass"
                    />
                    <span>{{ customUpdateStatusLabel }}</span>
                  </div>

                  <div class="mt-3 border-t border-gray-100 pt-2.5 dark:border-dark-700">
                    <div
                      class="mb-1.5 flex items-center justify-between text-[11px] text-gray-400 dark:text-dark-500"
                    >
                      <span>{{
                        t('version.customUpdateProgress', {
                          completed: customUpdateCompletedStepCount
                        })
                      }}</span>
                      <span>{{
                        customUpdateStatus?.updated_at
                          ? formatCustomUpdateTime(customUpdateStatus.updated_at)
                          : ''
                      }}</span>
                    </div>
                    <ol class="space-y-0.5">
                      <li
                        v-for="(step, index) in customUpdateSteps"
                        :key="step.id"
                        class="flex h-7 min-w-0 items-center gap-2 text-xs"
                      >
                        <span
                          class="flex h-5 w-5 flex-shrink-0 items-center justify-center"
                          :class="customUpdateStepIconClass(step.status)"
                          :title="t(`version.customUpdateStepStatuses.${step.status}`)"
                        >
                          <Icon
                            :name="customUpdateStepIcon(step.status)"
                            size="sm"
                            :stroke-width="2"
                            :class="{ 'animate-spin': step.status === 'running' }"
                          />
                        </span>
                        <span class="w-4 flex-shrink-0 text-[10px] tabular-nums text-gray-400">
                          {{ index + 1 }}
                        </span>
                        <span
                          class="min-w-0 flex-1 truncate"
                          :class="
                            step.status === 'pending' || step.status === 'skipped'
                              ? 'text-gray-400 dark:text-dark-500'
                              : 'text-gray-600 dark:text-dark-300'
                          "
                        >
                          {{ t(`version.customUpdateSteps.${step.id}`) }}
                        </span>
                        <span
                          class="flex-shrink-0 text-[10px]"
                          :class="customUpdateStepTextClass(step.status)"
                        >
                          {{ t(`version.customUpdateStepStatuses.${step.status}`) }}
                        </span>
                      </li>
                    </ol>
                  </div>

                  <div
                    v-if="customUpdateStatus?.image"
                    class="mt-2 border-l-2 border-gray-200 pl-2 dark:border-dark-600"
                  >
                    <p class="break-all font-mono text-[10px] leading-4 text-gray-500 dark:text-dark-400">
                      {{ customUpdateStatus.image }}
                    </p>
                    <p
                      v-if="customUpdateImageDigestLabel || customUpdateCommitLabel"
                      class="mt-0.5 font-mono text-[10px] text-gray-400 dark:text-dark-500"
                    >
                      {{ customUpdateImageDigestLabel }}
                      <span v-if="customUpdateImageDigestLabel && customUpdateCommitLabel"> · </span>
                      {{ customUpdateCommitLabel }}
                    </p>
                  </div>

                  <div
                    v-if="customUpdateHasConflictResolution"
                    class="mt-3 space-y-2 border border-amber-200 bg-amber-50 p-2.5 text-xs text-amber-900 dark:border-amber-800/60 dark:bg-amber-900/20 dark:text-amber-100"
                  >
                    <div class="flex items-start justify-between gap-2">
                      <div class="min-w-0">
                        <p class="font-medium">
                          {{
                            t('version.customUpdateConflictTitle', {
                              count: customUpdateConflictFiles.length
                            })
                          }}
                        </p>
                        <p
                          v-if="customUpdateStatus?.resolver_model"
                          class="mt-0.5 text-[10px] text-amber-700 dark:text-amber-300"
                        >
                          {{
                            t('version.customUpdateResolver', {
                              model: customUpdateStatus.resolver_model
                            })
                          }}
                        </p>
                      </div>
                      <span
                        v-if="customUpdateStatus?.resolution_risk_level"
                        class="flex-shrink-0 border px-1.5 py-0.5 text-[10px] font-medium"
                        :class="customUpdateResolutionRiskClass"
                      >
                        {{
                          t(
                            `version.customUpdateResolutionRisk.${customUpdateStatus.resolution_risk_level}`
                          )
                        }}
                      </span>
                    </div>

                    <ul
                      class="max-h-24 space-y-0.5 overflow-y-auto border-l-2 border-amber-300 pl-2 font-mono text-[10px] leading-4 dark:border-amber-700"
                    >
                      <li
                        v-for="path in customUpdateConflictFiles"
                        :key="path"
                        class="break-all"
                      >
                        {{ path }}
                      </li>
                    </ul>

                    <div v-if="customUpdateStatus?.resolution_summary">
                      <p class="font-medium">{{ t('version.customUpdateResolutionSummary') }}</p>
                      <p class="mt-0.5 whitespace-pre-wrap leading-4">
                        {{ customUpdateStatus.resolution_summary }}
                      </p>
                    </div>

                    <div v-if="customUpdateStatus?.resolution_warnings?.length">
                      <p class="font-medium">{{ t('version.customUpdateResolutionWarnings') }}</p>
                      <ul class="mt-0.5 list-disc space-y-0.5 pl-4 leading-4">
                        <li
                          v-for="warning in customUpdateStatus.resolution_warnings"
                          :key="warning"
                        >
                          {{ warning }}
                        </li>
                      </ul>
                    </div>

                    <div v-if="customUpdateStatus?.resolution_diff_stat">
                      <p class="font-medium">{{ t('version.customUpdateResolutionDiff') }}</p>
                      <pre
                        class="mt-0.5 max-h-24 overflow-auto whitespace-pre-wrap break-all bg-white/70 p-1.5 font-mono text-[10px] leading-4 text-gray-700 dark:bg-dark-800/70 dark:text-dark-200"
                      >{{ customUpdateStatus.resolution_diff_stat }}</pre>
                    </div>

                    <template v-if="customUpdateStatus?.state === 'resolution_ready'">
                      <p class="leading-4">{{ t('version.customUpdateResolutionReview') }}</p>
                      <div v-if="!resolutionConfirmationVisible" class="grid grid-cols-2 gap-2">
                        <button
                          @click="handleCustomUpdateResolutionAbort"
                          :disabled="customUpdateSubmitting || customUpdateDemoActive"
                          class="flex items-center justify-center gap-1.5 border border-red-200 bg-white px-2 py-2 font-medium text-red-600 transition-colors hover:bg-red-50 disabled:opacity-50 dark:border-red-800 dark:bg-dark-800 dark:text-red-400 dark:hover:bg-red-900/20"
                        >
                          <Icon name="xCircle" size="xs" :stroke-width="2" />
                          {{ t('version.customUpdateAbortResolution') }}
                        </button>
                        <button
                          @click="resolutionConfirmationVisible = true"
                          :disabled="customUpdateSubmitting || customUpdateDemoActive"
                          class="flex items-center justify-center gap-1.5 bg-amber-600 px-2 py-2 font-medium text-white transition-colors hover:bg-amber-700 disabled:opacity-50"
                        >
                          <Icon name="check" size="xs" :stroke-width="2" />
                          {{ t('version.customUpdateAcceptResolution') }}
                        </button>
                      </div>
                      <div v-else class="space-y-2">
                        <div class="grid grid-cols-2 gap-2">
                          <button
                            @click="resolutionConfirmationVisible = false"
                            :disabled="customUpdateSubmitting"
                            class="border border-amber-300 px-2 py-2 font-medium text-amber-800 disabled:opacity-50 dark:border-amber-700 dark:text-amber-200"
                          >
                            {{ t('common.cancel') }}
                          </button>
                          <button
                            @click="handleCustomUpdateResolutionAccept"
                            :disabled="customUpdateSubmitting || customUpdateDemoActive"
                            class="bg-amber-600 px-2 py-2 font-medium text-white transition-colors hover:bg-amber-700 disabled:opacity-50"
                          >
                            {{ t('version.customUpdateConfirmResolution') }}
                          </button>
                        </div>
                      </div>
                    </template>

                    <button
                      v-else-if="customUpdateStatus?.state === 'resolution_failed'"
                      @click="handleCustomUpdateResolutionAbort"
                      :disabled="customUpdateSubmitting || customUpdateDemoActive"
                      class="flex w-full items-center justify-center gap-1.5 border border-red-200 bg-white px-2 py-2 font-medium text-red-600 transition-colors hover:bg-red-50 disabled:opacity-50 dark:border-red-800 dark:bg-dark-800 dark:text-red-400 dark:hover:bg-red-900/20"
                    >
                      <Icon name="xCircle" size="xs" :stroke-width="2" />
                      {{ t('version.customUpdateAbortResolution') }}
                    </button>
                  </div>

                  <p
                    v-if="customUpdateError"
                    class="mt-2 bg-red-50 px-2.5 py-2 text-xs leading-4 text-red-600 dark:bg-red-900/20 dark:text-red-400"
                  >
                    {{ customUpdateError }}
                  </p>

                  <div
                    v-if="customUpdateStatus?.state === 'awaiting_approval'"
                    class="mt-3 space-y-2"
                  >
                    <div
                      class="flex items-start gap-2 bg-green-50 px-2.5 py-2 text-xs leading-4 text-green-700 dark:bg-green-900/20 dark:text-green-300"
                    >
                      <Icon
                        name="checkCircle"
                        size="xs"
                        :stroke-width="2"
                        class="mt-0.5 flex-shrink-0"
                      />
                      <span>{{ t('version.customUpdateStagingReady') }}</span>
                    </div>

                    <button
                      v-if="!promotionConfirmationVisible"
                      @click="promotionConfirmationVisible = true"
                      :disabled="customUpdateDemoActive"
                      class="flex w-full items-center justify-center gap-2 rounded-lg bg-green-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-green-700 disabled:cursor-not-allowed disabled:opacity-50"
                    >
                      <Icon name="server" size="sm" :stroke-width="2" />
                      {{ t('version.customUpdatePromote') }}
                    </button>

                    <div v-else class="space-y-2">
                      <p
                        class="flex items-start gap-1.5 text-xs leading-4 text-amber-600 dark:text-amber-400"
                      >
                        <Icon
                          name="exclamationTriangle"
                          size="xs"
                          :stroke-width="2"
                          class="mt-0.5 flex-shrink-0"
                        />
                        <span>{{ t('version.customUpdatePromoteWarning') }}</span>
                      </p>
                      <div class="grid grid-cols-2 gap-2">
                        <button
                          @click="promotionConfirmationVisible = false"
                          :disabled="customUpdateSubmitting"
                          class="rounded-lg border border-gray-200 px-3 py-2 text-sm font-medium text-gray-600 transition-colors hover:bg-gray-50 disabled:opacity-50 dark:border-dark-600 dark:text-dark-300 dark:hover:bg-dark-700"
                        >
                          {{ t('common.cancel') }}
                        </button>
                        <button
                          @click="handleCustomUpdatePromotion"
                          :disabled="customUpdateSubmitting || customUpdateDemoActive"
                          class="flex items-center justify-center gap-1.5 rounded-lg bg-green-600 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-green-700 disabled:opacity-50"
                        >
                          <Icon name="check" size="xs" :stroke-width="2" />
                          {{ t('version.customUpdateConfirmPromote') }}
                        </button>
                      </div>
                    </div>
                  </div>

                  <button
                    v-else-if="customUpdateCanStage"
                    @click="handleCustomUpdateStage"
                    :disabled="customUpdateSubmitting"
                    class="mt-3 flex w-full items-center justify-center gap-2 rounded-lg bg-primary-500 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-600 disabled:cursor-not-allowed disabled:opacity-50"
                  >
                    <Icon name="sync" size="sm" :stroke-width="2" />
                    {{
                      customUpdateStatus?.state === 'failed' ||
                      customUpdateStatus?.state === 'resolution_failed'
                        ? t('version.customUpdateRetry')
                        : t('version.customUpdateStage')
                    }}
                  </button>
                </template>
              </div>

              <!-- Priority 1: Update error (must check before hasUpdate) -->
              <div
                v-if="customUpdateStatusChecked && !customUpdateAvailable && updateError"
                class="space-y-2"
              >
                <div
                  class="flex items-center gap-3 rounded-lg border border-red-200 bg-red-50 p-3 dark:border-red-800/50 dark:bg-red-900/20"
                >
                  <div
                    class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-red-100 dark:bg-red-900/50"
                  >
                    <Icon
                      name="x"
                      size="sm"
                      :stroke-width="2"
                      class="text-red-600 dark:text-red-400"
                    />
                  </div>
                  <div class="min-w-0 flex-1">
                    <p class="text-sm font-medium text-red-700 dark:text-red-300">
                      {{ t('version.updateFailed') }}
                    </p>
                    <p class="truncate text-xs text-red-600/70 dark:text-red-400/70">
                      {{ updateError }}
                    </p>
                  </div>
                </div>

                <!-- Retry button -->
                <button
                  @click="handleUpdate"
                  :disabled="updating"
                  class="flex w-full items-center justify-center gap-2 rounded-lg bg-red-500 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-red-600 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  {{ t('version.retry') }}
                </button>
              </div>

              <!-- Priority 2: Update success - need restart -->
              <div
                v-else-if="
                  customUpdateStatusChecked &&
                  !customUpdateAvailable &&
                  updateSuccess &&
                  needRestart
                "
                class="space-y-2"
              >
                <div
                  class="flex items-center gap-3 rounded-lg border border-green-200 bg-green-50 p-3 dark:border-green-800/50 dark:bg-green-900/20"
                >
                  <div
                    class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-green-100 dark:bg-green-900/50"
                  >
                    <svg
                      class="h-4 w-4 text-green-600 dark:text-green-400"
                      fill="none"
                      viewBox="0 0 24 24"
                      stroke="currentColor"
                      stroke-width="2"
                    >
                      <path stroke-linecap="round" stroke-linejoin="round" d="M5 13l4 4L19 7" />
                    </svg>
                  </div>
                  <div class="min-w-0 flex-1">
                    <p class="text-sm font-medium text-green-700 dark:text-green-300">
                      {{
                        successKind === 'rollback'
                          ? t('version.rollbackComplete')
                          : t('version.updateComplete')
                      }}
                    </p>
                    <p class="text-xs text-green-600/70 dark:text-green-400/70">
                      {{ t('version.restartRequired') }}
                    </p>
                  </div>
                </div>

                <!-- Restart button with countdown -->
                <button
                  @click="handleRestart"
                  :disabled="restarting"
                  class="flex w-full items-center justify-center gap-2 rounded-lg bg-green-500 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-green-600 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <svg
                    v-if="restarting"
                    class="h-4 w-4 animate-spin"
                    fill="none"
                    viewBox="0 0 24 24"
                  >
                    <circle
                      class="opacity-25"
                      cx="12"
                      cy="12"
                      r="10"
                      stroke="currentColor"
                      stroke-width="4"
                    ></circle>
                    <path
                      class="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                    ></path>
                  </svg>
                  <svg
                    v-else
                    class="h-4 w-4"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"
                    />
                  </svg>
                  <template v-if="restarting">
                    <span>{{ t('version.restarting') }}</span>
                    <span v-if="restartCountdown > 0" class="tabular-nums"
                      >({{ restartCountdown }}s)</span
                    >
                  </template>
                  <span v-else>{{ t('version.restartNow') }}</span>
                </button>
              </div>

              <!-- Priority 3: Update available for source build - show git pull hint -->
              <div
                v-else-if="
                  customUpdateStatusChecked &&
                  !customUpdateAvailable &&
                  hasUpdate &&
                  !isReleaseBuild
                "
                class="space-y-2"
              >
                <a
                  v-if="releaseInfo?.html_url && releaseInfo.html_url !== '#'"
                  :href="releaseInfo.html_url"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="group flex items-center gap-3 rounded-lg border border-amber-200 bg-amber-50 p-3 transition-colors hover:bg-amber-100 dark:border-amber-800/50 dark:bg-amber-900/20 dark:hover:bg-amber-900/30"
                >
                  <div
                    class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-amber-100 dark:bg-amber-900/50"
                  >
                    <Icon
                      name="download"
                      size="sm"
                      :stroke-width="2"
                      class="text-amber-600 dark:text-amber-400"
                    />
                  </div>
                  <div class="min-w-0 flex-1">
                    <p class="text-sm font-medium text-amber-700 dark:text-amber-300">
                      {{ t('version.updateAvailable') }}
                    </p>
                    <p class="text-xs text-amber-600/70 dark:text-amber-400/70">
                      v{{ latestVersion }}
                    </p>
                  </div>
                  <svg
                    class="h-4 w-4 text-amber-500 transition-transform group-hover:translate-x-0.5 dark:text-amber-400"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <path stroke-linecap="round" stroke-linejoin="round" d="M9 5l7 7-7 7" />
                  </svg>
                </a>
                <!-- Source build hint -->
                <div
                  class="flex items-center gap-2 rounded-lg border border-blue-200 bg-blue-50 p-2 dark:border-blue-800/50 dark:bg-blue-900/20"
                >
                  <svg
                    class="h-3.5 w-3.5 flex-shrink-0 text-blue-500 dark:text-blue-400"
                    fill="none"
                    viewBox="0 0 24 24"
                    stroke="currentColor"
                    stroke-width="2"
                  >
                    <path
                      stroke-linecap="round"
                      stroke-linejoin="round"
                      d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                    />
                  </svg>
                  <p class="text-xs text-blue-600 dark:text-blue-400">
                    {{ t('version.sourceModeHint') }}
                  </p>
                </div>
              </div>

              <!-- Priority 4: Update available for release build - show update button -->
              <div
                v-else-if="
                  customUpdateStatusChecked &&
                  !customUpdateAvailable &&
                  hasUpdate &&
                  isReleaseBuild
                "
                class="space-y-2"
              >
                <!-- Update info card -->
                <div
                  class="flex items-center gap-3 rounded-lg border border-amber-200 bg-amber-50 p-3 dark:border-amber-800/50 dark:bg-amber-900/20"
                >
                <div
                  class="flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-full bg-amber-100 dark:bg-amber-900/50"
                >
                  <Icon
                    name="download"
                    size="sm"
                    :stroke-width="2"
                    class="text-amber-600 dark:text-amber-400"
                  />
                </div>
                  <div class="min-w-0 flex-1">
                    <p class="text-sm font-medium text-amber-700 dark:text-amber-300">
                      {{ t('version.updateAvailable') }}
                    </p>
                    <p class="text-xs text-amber-600/70 dark:text-amber-400/70">
                      v{{ latestVersion }}
                    </p>
                  </div>
                </div>

                <!-- Update button -->
                <button
                  @click="handleUpdate"
                  :disabled="updating"
                  class="flex w-full items-center justify-center gap-2 rounded-lg bg-primary-500 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-primary-600 disabled:cursor-not-allowed disabled:opacity-50"
                >
                  <svg v-if="updating" class="h-4 w-4 animate-spin" fill="none" viewBox="0 0 24 24">
                    <circle
                      class="opacity-25"
                      cx="12"
                      cy="12"
                      r="10"
                      stroke="currentColor"
                      stroke-width="4"
                    ></circle>
                    <path
                      class="opacity-75"
                      fill="currentColor"
                      d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                    ></path>
                  </svg>
                  <Icon v-else name="download" size="sm" :stroke-width="2" />
                  {{ updating ? t('version.updating') : t('version.updateNow') }}
                </button>

                <!-- View release link -->
                <a
                  v-if="releaseInfo?.html_url && releaseInfo.html_url !== '#'"
                  :href="releaseInfo.html_url"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="flex items-center justify-center gap-1 text-xs text-gray-500 transition-colors hover:text-gray-700 dark:text-dark-400 dark:hover:text-dark-200"
                >
                  {{ t('version.viewChangelog') }}
                  <Icon name="externalLink" size="xs" :stroke-width="2" />
                </a>
              </div>

              <!-- Priority 5: Up to date - GitHub link + version rollback -->
              <div
                v-else-if="customUpdateStatusChecked && !customUpdateAvailable"
                class="space-y-2"
              >
                <a
                  v-if="releaseInfo?.html_url && releaseInfo.html_url !== '#'"
                  :href="releaseInfo.html_url"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="flex items-center justify-center gap-2 py-2 text-sm text-gray-500 transition-colors hover:text-gray-700 dark:text-dark-400 dark:hover:text-dark-200"
                >
                  <svg class="h-4 w-4" fill="currentColor" viewBox="0 0 24 24">
                    <path
                      fill-rule="evenodd"
                      clip-rule="evenodd"
                      d="M12 2C6.477 2 2 6.477 2 12c0 4.42 2.865 8.17 6.839 9.49.5.092.682-.217.682-.482 0-.237-.008-.866-.013-1.7-2.782.604-3.369-1.34-3.369-1.34-.454-1.156-1.11-1.464-1.11-1.464-.908-.62.069-.608.069-.608 1.003.07 1.531 1.03 1.531 1.03.892 1.529 2.341 1.087 2.91.831.092-.646.35-1.086.636-1.336-2.22-.253-4.555-1.11-4.555-4.943 0-1.091.39-1.984 1.029-2.683-.103-.253-.446-1.27.098-2.647 0 0 .84-.269 2.75 1.025A9.578 9.578 0 0112 6.836c.85.004 1.705.114 2.504.336 1.909-1.294 2.747-1.025 2.747-1.025.546 1.377.203 2.394.1 2.647.64.699 1.028 1.592 1.028 2.683 0 3.842-2.339 4.687-4.566 4.935.359.309.678.919.678 1.852 0 1.336-.012 2.415-.012 2.743 0 .267.18.578.688.48C19.138 20.167 22 16.418 22 12c0-5.523-4.477-10-10-10z"
                    />
                  </svg>
                  {{ t('version.viewRelease') }}
                </a>

                <!-- Version rollback entry -->
                <div class="border-t border-gray-100 pt-2 dark:border-dark-700">
                  <button
                    @click="toggleRollbackPanel"
                    class="group flex w-full items-center justify-between rounded-lg px-2 py-1.5 text-xs text-gray-400 transition-colors hover:bg-gray-50 hover:text-gray-600 dark:text-dark-500 dark:hover:bg-dark-700/50 dark:hover:text-dark-300"
                  >
                    <span class="flex items-center gap-1.5">
                      <Icon name="clock" size="xs" :stroke-width="2" />
                      {{ t('version.rollback') }}
                    </span>
                    <Icon
                      name="chevronDown"
                      size="xs"
                      :stroke-width="2"
                      class="transition-transform duration-200"
                      :class="{ 'rotate-180': rollbackPanelOpen }"
                    />
                  </button>

                  <transition name="rollback">
                    <div v-if="rollbackPanelOpen" class="mt-2 space-y-2">
                      <!-- Source build: online rollback unavailable, use git instead -->
                      <div
                        v-if="!isReleaseBuild"
                        class="flex items-center gap-2 rounded-lg border border-blue-200 bg-blue-50 p-2 dark:border-blue-800/50 dark:bg-blue-900/20"
                      >
                        <svg
                          class="h-3.5 w-3.5 flex-shrink-0 text-blue-500 dark:text-blue-400"
                          fill="none"
                          viewBox="0 0 24 24"
                          stroke="currentColor"
                          stroke-width="2"
                        >
                          <path
                            stroke-linecap="round"
                            stroke-linejoin="round"
                            d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                          />
                        </svg>
                        <p class="min-w-0 flex-1 text-xs leading-4 text-blue-600 dark:text-blue-400">
                          {{ t('version.rollbackSourceHint') }}
                        </p>
                      </div>

                      <!-- Loading versions -->
                      <div
                        v-else-if="rollbackVersionsLoading"
                        class="flex items-center justify-center py-4"
                      >
                        <svg
                          class="h-5 w-5 animate-spin text-primary-500"
                          fill="none"
                          viewBox="0 0 24 24"
                        >
                          <circle
                            class="opacity-25"
                            cx="12"
                            cy="12"
                            r="10"
                            stroke="currentColor"
                            stroke-width="4"
                          ></circle>
                          <path
                            class="opacity-75"
                            fill="currentColor"
                            d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                          ></path>
                        </svg>
                      </div>

                      <!-- Load error + retry -->
                      <div v-else-if="rollbackVersionsError" class="space-y-2">
                        <p
                          class="rounded-lg border border-red-200 bg-red-50 p-2.5 text-xs text-red-600 dark:border-red-800/50 dark:bg-red-900/20 dark:text-red-400"
                        >
                          {{ rollbackVersionsError }}
                        </p>
                        <button
                          @click="loadRollbackVersions"
                          class="w-full rounded-lg border border-gray-200 py-1.5 text-xs text-gray-500 transition-colors hover:bg-gray-50 hover:text-gray-700 dark:border-dark-700 dark:text-dark-400 dark:hover:bg-dark-700/50 dark:hover:text-dark-200"
                        >
                          {{ t('version.retry') }}
                        </button>
                      </div>

                      <!-- No versions available -->
                      <p
                        v-else-if="rollbackVersions.length === 0"
                        class="py-3 text-center text-xs text-gray-400 dark:text-dark-500"
                      >
                        {{ t('version.noRollbackVersions') }}
                      </p>

                      <!-- Version list -->
                      <template v-else>
                        <p class="px-0.5 text-[11px] text-gray-400 dark:text-dark-500">
                          {{ t('version.rollbackSelectVersion') }}
                        </p>

                        <button
                          v-for="item in rollbackVersions"
                          :key="item.version"
                          @click="selectRollbackVersion(item.version)"
                          :disabled="rollingBack"
                          class="flex w-full items-center justify-between rounded-lg border px-3 py-2 text-left transition-all disabled:cursor-not-allowed disabled:opacity-60"
                          :class="
                            selectedRollbackVersion === item.version
                              ? 'border-amber-300 bg-amber-50 shadow-sm dark:border-amber-700 dark:bg-amber-900/20'
                              : 'border-gray-200 hover:border-gray-300 hover:bg-gray-50 dark:border-dark-700 dark:hover:border-dark-600 dark:hover:bg-dark-700/40'
                          "
                        >
                          <span class="flex items-center gap-2">
                            <span
                              class="flex h-3.5 w-3.5 items-center justify-center rounded-full border transition-colors"
                              :class="
                                selectedRollbackVersion === item.version
                                  ? 'border-amber-500'
                                  : 'border-gray-300 dark:border-dark-500'
                              "
                            >
                              <span
                                v-if="selectedRollbackVersion === item.version"
                                class="h-1.5 w-1.5 rounded-full bg-amber-500"
                              ></span>
                            </span>
                            <span
                              class="text-sm font-semibold"
                              :class="
                                selectedRollbackVersion === item.version
                                  ? 'text-amber-700 dark:text-amber-300'
                                  : 'text-gray-700 dark:text-dark-200'
                              "
                              >v{{ item.version }}</span
                            >
                          </span>
                          <span class="text-[11px] tabular-nums text-gray-400 dark:text-dark-500">
                            {{ formatPublishedAt(item.published_at) }}
                          </span>
                        </button>

                        <!-- Selected version: manual command (per deploy method) + confirm -->
                        <transition name="rollback">
                          <div v-if="selectedRollbackVersion" class="space-y-2">
                            <p class="px-0.5 text-[11px] text-gray-400 dark:text-dark-500">
                              {{ t('version.manualRollbackCommand') }}
                            </p>

                            <!-- Terminal-style block with deploy-method tabs -->
                            <div
                              class="overflow-hidden rounded-lg border border-gray-200 dark:border-dark-600"
                            >
                              <div
                                class="flex items-center justify-between border-b border-gray-200 bg-gray-100 px-2 py-1.5 dark:border-dark-600 dark:bg-dark-700"
                              >
                                <div
                                  class="flex items-center gap-0.5 rounded-md bg-gray-200/70 p-0.5 dark:bg-dark-600/70"
                                >
                                  <button
                                    v-for="tab in manualTabs"
                                    :key="tab.key"
                                    @click="manualTab = tab.key"
                                    class="rounded px-2 py-0.5 text-[11px] font-medium transition-colors"
                                    :class="
                                      manualTab === tab.key
                                        ? 'bg-white text-gray-700 shadow-sm dark:bg-dark-800 dark:text-dark-100'
                                        : 'text-gray-400 hover:text-gray-600 dark:text-dark-400 dark:hover:text-dark-200'
                                    "
                                  >
                                    {{ tab.label }}
                                  </button>
                                </div>
                                <button
                                  @click="copyToClipboard(activeManualCommand)"
                                  class="flex items-center gap-1 rounded px-1.5 py-0.5 text-[11px] text-gray-400 transition-colors hover:bg-gray-200 hover:text-gray-600 dark:text-dark-400 dark:hover:bg-dark-600 dark:hover:text-dark-200"
                                >
                                  <Icon
                                    :name="copied ? 'check' : 'copy'"
                                    size="xs"
                                    :stroke-width="2"
                                    :class="copied ? 'text-green-500' : ''"
                                  />
                                  {{ copied ? t('version.copied') : t('version.copyCommand') }}
                                </button>
                              </div>
                              <code
                                class="block select-all whitespace-pre-wrap break-all bg-gray-50 p-2.5 font-mono text-[10px] leading-relaxed text-gray-600 dark:bg-dark-900 dark:text-dark-300"
                                >{{ activeManualCommand }}</code
                              >
                            </div>

                            <p
                              class="flex items-start gap-1.5 px-0.5 text-[11px] leading-4 text-amber-600 dark:text-amber-400"
                            >
                              <Icon
                                name="exclamationTriangle"
                                size="xs"
                                :stroke-width="2"
                                class="mt-px flex-shrink-0"
                              />
                              {{ t('version.rollbackWarning') }}
                            </p>

                            <p
                              v-if="rollbackError"
                              class="rounded-lg border border-red-200 bg-red-50 p-2 text-xs text-red-600 dark:border-red-800/50 dark:bg-red-900/20 dark:text-red-400"
                            >
                              {{ rollbackError }}
                            </p>

                            <button
                              @click="handleRollback"
                              :disabled="rollingBack"
                              class="flex w-full items-center justify-center gap-2 rounded-lg bg-amber-500 px-4 py-2 text-sm font-medium text-white shadow-sm transition-colors hover:bg-amber-600 disabled:cursor-not-allowed disabled:opacity-50"
                            >
                              <svg
                                v-if="rollingBack"
                                class="h-4 w-4 animate-spin"
                                fill="none"
                                viewBox="0 0 24 24"
                              >
                                <circle
                                  class="opacity-25"
                                  cx="12"
                                  cy="12"
                                  r="10"
                                  stroke="currentColor"
                                  stroke-width="4"
                                ></circle>
                                <path
                                  class="opacity-75"
                                  fill="currentColor"
                                  d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
                                ></path>
                              </svg>
                              <Icon v-else name="clock" size="sm" :stroke-width="2" />
                              <span>{{
                                rollingBack
                                  ? t('version.rollingBack')
                                  : t('version.rollbackConfirm', {
                                      version: 'v' + selectedRollbackVersion
                                    })
                              }}</span>
                            </button>
                          </div>
                        </transition>
                      </template>
                    </div>
                  </transition>
                </div>
              </div>
            </template>
          </div>
        </div>
      </transition>
    </template>

    <!-- Non-admin: Simple static version text -->
    <span v-else-if="version" class="text-xs text-gray-500 dark:text-dark-400">
      v{{ version }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onBeforeUnmount } from 'vue'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import {
  performUpdate,
  restartService,
  getRollbackVersions,
  rollback as rollbackAPI,
  type RollbackVersionInfo
} from '@/api/admin/system'
import {
  getCustomUpdateStatus,
  startCustomUpdate,
  acceptCustomUpdateResolution,
  abortCustomUpdateResolution,
  promoteCustomUpdate,
  type CustomUpdateState,
  type CustomUpdateStatus,
  type CustomUpdateStepStatus
} from '@/api/admin/customBuild'
import { useClipboard } from '@/composables/useClipboard'
import Icon from '@/components/icons/Icon.vue'
import OfficialVersionStatus from '@/components/common/OfficialVersionStatus.vue'
import CustomUpdateResolverSettings from '@/components/common/CustomUpdateResolverSettings.vue'
import { OFFICIAL_REPOSITORY } from '@/constants/version'
import {
  CUSTOM_UPDATE_DEMO_QUERY_PARAM,
  CUSTOM_UPDATE_DEMO_SCENARIOS,
  createCustomUpdateDemoStatus,
  isCustomUpdateDemoScenario,
  resolveCustomUpdateDemoScenario,
  type CustomUpdateDemoCopy,
  type CustomUpdateDemoScenario
} from '@/utils/customUpdateDemo'
import { resolveCustomUpdateSteps } from '@/utils/customUpdateSteps'

const GITHUB_REPO = OFFICIAL_REPOSITORY
// Docker Hub image published by CI (tags carry no "v" prefix, e.g. weishaw/sub2api:0.1.146)
const DOCKER_IMAGE = 'weishaw/sub2api'

const { t } = useI18n()

const props = defineProps<{
  version?: string
}>()

const authStore = useAuthStore()
const appStore = useAppStore()

const isAdmin = computed(() => authStore.isAdmin)

const dropdownOpen = ref(false)
const dropdownRef = ref<HTMLElement | null>(null)

// Use store's cached version state
const loading = computed(() => appStore.versionLoading)
const currentVersion = computed(() => appStore.currentVersion || props.version || '')
const latestVersion = computed(() => appStore.latestVersion)
const hasUpdate = computed(() => appStore.hasUpdate)
const releaseInfo = computed(() => appStore.releaseInfo)
const buildType = computed(() => appStore.buildType)

// Update process states (local to this component)
const updating = ref(false)
const restarting = ref(false)
const needRestart = ref(false)
const updateError = ref('')
const updateSuccess = ref(false)
const restartCountdown = ref(0)
// Distinguishes the success + restart panel between update and rollback flows
const successKind = ref<'update' | 'rollback'>('update')

// ToCreate custom image update states
const customUpdateStatus = ref<CustomUpdateStatus | null>(null)
const customUpdateStatusChecked = ref(false)
const customUpdateSubmitting = ref(false)
const customUpdateLoadError = ref('')
const customUpdateActionError = ref('')
const promotionConfirmationVisible = ref(false)
const resolutionConfirmationVisible = ref(false)
const resolverSettingsVisible = ref(false)
let customUpdatePollingTimer: number | undefined
const initialCustomUpdateDemoScenario =
  typeof window === 'undefined'
    ? null
    : resolveCustomUpdateDemoScenario(window.location.search, window.location.port)
const customUpdateDemoScenario = ref<CustomUpdateDemoScenario | null>(
  initialCustomUpdateDemoScenario
)

// Rollback states
const rollbackPanelOpen = ref(false)
const rollbackVersions = ref<RollbackVersionInfo[]>([])
const rollbackVersionsLoading = ref(false)
const rollbackVersionsError = ref('')
const selectedRollbackVersion = ref('')
const rollingBack = ref(false)
const rollbackError = ref('')

const { copied, copyToClipboard } = useClipboard()

const customUpdateBusyStates = new Set<CustomUpdateState>([
  'queued',
  'checking',
  'merging',
  'conflict_detected',
  'ai_resolving',
  'pushing',
  'building',
  'staging',
  'validating',
  'promoting'
])

const customUpdateDemoActive = computed(() => customUpdateDemoScenario.value !== null)
const customUpdateDemoOptions = computed(() =>
  CUSTOM_UPDATE_DEMO_SCENARIOS.map((scenario) => ({
    value: scenario,
    label: t(`version.customUpdateDemoScenarios.${scenario}`)
  }))
)
const customUpdateDemoCopy = computed<CustomUpdateDemoCopy>(() => ({
  reviewSummary: t('version.customUpdateDemoReviewSummary'),
  highRiskSummary: t('version.customUpdateDemoHighRiskSummary'),
  reviewWarnings: [t('version.customUpdateDemoReviewWarning')],
  highRiskWarnings: [
    t('version.customUpdateDemoHighRiskWarningPrimary'),
    t('version.customUpdateDemoHighRiskWarningSecondary')
  ],
  failedError: t('version.customUpdateDemoFailedError')
}))
const customUpdateAvailable = computed(() => customUpdateStatus.value?.enabled === true)
const customUpdateControllerOnline = computed(
  () => customUpdateStatus.value?.controller_online === true
)
const customUpdateBusy = computed(() => {
  const state = customUpdateStatus.value?.state
  return state ? customUpdateBusyStates.has(state) : false
})
const customUpdateCanStage = computed(() => {
  if (customUpdateDemoActive.value) return false
  if (!customUpdateControllerOnline.value) return false
  const state = customUpdateStatus.value?.state
  return (
    state === 'idle' ||
    state === 'completed' ||
    state === 'failed' ||
    state === 'resolution_failed' ||
    state === 'aborted'
  )
})
const customUpdateFailed = computed(
  () =>
    customUpdateStatus.value?.state === 'failed' ||
    customUpdateStatus.value?.state === 'resolution_failed'
)
const customUpdateStatusIcon = computed(() => {
  if (customUpdateStatus.value?.state === 'resolution_ready') return 'exclamationCircle' as const
  return customUpdateFailed.value ? ('xCircle' as const) : ('checkCircle' as const)
})
const customUpdateStatusIconClass = computed(() => {
  if (customUpdateStatus.value?.state === 'resolution_ready') return 'text-amber-500'
  return customUpdateFailed.value ? 'text-red-500' : 'text-green-500'
})
const customUpdateConflictFiles = computed(() => customUpdateStatus.value?.conflict_files || [])
const customUpdateHasConflictResolution = computed(
  () => customUpdateConflictFiles.value.length > 0
)
const customUpdateStatusLabel = computed(() => {
  const state = customUpdateStatus.value?.state || 'idle'
  return t(`version.customUpdateStates.${state}`)
})
const customUpdateSteps = computed(() => resolveCustomUpdateSteps(customUpdateStatus.value))
const customUpdateCompletedStepCount = computed(
  () =>
    customUpdateSteps.value.filter(
      (step) => step.status === 'completed' || step.status === 'skipped'
    ).length
)
const customUpdateStatusDotClass = computed(() => {
  if (!customUpdateControllerOnline.value) return 'bg-gray-300 dark:bg-dark-500'
  switch (customUpdateStatus.value?.state) {
    case 'failed':
    case 'resolution_failed':
      return 'bg-red-500'
    case 'resolution_ready':
      return 'bg-amber-500'
    case 'awaiting_approval':
    case 'completed':
      return 'bg-green-500'
    case 'queued':
    case 'checking':
    case 'merging':
    case 'conflict_detected':
    case 'ai_resolving':
    case 'pushing':
    case 'building':
    case 'staging':
    case 'validating':
    case 'promoting':
      return 'animate-pulse bg-primary-500'
    default:
      return 'bg-gray-400'
  }
})
const customUpdateResolutionRiskClass = computed(() => {
  switch (customUpdateStatus.value?.resolution_risk_level) {
    case 'high':
      return 'border-red-300 bg-red-100 text-red-700 dark:border-red-700 dark:bg-red-900/40 dark:text-red-300'
    case 'medium':
      return 'border-amber-300 bg-amber-100 text-amber-800 dark:border-amber-700 dark:bg-amber-900/40 dark:text-amber-200'
    default:
      return 'border-green-300 bg-green-100 text-green-700 dark:border-green-700 dark:bg-green-900/40 dark:text-green-300'
  }
})
const customUpdateImageDigestLabel = computed(() => {
  const digest = customUpdateStatus.value?.image_digest?.split('@').pop() || ''
  return digest.length > 22 ? `${digest.slice(0, 22)}...` : digest
})
const customUpdateCommitLabel = computed(() => {
  const commit = customUpdateStatus.value?.source_commit
  return commit ? t('version.customUpdateCommit', { commit: commit.slice(0, 8) }) : ''
})
const customUpdateError = computed(
  () =>
    customUpdateStatus.value?.error ||
    customUpdateActionError.value ||
    customUpdateLoadError.value
)

function customUpdateStepIcon(
  status: CustomUpdateStepStatus
): 'clock' | 'sync' | 'checkCircle' | 'xCircle' | 'exclamationCircle' | 'check' {
  switch (status) {
    case 'running':
      return 'sync'
    case 'completed':
      return 'checkCircle'
    case 'failed':
      return 'xCircle'
    case 'action_required':
      return 'exclamationCircle'
    case 'skipped':
      return 'check'
    default:
      return 'clock'
  }
}

function customUpdateStepIconClass(status: CustomUpdateStepStatus): string {
  switch (status) {
    case 'running':
      return 'text-primary-500'
    case 'completed':
      return 'text-green-500'
    case 'failed':
      return 'text-red-500'
    case 'action_required':
      return 'text-amber-500'
    default:
      return 'text-gray-300 dark:text-dark-500'
  }
}

function customUpdateStepTextClass(status: CustomUpdateStepStatus): string {
  switch (status) {
    case 'running':
      return 'text-primary-500'
    case 'completed':
      return 'text-green-600 dark:text-green-400'
    case 'failed':
      return 'text-red-600 dark:text-red-400'
    case 'action_required':
      return 'text-amber-600 dark:text-amber-400'
    default:
      return 'text-gray-400 dark:text-dark-500'
  }
}

function formatCustomUpdateTime(value: string): string {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat(undefined, {
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  }).format(date)
}

// Manual rollback methods differ by deployment: script installs use install.sh,
// docker deployments pin the image tag instead
const manualTab = ref<'script' | 'docker'>('script')

const manualTabs = computed(() => [
  { key: 'script' as const, label: t('version.deployScript') },
  { key: 'docker' as const, label: t('version.deployDocker') }
])

const scriptRollbackCommand = computed(() => {
  if (!selectedRollbackVersion.value) return ''
  const tag = `v${selectedRollbackVersion.value}`
  return `curl -sSL https://raw.githubusercontent.com/${GITHUB_REPO}/${tag}/deploy/install.sh | sudo bash -s -- rollback ${tag}`
})

const dockerRollbackCommand = computed(() => {
  if (!selectedRollbackVersion.value) return ''
  return [
    `# ${t('version.dockerEditCompose')}`,
    `image: ${DOCKER_IMAGE}:${selectedRollbackVersion.value}`,
    '',
    `# ${t('version.dockerRecreate')}`,
    'docker compose up -d'
  ].join('\n')
})

const activeManualCommand = computed(() =>
  manualTab.value === 'docker' ? dockerRollbackCommand.value : scriptRollbackCommand.value
)

// Only show update check for release builds (binary/docker deployment)
const isReleaseBuild = computed(() => buildType.value === 'release')

function toggleDropdown() {
  dropdownOpen.value = !dropdownOpen.value
  if (dropdownOpen.value) {
    void refreshCustomUpdateStatus()
    startCustomUpdatePolling()
  } else {
    stopCustomUpdatePolling()
    promotionConfirmationVisible.value = false
    resolutionConfirmationVisible.value = false
    resolverSettingsVisible.value = false
  }
}

function closeDropdown() {
  dropdownOpen.value = false
  stopCustomUpdatePolling()
  promotionConfirmationVisible.value = false
  resolutionConfirmationVisible.value = false
  resolverSettingsVisible.value = false
}

async function refreshVersion(force = true) {
  if (!isAdmin.value) return

  // Reset update states when refreshing
  updateError.value = ''
  updateSuccess.value = false
  needRestart.value = false
  resetRollbackState()

  await appStore.fetchVersion(force)
}

async function handleUpdate() {
  if (updating.value) return

  updating.value = true
  updateError.value = ''
  updateSuccess.value = false

  try {
    const result = await performUpdate()
    successKind.value = 'update'
    updateSuccess.value = true
    needRestart.value = result.need_restart
    // Clear version cache to reflect update completed
    appStore.clearVersionCache()
  } catch (error: unknown) {
    const err = error as { response?: { data?: { message?: string } }; message?: string }
    updateError.value = err.response?.data?.message || err.message || t('version.updateFailed')
  } finally {
    updating.value = false
  }
}

function applyCustomUpdateDemoStatus() {
  const scenario = customUpdateDemoScenario.value
  if (!scenario) return
  customUpdateStatus.value = createCustomUpdateDemoStatus(
    scenario,
    customUpdateDemoCopy.value
  )
  customUpdateLoadError.value = ''
  customUpdateActionError.value = ''
  customUpdateStatusChecked.value = true
  promotionConfirmationVisible.value = false
  resolutionConfirmationVisible.value = false
  resolverSettingsVisible.value = false
}

function replaceCustomUpdateDemoQuery(scenario: CustomUpdateDemoScenario | null) {
  const url = new URL(window.location.href)
  if (scenario) {
    url.searchParams.set(CUSTOM_UPDATE_DEMO_QUERY_PARAM, scenario)
  } else {
    url.searchParams.delete(CUSTOM_UPDATE_DEMO_QUERY_PARAM)
  }
  window.history.replaceState(window.history.state, '', url.toString())
}

function handleCustomUpdateDemoScenarioChange(event: Event) {
  const value = (event.target as HTMLSelectElement).value
  if (!isCustomUpdateDemoScenario(value)) return
  customUpdateDemoScenario.value = value
  replaceCustomUpdateDemoQuery(value)
  applyCustomUpdateDemoStatus()
}

async function exitCustomUpdateDemo() {
  customUpdateDemoScenario.value = null
  replaceCustomUpdateDemoQuery(null)
  customUpdateStatus.value = null
  await refreshCustomUpdateStatus()
  if (dropdownOpen.value) {
    startCustomUpdatePolling()
  }
}

async function refreshCustomUpdateStatus() {
  if (!isAdmin.value) return
  if (customUpdateDemoActive.value) {
    applyCustomUpdateDemoStatus()
    return
  }
  try {
    const status = await getCustomUpdateStatus()
    customUpdateStatus.value = status
    customUpdateLoadError.value = ''
    if (status.state !== 'awaiting_approval') {
      promotionConfirmationVisible.value = false
    }
    if (status.state !== 'resolution_ready') {
      resolutionConfirmationVisible.value = false
    }
  } catch (error: unknown) {
    const err = error as { response?: { data?: { message?: string } }; message?: string }
    customUpdateLoadError.value = err.response?.data?.message || err.message || ''
  } finally {
    customUpdateStatusChecked.value = true
  }
}

function startCustomUpdatePolling() {
  stopCustomUpdatePolling()
  if (customUpdateDemoActive.value) return
  customUpdatePollingTimer = window.setInterval(() => {
    void refreshCustomUpdateStatus()
  }, 3000)
}

function stopCustomUpdatePolling() {
  if (customUpdatePollingTimer !== undefined) {
    window.clearInterval(customUpdatePollingTimer)
    customUpdatePollingTimer = undefined
  }
}

async function handleCustomUpdateStage() {
  if (
    customUpdateDemoActive.value ||
    customUpdateSubmitting.value ||
    !customUpdateCanStage.value
  ) {
    return
  }

  customUpdateSubmitting.value = true
  customUpdateActionError.value = ''
  try {
    const result = await startCustomUpdate()
    if (customUpdateStatus.value) {
      customUpdateStatus.value = {
        ...customUpdateStatus.value,
        state: result.state,
        action: result.action,
        request_id: result.request_id,
        message: result.message,
        error: undefined,
        steps: [],
        resolution_id: undefined,
        conflict_files: [],
        resolution_summary: undefined,
        resolution_risk_level: undefined,
        resolution_warnings: [],
        resolution_diff_stat: undefined,
        resolver_model: undefined
      }
    }
    startCustomUpdatePolling()
  } catch (error: unknown) {
    const err = error as { response?: { data?: { message?: string } }; message?: string }
    customUpdateActionError.value =
      err.response?.data?.message || err.message || t('version.customUpdateRequestFailed')
  } finally {
    customUpdateSubmitting.value = false
  }
}

async function handleCustomUpdatePromotion() {
  const image = customUpdateStatus.value?.image
  if (customUpdateDemoActive.value || customUpdateSubmitting.value || !image) return

  customUpdateSubmitting.value = true
  customUpdateActionError.value = ''
  try {
    const result = await promoteCustomUpdate(image)
    if (customUpdateStatus.value) {
      customUpdateStatus.value = {
        ...customUpdateStatus.value,
        state: result.state,
        action: result.action,
        request_id: result.request_id,
        message: result.message,
        error: undefined
      }
    }
    promotionConfirmationVisible.value = false
    startCustomUpdatePolling()
  } catch (error: unknown) {
    const err = error as { response?: { data?: { message?: string } }; message?: string }
    customUpdateActionError.value =
      err.response?.data?.message || err.message || t('version.customUpdateRequestFailed')
  } finally {
    customUpdateSubmitting.value = false
  }
}

async function handleCustomUpdateResolutionAccept() {
  const resolutionId = customUpdateStatus.value?.resolution_id
  if (customUpdateDemoActive.value || customUpdateSubmitting.value || !resolutionId) return

  customUpdateSubmitting.value = true
  customUpdateActionError.value = ''
  try {
    const result = await acceptCustomUpdateResolution(resolutionId)
    if (customUpdateStatus.value) {
      customUpdateStatus.value = {
        ...customUpdateStatus.value,
        state: result.state,
        action: result.action,
        request_id: result.request_id,
        message: result.message,
        error: undefined,
        resolution_id: undefined,
        conflict_files: [],
        resolution_summary: undefined,
        resolution_risk_level: undefined,
        resolution_warnings: [],
        resolution_diff_stat: undefined,
        resolver_model: undefined
      }
    }
    resolutionConfirmationVisible.value = false
    startCustomUpdatePolling()
  } catch (error: unknown) {
    const err = error as { response?: { data?: { message?: string } }; message?: string }
    customUpdateActionError.value =
      err.response?.data?.message || err.message || t('version.customUpdateRequestFailed')
  } finally {
    customUpdateSubmitting.value = false
  }
}

async function handleCustomUpdateResolutionAbort() {
  const resolutionId = customUpdateStatus.value?.resolution_id
  if (customUpdateDemoActive.value || customUpdateSubmitting.value || !resolutionId) return

  customUpdateSubmitting.value = true
  customUpdateActionError.value = ''
  try {
    const result = await abortCustomUpdateResolution(resolutionId)
    if (customUpdateStatus.value) {
      customUpdateStatus.value = {
        ...customUpdateStatus.value,
        state: result.state,
        action: result.action,
        request_id: result.request_id,
        message: result.message,
        error: undefined,
        resolution_id: undefined,
        conflict_files: [],
        resolution_summary: undefined,
        resolution_risk_level: undefined,
        resolution_warnings: [],
        resolution_diff_stat: undefined,
        resolver_model: undefined
      }
    }
    resolutionConfirmationVisible.value = false
    startCustomUpdatePolling()
  } catch (error: unknown) {
    const err = error as { response?: { data?: { message?: string } }; message?: string }
    customUpdateActionError.value =
      err.response?.data?.message || err.message || t('version.customUpdateRequestFailed')
  } finally {
    customUpdateSubmitting.value = false
  }
}

function resetRollbackState() {
  rollbackPanelOpen.value = false
  rollbackVersions.value = []
  rollbackVersionsError.value = ''
  selectedRollbackVersion.value = ''
  rollbackError.value = ''
  manualTab.value = 'script'
}

async function toggleRollbackPanel() {
  if (!isAdmin.value) return
  rollbackPanelOpen.value = !rollbackPanelOpen.value
  // Source builds only show a hint, no version list to fetch
  if (
    rollbackPanelOpen.value &&
    isReleaseBuild.value &&
    rollbackVersions.value.length === 0 &&
    !rollbackVersionsLoading.value
  ) {
    await loadRollbackVersions()
  }
}

async function loadRollbackVersions() {
  if (!isAdmin.value) return
  rollbackVersionsLoading.value = true
  rollbackVersionsError.value = ''
  try {
    const data = await getRollbackVersions()
    rollbackVersions.value = data.versions || []
  } catch (error: unknown) {
    const err = error as { response?: { data?: { message?: string } }; message?: string }
    rollbackVersionsError.value =
      err.response?.data?.message || err.message || t('version.loadVersionsFailed')
  } finally {
    rollbackVersionsLoading.value = false
  }
}

function selectRollbackVersion(version: string) {
  if (rollingBack.value) return
  rollbackError.value = ''
  selectedRollbackVersion.value = selectedRollbackVersion.value === version ? '' : version
}

function formatPublishedAt(publishedAt: string): string {
  if (!publishedAt) return ''
  const date = new Date(publishedAt)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleDateString()
}

async function handleRollback() {
  if (!isAdmin.value) return
  if (rollingBack.value || !selectedRollbackVersion.value) return

  rollingBack.value = true
  rollbackError.value = ''

  try {
    const result = await rollbackAPI(selectedRollbackVersion.value)
    successKind.value = 'rollback'
    updateSuccess.value = true
    needRestart.value = result.need_restart
    rollbackPanelOpen.value = false
    // Clear version cache so the next check reflects the rolled-back version
    appStore.clearVersionCache()
  } catch (error: unknown) {
    const err = error as { response?: { data?: { message?: string } }; message?: string }
    rollbackError.value = err.response?.data?.message || err.message || t('version.rollbackFailed')
  } finally {
    rollingBack.value = false
  }
}

async function handleRestart() {
  if (restarting.value) return

  restarting.value = true
  restartCountdown.value = 8

  try {
    await restartService()
    // Service will restart, page will reload automatically or show disconnected
  } catch (error) {
    // Expected - connection will be lost during restart
    console.log('Service restarting...')
  }

  // Start countdown
  const countdownInterval = setInterval(() => {
    restartCountdown.value--
    if (restartCountdown.value <= 0) {
      clearInterval(countdownInterval)
      // Try to check if service is back before reload
      checkServiceAndReload()
    }
  }, 1000)
}

async function checkServiceAndReload() {
  const maxRetries = 5
  const retryDelay = 1000

  for (let i = 0; i < maxRetries; i++) {
    try {
      const response = await fetch('/health', {
        method: 'GET',
        cache: 'no-cache'
      })
      if (response.ok) {
        // Service is back, reload page
        window.location.reload()
        return
      }
    } catch {
      // Service not ready yet
    }

    if (i < maxRetries - 1) {
      await new Promise((resolve) => setTimeout(resolve, retryDelay))
    }
  }

  // After retries, reload anyway
  window.location.reload()
}

function handleClickOutside(event: MouseEvent) {
  const target = event.target as Node
  const button = (event.target as Element).closest('button')
  if (dropdownRef.value && !dropdownRef.value.contains(target) && !button?.contains(target)) {
    closeDropdown()
  }
}

onMounted(() => {
  if (isAdmin.value) {
    if (customUpdateDemoActive.value) {
      dropdownOpen.value = true
    }
    // Use cached version if available, otherwise fetch
    appStore.fetchVersion(false)
    void refreshCustomUpdateStatus()
  }
  document.addEventListener('click', handleClickOutside)
})

onBeforeUnmount(() => {
  stopCustomUpdatePolling()
  document.removeEventListener('click', handleClickOutside)
})
</script>

<style scoped>
.dropdown-enter-active,
.dropdown-leave-active {
  transition: all 0.2s ease;
}

.dropdown-enter-from,
.dropdown-leave-to {
  opacity: 0;
  transform: scale(0.95) translateY(-4px);
}

.rollback-enter-active,
.rollback-leave-active {
  transition: all 0.2s ease;
}

.rollback-enter-from,
.rollback-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

.line-clamp-3 {
  display: -webkit-box;
  -webkit-line-clamp: 3;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
</style>
