<template>
  <div class="mx-auto w-full max-w-6xl pb-12">
    <header class="flex flex-col gap-4 border-b border-gray-200 pb-6 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
      <div class="flex items-start gap-4">
        <div class="flex h-12 w-12 flex-none items-center justify-center rounded-lg bg-gray-900 text-white dark:bg-white dark:text-gray-900">
          <Icon name="terminal" size="lg" :stroke-width="2" />
        </div>
        <div>
          <h1 class="text-2xl font-bold text-gray-950 dark:text-white">
            {{ t('quickStart.title') }}
          </h1>
          <p class="mt-1 max-w-2xl text-sm leading-6 text-gray-600 dark:text-dark-300">
            {{ t('quickStart.subtitle') }}
          </p>
        </div>
      </div>
    </header>

    <nav class="mt-7" :aria-label="t('quickStart.title')">
      <ol class="grid grid-cols-3">
        <li
          v-for="(step, index) in steps"
          :key="step.id"
          class="relative flex justify-center"
        >
          <div
            v-if="index < steps.length - 1"
            class="absolute left-[calc(50%+24px)] right-[calc(-50%+24px)] top-5 h-px"
            :class="currentStep > index + 1 ? 'bg-primary-500' : 'bg-gray-200 dark:bg-dark-600'"
          ></div>
          <button
            type="button"
            class="relative z-10 flex min-w-0 flex-col items-center gap-2 px-2 text-center"
            :class="isStepAccessible(index + 1) ? 'cursor-pointer' : 'cursor-default'"
            :disabled="!isStepAccessible(index + 1)"
            @click="goToStep(index + 1)"
          >
            <span
              class="flex h-10 w-10 items-center justify-center rounded-full border text-sm font-semibold transition-colors"
              :class="stepCircleClass(index + 1)"
            >
              <Icon
                v-if="currentStep > index + 1"
                name="check"
                size="sm"
                :stroke-width="2.5"
              />
              <span v-else>{{ index + 1 }}</span>
            </span>
            <span
              class="text-xs font-medium sm:text-sm"
              :class="currentStep === index + 1
                ? 'text-gray-950 dark:text-white'
                : 'text-gray-500 dark:text-dark-400'"
            >
              {{ step.label }}
            </span>
          </button>
        </li>
      </ol>
    </nav>

    <main class="mt-9 min-h-[440px]">
      <section v-if="currentStep === 1">
        <div class="max-w-2xl">
          <h2 class="text-xl font-semibold text-gray-950 dark:text-white">
            {{ t('quickStart.clientStep.title') }}
          </h2>
          <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
            {{ t('quickStart.clientStep.description') }}
          </p>
        </div>

        <div class="mt-6 grid gap-4 md:grid-cols-3">
          <button
            v-for="client in clients"
            :key="client.id"
            type="button"
            class="group flex min-h-[148px] flex-col items-start rounded-lg border bg-white p-5 text-left transition-colors dark:bg-dark-900"
            :class="selectedClient === client.id
              ? 'border-primary-500 ring-2 ring-primary-100 dark:ring-primary-900/50'
              : 'border-gray-200 hover:border-gray-400 dark:border-dark-600 dark:hover:border-dark-400'"
            :aria-pressed="selectedClient === client.id"
            @click="selectClient(client.id)"
          >
            <span
              class="flex h-10 w-10 items-center justify-center rounded-lg"
              :class="client.iconClass"
            >
              <Icon :name="client.icon" size="md" :stroke-width="2" />
            </span>
            <span class="mt-4 text-base font-semibold text-gray-950 dark:text-white">
              {{ t(client.nameKey) }}
            </span>
            <span class="mt-1 text-sm leading-5 text-gray-500 dark:text-dark-400">
              {{ t(client.descriptionKey) }}
            </span>
          </button>
        </div>

        <div class="mt-8 max-w-3xl">
          <h3 class="text-base font-semibold text-gray-950 dark:text-white">
            {{ t('quickStart.clientStep.methodTitle') }}
          </h3>
          <p class="mt-1 text-sm leading-6 text-gray-600 dark:text-dark-300">
            {{ t('quickStart.clientStep.methodDescription') }}
          </p>

          <div
            class="mt-4 inline-grid w-full grid-cols-2 rounded-lg bg-gray-100 p-1 dark:bg-dark-800 sm:w-auto"
            role="radiogroup"
            :aria-label="t('quickStart.clientStep.methodTitle')"
          >
            <button
              v-for="method in installMethods"
              :key="method.id"
              type="button"
              class="flex min-h-11 items-center justify-center gap-2 rounded-md px-4 py-2 text-sm font-medium transition-colors"
              :class="selectedInstallMethod === method.id
                ? 'bg-white text-gray-950 shadow-sm dark:bg-dark-700 dark:text-white'
                : 'text-gray-500 hover:text-gray-800 dark:text-dark-300 dark:hover:text-white'"
              role="radio"
              :aria-checked="selectedInstallMethod === method.id"
              @click="selectedInstallMethod = method.id"
            >
              <Icon :name="method.icon" size="sm" :stroke-width="2" />
              <span>{{ t(method.nameKey) }}</span>
            </button>
          </div>

          <p class="mt-3 text-sm text-gray-500 dark:text-dark-400">
            {{ selectedInstallMethodConfig
              ? t(selectedInstallMethodConfig.descriptionKey)
              : '' }}
          </p>
        </div>

        <div class="mt-8 flex justify-end border-t border-gray-200 pt-5 dark:border-dark-700">
          <button
            type="button"
            class="btn btn-primary min-w-28"
            :disabled="!selectedClient"
            @click="continueFromClient"
          >
            {{ t('quickStart.actions.continue') }}
            <Icon name="arrowRight" size="sm" class="ml-2" :stroke-width="2" />
          </button>
        </div>
      </section>

      <section v-else-if="currentStep === 2">
        <div class="max-w-2xl">
          <h2 class="text-xl font-semibold text-gray-950 dark:text-white">
            {{ t('quickStart.keyStep.title') }}
          </h2>
          <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
            {{ t('quickStart.keyStep.description') }}
          </p>
        </div>

        <div v-if="setupLoading" class="flex min-h-64 items-center justify-center">
          <div class="text-center">
            <div class="mx-auto h-7 w-7 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
            <p class="mt-3 text-sm text-gray-500 dark:text-dark-400">
              {{ t('quickStart.keyStep.loading') }}
            </p>
          </div>
        </div>

        <div
          v-else-if="setupError"
          class="mt-6 rounded-lg border border-red-200 bg-red-50 p-5 dark:border-red-900/60 dark:bg-red-950/30"
        >
          <div class="flex items-start gap-3">
            <Icon name="exclamationCircle" size="md" class="mt-0.5 flex-none text-red-600 dark:text-red-400" />
            <div>
              <p class="text-sm font-medium text-red-800 dark:text-red-200">
                {{ t('quickStart.keyStep.loadFailed') }}
              </p>
              <button type="button" class="btn btn-secondary btn-sm mt-4" @click="loadSetupData">
                <Icon name="refresh" size="sm" class="mr-2" />
                {{ t('quickStart.keyStep.retry') }}
              </button>
            </div>
          </div>
        </div>

        <template v-else>
          <div
            v-if="!hasFundedCompatibleGroup"
            class="mt-6 rounded-lg border border-amber-200 bg-amber-50 p-5 dark:border-amber-900/60 dark:bg-amber-950/20"
          >
            <div class="flex items-start gap-3">
              <Icon name="exclamationTriangle" size="md" class="mt-0.5 flex-none text-amber-600 dark:text-amber-400" />
              <div>
                <p class="text-sm font-semibold text-amber-900 dark:text-amber-100">
                  {{ t('quickStart.keyStep.noFundedGroup') }}
                </p>
                <p class="mt-1 text-sm text-amber-800 dark:text-amber-200">
                  {{ t('quickStart.keyStep.noCredit') }}
                </p>
                <div class="mt-4 flex flex-wrap gap-3">
                  <RouterLink to="/purchase" class="btn btn-primary btn-sm">
                    {{ t('quickStart.keyStep.recharge') }}
                  </RouterLink>
                  <RouterLink to="/purchase?tab=subscription" class="btn btn-secondary btn-sm">
                    {{ t('quickStart.keyStep.subscribe') }}
                  </RouterLink>
                </div>
              </div>
            </div>
          </div>

          <template v-else>
            <div class="mt-6 inline-flex rounded-lg bg-gray-100 p-1 dark:bg-dark-800" role="tablist">
              <button
                type="button"
                class="rounded-md px-4 py-2 text-sm font-medium transition-colors"
                :class="keyMode === 'existing'
                  ? 'bg-white text-gray-950 shadow-sm dark:bg-dark-700 dark:text-white'
                  : 'text-gray-500 hover:text-gray-800 dark:text-dark-300 dark:hover:text-white'"
                role="tab"
                :aria-selected="keyMode === 'existing'"
                @click="keyMode = 'existing'"
              >
                {{ t('quickStart.keyStep.existing') }}
              </button>
              <button
                type="button"
                class="rounded-md px-4 py-2 text-sm font-medium transition-colors"
                :class="keyMode === 'create'
                  ? 'bg-white text-gray-950 shadow-sm dark:bg-dark-700 dark:text-white'
                  : 'text-gray-500 hover:text-gray-800 dark:text-dark-300 dark:hover:text-white'"
                role="tab"
                :aria-selected="keyMode === 'create'"
                @click="keyMode = 'create'"
              >
                {{ t('quickStart.keyStep.create') }}
              </button>
            </div>

            <div v-if="keyMode === 'existing'" class="mt-5">
              <div
                v-if="eligibleKeys.length === 0"
                class="rounded-lg border border-dashed border-gray-300 px-5 py-10 text-center dark:border-dark-600"
              >
                <Icon name="key" size="lg" class="mx-auto text-gray-400 dark:text-dark-400" />
                <p class="mt-3 text-sm text-gray-600 dark:text-dark-300">
                  {{ t('quickStart.keyStep.noExisting') }}
                </p>
                <button type="button" class="btn btn-secondary btn-sm mt-4" @click="keyMode = 'create'">
                  <Icon name="plus" size="sm" class="mr-2" />
                  {{ t('quickStart.keyStep.create') }}
                </button>
              </div>

              <div v-else class="grid gap-3 lg:grid-cols-2">
                <button
                  v-for="key in eligibleKeys"
                  :key="key.id"
                  type="button"
                  class="flex min-h-[104px] items-start justify-between gap-4 rounded-lg border bg-white p-4 text-left transition-colors dark:bg-dark-900"
                  :class="selectedKeyId === key.id
                    ? 'border-primary-500 ring-2 ring-primary-100 dark:ring-primary-900/50'
                    : 'border-gray-200 hover:border-gray-400 dark:border-dark-600 dark:hover:border-dark-400'"
                  :aria-pressed="selectedKeyId === key.id"
                  @click="selectedKeyId = key.id"
                >
                  <span class="min-w-0">
                    <span class="block truncate text-sm font-semibold text-gray-950 dark:text-white">
                      {{ key.name }}
                    </span>
                    <span class="mt-1 block font-mono text-xs text-gray-500 dark:text-dark-400">
                      {{ maskQuickStartKey(key.key) }}
                    </span>
                    <span class="mt-3 flex flex-wrap items-center gap-2 text-xs text-gray-500 dark:text-dark-400">
                      <span>{{ groupForKey(key)?.name }}</span>
                      <span aria-hidden="true">·</span>
                      <span>{{ rateLabel(groupForKey(key)) }}</span>
                    </span>
                  </span>
                  <span
                    class="mt-0.5 flex h-5 w-5 flex-none items-center justify-center rounded-full border"
                    :class="selectedKeyId === key.id
                      ? 'border-primary-500 bg-primary-500 text-white'
                      : 'border-gray-300 dark:border-dark-500'"
                  >
                    <Icon v-if="selectedKeyId === key.id" name="check" size="xs" :stroke-width="3" />
                  </span>
                </button>
              </div>
            </div>

            <div
              v-else
              class="mt-5 rounded-lg border border-gray-200 bg-gray-50 p-5 dark:border-dark-600 dark:bg-dark-900"
            >
              <div class="grid gap-5 md:grid-cols-2">
                <label class="block">
                  <span class="mb-2 block text-sm font-medium text-gray-800 dark:text-dark-200">
                    {{ t('quickStart.keyStep.createName') }}
                  </span>
                  <input
                    v-model="createKeyName"
                    type="text"
                    maxlength="100"
                    class="input w-full"
                    :placeholder="t('quickStart.keyStep.createNamePlaceholder')"
                  />
                </label>
                <label class="block">
                  <span class="mb-2 block text-sm font-medium text-gray-800 dark:text-dark-200">
                    {{ t('quickStart.keyStep.createGroup') }}
                  </span>
                  <select v-model.number="createGroupId" class="input w-full">
                    <option :value="null" disabled>
                      {{ t('quickStart.keyStep.chooseGroup') }}
                    </option>
                    <option
                      v-for="group in fundedCompatibleGroups"
                      :key="group.id"
                      :value="group.id"
                    >
                      {{ group.name }} · {{ rateLabel(group) }}
                    </option>
                  </select>
                </label>
              </div>
            </div>
          </template>
        </template>

        <div class="mt-8 flex flex-col-reverse gap-3 border-t border-gray-200 pt-5 dark:border-dark-700 sm:flex-row sm:items-center sm:justify-between">
          <button type="button" class="btn btn-secondary" @click="goToStep(1)">
            <Icon name="arrowLeft" size="sm" class="mr-2" />
            {{ t('quickStart.actions.back') }}
          </button>
          <button
            v-if="hasFundedCompatibleGroup"
            type="button"
            class="btn btn-primary"
            :disabled="!canContinueFromKey || creatingKey || issuingToken"
            @click="continueFromKey"
          >
            <span v-if="creatingKey">{{ t('quickStart.actions.creating') }}</span>
            <span v-else-if="keyMode === 'create'">{{ t('quickStart.actions.createAndContinue') }}</span>
            <span v-else>{{ t('quickStart.actions.continue') }}</span>
            <Icon v-if="!creatingKey" name="arrowRight" size="sm" class="ml-2" />
          </button>
        </div>
      </section>

      <section v-else>
        <div v-if="issuingToken && !installSession" class="flex min-h-80 items-center justify-center">
          <div class="text-center">
            <div class="mx-auto h-8 w-8 animate-spin rounded-full border-2 border-primary-500 border-t-transparent"></div>
            <p class="mt-3 text-sm text-gray-500 dark:text-dark-400">
              {{ t('quickStart.installStep.issuing') }}
            </p>
          </div>
        </div>

        <template v-else>
          <div class="flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
            <div class="max-w-2xl">
              <h2 class="text-xl font-semibold text-gray-950 dark:text-white">
                {{ installStepTitle }}
              </h2>
              <p class="mt-2 text-sm leading-6 text-gray-600 dark:text-dark-300">
                {{ installStepDescription }}
              </p>
            </div>
            <button type="button" class="btn btn-secondary btn-sm flex-none" @click="goToStep(2)">
              <Icon name="arrowLeft" size="sm" class="mr-2" />
              {{ t('quickStart.actions.back') }}
            </button>
          </div>

          <div
            v-if="installError"
            class="mt-6 rounded-lg border border-red-200 bg-red-50 p-5 dark:border-red-900/60 dark:bg-red-950/30"
          >
            <div class="flex items-start gap-3">
              <Icon name="exclamationCircle" size="md" class="mt-0.5 flex-none text-red-600 dark:text-red-400" />
              <div class="min-w-0">
                <p class="text-sm font-medium text-red-800 dark:text-red-200">
                  {{ installError }}
                </p>
                <button
                  v-if="!installSession"
                  type="button"
                  class="btn btn-secondary btn-sm mt-4"
                  :disabled="issuingToken"
                  @click="issueSelectedKey"
                >
                  <Icon name="refresh" size="sm" class="mr-2" />
                  {{ t('quickStart.keyStep.retry') }}
                </button>
              </div>
            </div>
          </div>

          <template v-if="installSession">
            <div class="mt-6 flex flex-wrap items-center gap-x-6 gap-y-3 border-y border-gray-200 py-4 text-sm dark:border-dark-700">
              <div class="flex items-center gap-2">
                <span class="text-gray-500 dark:text-dark-400">
                  {{ t('quickStart.installStep.selectedClient') }}
                </span>
                <span class="font-semibold text-gray-950 dark:text-white">
                  {{ selectedClientLabel }}
                </span>
              </div>
              <div class="flex items-center gap-2">
                <span class="text-gray-500 dark:text-dark-400">
                  {{ t('quickStart.installStep.selectedMethod') }}
                </span>
                <span class="font-semibold text-gray-950 dark:text-white">
                  {{ selectedInstallMethodLabel }}
                </span>
              </div>
              <div class="flex min-w-0 items-center gap-2">
                <span class="text-gray-500 dark:text-dark-400">
                  {{ t('quickStart.installStep.selectedKey') }}
                </span>
                <span class="truncate font-semibold text-gray-950 dark:text-white">
                  {{ installSession.key.name }}
                </span>
                <code class="font-mono text-xs text-gray-500 dark:text-dark-400">
                  {{ installSession.key.prefix }}
                </code>
              </div>
              <div class="ml-auto text-xs text-gray-500 dark:text-dark-400">
                {{ t('quickStart.installStep.expiresAt', { time: formatDate(installSession.expires_at) }) }}
              </div>
            </div>

            <div
              v-if="installExpired"
              class="mt-5 flex items-center gap-3 rounded-lg border border-amber-200 bg-amber-50 px-4 py-3 text-sm text-amber-900 dark:border-amber-900/60 dark:bg-amber-950/20 dark:text-amber-100"
            >
              <Icon name="clock" size="sm" class="flex-none" />
              {{ t('quickStart.installStep.tokenExpired') }}
            </div>

            <template v-if="selectedInstallMethod === 'command'">
              <div class="mt-5 overflow-hidden rounded-lg border border-gray-800 bg-gray-950">
                <div class="flex items-center justify-between border-b border-gray-800 px-3 py-2">
                  <div class="inline-flex rounded-md bg-gray-900 p-1" role="tablist">
                    <button
                      type="button"
                      class="rounded px-3 py-1.5 text-xs font-medium transition-colors"
                      :class="commandPlatform === 'unix'
                        ? 'bg-gray-700 text-white'
                        : 'text-gray-400 hover:text-white'"
                      role="tab"
                      :aria-selected="commandPlatform === 'unix'"
                      @click="commandPlatform = 'unix'"
                    >
                      {{ t('quickStart.installStep.unix') }}
                    </button>
                    <button
                      type="button"
                      class="rounded px-3 py-1.5 text-xs font-medium transition-colors"
                      :class="commandPlatform === 'windows'
                        ? 'bg-gray-700 text-white'
                        : 'text-gray-400 hover:text-white'"
                      role="tab"
                      :aria-selected="commandPlatform === 'windows'"
                      @click="commandPlatform = 'windows'"
                    >
                      {{ t('quickStart.installStep.windows') }}
                    </button>
                  </div>
                  <button
                    type="button"
                    class="flex h-8 w-8 items-center justify-center rounded-md text-gray-400 transition-colors hover:bg-gray-800 hover:text-white"
                    :title="t('quickStart.installStep.copy')"
                    :aria-label="t('quickStart.installStep.copy')"
                    @click="copyCommand"
                  >
                    <Icon name="copy" size="sm" :stroke-width="2" />
                  </button>
                </div>
                <div class="min-h-[112px] overflow-x-auto p-5">
                  <code class="whitespace-pre font-mono text-sm leading-6 text-emerald-300">
                    {{ currentCommand }}
                  </code>
                </div>
              </div>

              <div class="mt-4 flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
                <a
                  :href="installSession.fallback_url"
                  target="_blank"
                  rel="noopener noreferrer"
                  class="inline-flex items-center text-sm font-medium text-gray-600 hover:text-gray-950 dark:text-dark-300 dark:hover:text-white"
                >
                  {{ t('quickStart.installStep.fallback') }}
                  <Icon name="externalLink" size="sm" class="ml-1.5" />
                </a>
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="issuingToken"
                  @click="refreshCommand"
                >
                  <Icon name="refresh" size="sm" class="mr-2" :class="{ 'animate-spin': issuingToken }" />
                  {{ issuingToken ? t('quickStart.installStep.refreshing') : t('quickStart.installStep.refresh') }}
                </button>
              </div>
            </template>

            <template v-else>
              <div class="mt-5 border-y border-gray-200 py-7 dark:border-dark-700">
                <div class="flex flex-col items-start gap-5 sm:flex-row sm:items-center sm:justify-between">
                  <div class="flex min-w-0 items-start gap-4">
                    <span class="flex h-11 w-11 flex-none items-center justify-center rounded-lg bg-emerald-100 text-emerald-700 dark:bg-emerald-950/50 dark:text-emerald-300">
                      <Icon name="sync" size="md" :stroke-width="2" />
                    </span>
                    <div class="min-w-0">
                      <h3 class="font-semibold text-gray-950 dark:text-white">
                        {{ t('quickStart.installStep.ccSwitchActionTitle') }}
                      </h3>
                      <p class="mt-1 text-sm leading-6 text-gray-600 dark:text-dark-300">
                        {{ t('quickStart.installStep.ccSwitchActionDescription') }}
                      </p>
                    </div>
                  </div>
                  <a
                    :href="installSession.fallback_url"
                    target="_blank"
                    rel="noopener noreferrer"
                    class="btn btn-primary flex-none"
                  >
                    <Icon name="externalLink" size="sm" class="mr-2" />
                    {{ t('quickStart.installStep.openCcSwitch') }}
                  </a>
                </div>
                <p class="mt-4 text-xs leading-5 text-gray-500 dark:text-dark-400">
                  {{ t('quickStart.installStep.ccSwitchOnlyConfigures') }}
                </p>
              </div>

              <div class="mt-4 flex justify-end">
                <button
                  type="button"
                  class="btn btn-secondary btn-sm"
                  :disabled="issuingToken"
                  @click="refreshCommand"
                >
                  <Icon name="refresh" size="sm" class="mr-2" :class="{ 'animate-spin': issuingToken }" />
                  {{ issuingToken ? t('quickStart.installStep.refreshing') : t('quickStart.installStep.refresh') }}
                </button>
              </div>
            </template>

            <div class="mt-8 border-t border-gray-200 pt-6 dark:border-dark-700">
              <h3 class="text-sm font-semibold uppercase text-gray-500 dark:text-dark-400">
                {{ installActionsTitle }}
              </h3>
              <div
                class="mt-4 grid gap-3 sm:grid-cols-2"
                :class="selectedInstallMethod === 'command' ? 'lg:grid-cols-4' : 'lg:grid-cols-3'"
              >
                <div
                  v-for="item in installActions"
                  :key="item"
                  class="flex items-start gap-2 text-sm leading-5 text-gray-700 dark:text-dark-200"
                >
                  <Icon name="checkCircle" size="sm" class="mt-0.5 flex-none text-emerald-600 dark:text-emerald-400" />
                  <span>{{ item }}</span>
                </div>
              </div>
            </div>

            <div class="mt-8 flex justify-end">
              <button type="button" class="btn btn-primary" @click="showNextSteps = true">
                <Icon name="check" size="sm" class="mr-2" :stroke-width="2.5" />
                {{ selectedInstallMethod === 'command'
                  ? t('quickStart.installStep.done')
                  : t('quickStart.installStep.importDone') }}
              </button>
            </div>

            <div
              v-if="showNextSteps"
              class="mt-5 rounded-lg border border-emerald-200 bg-emerald-50 p-5 dark:border-emerald-900/60 dark:bg-emerald-950/20"
            >
              <h3 class="font-semibold text-emerald-950 dark:text-emerald-100">
                {{ t('quickStart.installStep.nextTitle') }}
              </h3>
              <ol class="mt-3 space-y-2 text-sm leading-6 text-emerald-900 dark:text-emerald-200">
                <li v-for="(item, index) in nextSteps" :key="item">
                  {{ index + 1 }}. {{ item }}
                </li>
              </ol>
            </div>
          </template>
        </template>
      </section>
    </main>

    <section class="mt-10 border-t border-gray-200 pt-6 dark:border-dark-700">
      <button
        type="button"
        class="flex w-full items-center justify-between gap-4 py-2 text-left"
        :aria-expanded="faqOpen"
        @click="toggleFaq"
      >
        <span>
          <span class="block text-base font-semibold text-gray-950 dark:text-white">
            {{ t('quickStart.faq.title') }}
          </span>
          <span class="mt-1 block text-sm text-gray-500 dark:text-dark-400">
            {{ t('quickStart.faq.description') }}
          </span>
        </span>
        <Icon
          :name="faqOpen ? 'chevronUp' : 'chevronDown'"
          size="sm"
          class="flex-none text-gray-500 dark:text-dark-400"
        />
      </button>

      <div v-if="faqOpen" class="mt-5">
        <div v-if="faqLoading" class="py-8 text-center text-sm text-gray-500 dark:text-dark-400">
          {{ t('quickStart.faq.loading') }}
        </div>
        <div
          v-else-if="faqError"
          class="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-200"
        >
          {{ t('quickStart.faq.failed') }}
        </div>
        <div
          v-else
          class="quickstart-faq-content overflow-hidden rounded-lg border border-gray-200 bg-white p-5 text-gray-800 dark:border-dark-600 dark:bg-dark-900 dark:text-dark-100 md:p-7"
          v-html="faqHtml"
        ></div>
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { RouterLink } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
import type { ApiKey, Group } from '@/types'
import type { InstallClient, InstallTokenIssueResult } from '@/api/installTokens'
import { keysAPI, installTokensAPI, userGroupsAPI } from '@/api'
import { buildApiUrl } from '@/api/client'
import { useAuthStore } from '@/stores/auth'
import { useAppStore } from '@/stores/app'
import { useClipboard } from '@/composables/useClipboard'
import Icon from '@/components/icons/Icon.vue'
import {
  buildInstallTokenIssueRequest,
  isQuickStartPlatformCompatible,
  maskQuickStartKey
} from '@/utils/quickstart'

const props = withDefaults(defineProps<{
  faqSlug?: string
}>(), {
  faqSlug: 'codex-claude-import'
})

const { t, locale } = useI18n()
const authStore = useAuthStore()
const appStore = useAppStore()
const { copyToClipboard } = useClipboard()

type InstallMethod = 'command' | 'cc-switch'

const currentStep = ref(1)
const maxReachedStep = ref(1)
const selectedClient = ref<InstallClient | ''>('')
const selectedInstallMethod = ref<InstallMethod>('command')
const keyMode = ref<'existing' | 'create'>('existing')
const selectedKeyId = ref<number | null>(null)
const createKeyName = ref('')
const createGroupId = ref<number | null>(null)
const apiKeys = ref<ApiKey[]>([])
const groups = ref<Group[]>([])
const userGroupRates = ref<Record<number, number>>({})
const setupLoading = ref(false)
const setupLoaded = ref(false)
const setupError = ref('')
const creatingKey = ref(false)
const issuingToken = ref(false)
const installError = ref('')
const installSession = ref<InstallTokenIssueResult | null>(null)
const commandPlatform = ref<'unix' | 'windows'>('unix')
const showNextSteps = ref(false)
const now = ref(Date.now())
const faqOpen = ref(false)
const faqLoading = ref(false)
const faqLoaded = ref(false)
const faqError = ref('')
const faqHtml = ref('')
let clockTimer: ReturnType<typeof setInterval> | null = null

const steps = computed(() => [
  { id: 'client', label: t('quickStart.steps.client') },
  { id: 'key', label: t('quickStart.steps.key') },
  { id: 'install', label: t('quickStart.steps.install') }
])

const clients = computed(() => [
  {
    id: 'claude-code' as const,
    nameKey: 'quickStart.clientStep.claude',
    descriptionKey: 'quickStart.clientStep.claudeDescription',
    icon: 'terminal' as const,
    iconClass: 'bg-orange-100 text-orange-700 dark:bg-orange-950/50 dark:text-orange-300'
  },
  {
    id: 'codex' as const,
    nameKey: 'quickStart.clientStep.codex',
    descriptionKey: 'quickStart.clientStep.codexDescription',
    icon: 'cpu' as const,
    iconClass: 'bg-gray-900 text-white dark:bg-gray-100 dark:text-gray-900'
  },
  {
    id: 'gemini-cli' as const,
    nameKey: 'quickStart.clientStep.gemini',
    descriptionKey: 'quickStart.clientStep.geminiDescription',
    icon: 'sparkles' as const,
    iconClass: 'bg-sky-100 text-sky-700 dark:bg-sky-950/50 dark:text-sky-300'
  }
])

const installMethods = computed(() => [
  {
    id: 'command' as const,
    nameKey: 'quickStart.clientStep.commandMethod',
    descriptionKey: 'quickStart.clientStep.commandMethodDescription',
    icon: 'terminal' as const
  },
  {
    id: 'cc-switch' as const,
    nameKey: 'quickStart.clientStep.ccSwitchMethod',
    descriptionKey: 'quickStart.clientStep.ccSwitchMethodDescription',
    icon: 'sync' as const
  }
])

const groupMap = computed(() => new Map(groups.value.map((group) => [group.id, group])))

const compatibleGroups = computed(() => {
  if (!selectedClient.value) return []
  return groups.value.filter((group) =>
    group.status === 'active' &&
    isQuickStartPlatformCompatible(selectedClient.value as InstallClient, group.platform)
  )
})

const availableBalance = computed(() => {
  const balance = authStore.user?.balance ?? 0
  const frozen = authStore.user?.frozen_balance ?? 0
  return Math.max(0, balance - frozen)
})

function isGroupFunded(group: Group): boolean {
  return authStore.isSimpleMode || group.subscription_type === 'subscription' || availableBalance.value > 0
}

const fundedCompatibleGroups = computed(() => compatibleGroups.value.filter(isGroupFunded))
const fundedGroupIds = computed(() => new Set(fundedCompatibleGroups.value.map((group) => group.id)))
const hasFundedCompatibleGroup = computed(() => fundedCompatibleGroups.value.length > 0)

function groupForKey(key: ApiKey): Group | undefined {
  return key.group || (key.group_id ? groupMap.value.get(key.group_id) : undefined)
}

function isKeyUsable(key: ApiKey): boolean {
  if (key.status !== 'active' || !key.group_id || !fundedGroupIds.value.has(key.group_id)) {
    return false
  }
  if (key.quota > 0 && key.quota_used >= key.quota) {
    return false
  }
  if (key.expires_at && Date.parse(key.expires_at) <= now.value) {
    return false
  }
  const group = groupForKey(key)
  return !!group && !!selectedClient.value &&
    isQuickStartPlatformCompatible(selectedClient.value as InstallClient, group.platform)
}

const eligibleKeys = computed(() => apiKeys.value.filter(isKeyUsable))

const selectedKey = computed(() =>
  apiKeys.value.find((key) => key.id === selectedKeyId.value) || null
)

const canContinueFromKey = computed(() => {
  if (!hasFundedCompatibleGroup.value) return false
  if (keyMode.value === 'existing') {
    return selectedKeyId.value !== null && eligibleKeys.value.some((key) => key.id === selectedKeyId.value)
  }
  return createKeyName.value.trim().length > 0 &&
    createGroupId.value !== null &&
    fundedGroupIds.value.has(createGroupId.value)
})

const selectedClientConfig = computed(() =>
  clients.value.find((client) => client.id === selectedClient.value)
)

const selectedClientLabel = computed(() =>
  selectedClientConfig.value ? t(selectedClientConfig.value.nameKey) : ''
)

const selectedInstallMethodConfig = computed(() =>
  installMethods.value.find((method) => method.id === selectedInstallMethod.value)
)

const selectedInstallMethodLabel = computed(() =>
  selectedInstallMethodConfig.value ? t(selectedInstallMethodConfig.value.nameKey) : ''
)

const installStepTitle = computed(() =>
  t(selectedInstallMethod.value === 'command'
    ? 'quickStart.installStep.title'
    : 'quickStart.installStep.ccSwitchTitle')
)

const installStepDescription = computed(() =>
  t(selectedInstallMethod.value === 'command'
    ? 'quickStart.installStep.description'
    : 'quickStart.installStep.ccSwitchDescription')
)

const currentCommand = computed(() => {
  if (!installSession.value) return ''
  return installSession.value.commands[commandPlatform.value]
})

const installExpired = computed(() =>
  !!installSession.value && Date.parse(installSession.value.expires_at) <= now.value
)

const installSessionMatchesSelection = computed(() =>
  !!installSession.value &&
  keyMode.value === 'existing' &&
  installSession.value.client === selectedClient.value &&
  installSession.value.key.id === selectedKeyId.value
)

const installActionsTitle = computed(() =>
  t(selectedInstallMethod.value === 'command'
    ? 'quickStart.installStep.scriptDoes'
    : 'quickStart.installStep.ccSwitchDoes')
)

const installActions = computed(() =>
  selectedInstallMethod.value === 'command'
    ? [
        t('quickStart.installStep.runtime'),
        t('quickStart.installStep.cli'),
        t('quickStart.installStep.ccSwitch'),
        t('quickStart.installStep.import')
      ]
    : [
        t('quickStart.installStep.ccSwitchValidate'),
        t('quickStart.installStep.ccSwitchOpen'),
        t('quickStart.installStep.ccSwitchImport')
      ]
)

const nextSteps = computed(() =>
  selectedInstallMethod.value === 'command'
    ? [
        t('quickStart.installStep.nextFinishCommand'),
        t('quickStart.installStep.nextRestart', { client: selectedClientLabel.value }),
        t('quickStart.installStep.nextPrompt')
      ]
    : [
        t('quickStart.installStep.nextCheck'),
        t('quickStart.installStep.nextRestart', { client: selectedClientLabel.value }),
        t('quickStart.installStep.nextPrompt')
      ]
)

function stepCircleClass(step: number): string {
  if (currentStep.value > step) {
    return 'border-primary-500 bg-primary-500 text-white'
  }
  if (currentStep.value === step) {
    return 'border-primary-500 bg-white text-primary-600 ring-4 ring-primary-100 dark:bg-dark-900 dark:text-primary-400 dark:ring-primary-900/40'
  }
  return 'border-gray-200 bg-gray-100 text-gray-500 dark:border-dark-600 dark:bg-dark-800 dark:text-dark-400'
}

function selectClient(client: InstallClient) {
  if (selectedClient.value === client) return
  selectedClient.value = client
  selectedKeyId.value = null
  createGroupId.value = null
  createKeyName.value = defaultKeyName(client)
  if (setupLoaded.value) {
    keyMode.value = eligibleKeys.value.length > 0 ? 'existing' : 'create'
  }
  showNextSteps.value = false
}

function defaultKeyName(client: InstallClient): string {
  const label = clients.value.find((item) => item.id === client)
  return label ? `${t(label.nameKey)} ${t('quickStart.title')}` : t('quickStart.title')
}

function isStepAccessible(step: number): boolean {
  if (step < 1 || step > maxReachedStep.value) return false
  return step !== 3 || currentStep.value === 3 || installSessionMatchesSelection.value
}

function goToStep(step: number) {
  if (!isStepAccessible(step)) return
  currentStep.value = step
}

async function continueFromClient() {
  if (!selectedClient.value) return
  currentStep.value = 2
  maxReachedStep.value = Math.max(maxReachedStep.value, 2)
  if (!setupLoaded.value && !setupLoading.value) {
    await loadSetupData()
  }
}

async function loadSetupData() {
  setupLoading.value = true
  setupError.value = ''
  try {
    const refreshUser = authStore.refreshUser().catch(() => null)
    const [keys, availableGroups, rates] = await Promise.all([
      keysAPI.list(1, 1000, { sort_by: 'created_at', sort_order: 'desc' }),
      userGroupsAPI.getAvailable(),
      userGroupsAPI.getUserGroupRates(),
      refreshUser
    ])
    apiKeys.value = keys.items || []
    groups.value = availableGroups || []
    userGroupRates.value = rates || {}
    setupLoaded.value = true
    normalizeKeySelections()
  } catch (error) {
    setupError.value = errorMessage(error)
  } finally {
    setupLoading.value = false
  }
}

function normalizeKeySelections() {
  if (!selectedClient.value) return
  const keys = eligibleKeys.value
  if (!keys.some((key) => key.id === selectedKeyId.value)) {
    selectedKeyId.value = keys[0]?.id ?? null
  }
  if (!fundedGroupIds.value.has(createGroupId.value || 0)) {
    createGroupId.value = fundedCompatibleGroups.value[0]?.id ?? null
  }
  if (keys.length === 0 && keyMode.value === 'existing') {
    keyMode.value = 'create'
  }
}

function rateLabel(group: Group | undefined): string {
  if (!group) return ''
  const rate = userGroupRates.value[group.id] ?? group.rate_multiplier
  const normalized = Number.isInteger(rate) ? rate.toFixed(0) : rate.toFixed(2).replace(/0+$/, '').replace(/\.$/, '')
  return t('quickStart.keyStep.rate', { rate: normalized })
}

async function continueFromKey() {
  if (!canContinueFromKey.value || !selectedClient.value) return
  if (keyMode.value === 'create') {
    await createKeyAndContinue()
    return
  }
  await issueSelectedKey()
}

async function createKeyAndContinue() {
  if (!selectedClient.value || !createGroupId.value || !createKeyName.value.trim()) return
  creatingKey.value = true
  installError.value = ''
  try {
    const group = groupMap.value.get(createGroupId.value)
    const created = await keysAPI.create(createKeyName.value.trim(), createGroupId.value)
    const hydrated: ApiKey = {
      ...created,
      group
    }
    apiKeys.value = [hydrated, ...apiKeys.value.filter((key) => key.id !== hydrated.id)]
    selectedKeyId.value = hydrated.id
    keyMode.value = 'existing'
    await issueSelectedKey()
  } catch (error) {
    appStore.showError(errorMessage(error))
  } finally {
    creatingKey.value = false
  }
}

function previousToken(): string | undefined {
  if (!installSession.value || installExpired.value) return undefined
  return installSession.value.token
}

async function issueSelectedKey() {
  if (!selectedClient.value || !selectedKey.value) return
  currentStep.value = 3
  maxReachedStep.value = 3
  issuingToken.value = true
  installError.value = ''
  showNextSteps.value = false
  const priorToken = previousToken()
  if (priorToken) {
    installSession.value = null
  }
  try {
    installSession.value = await installTokensAPI.issue(
      buildInstallTokenIssueRequest(selectedClient.value, selectedKey.value.id, priorToken)
    )
    now.value = Date.now()
  } catch (error) {
    installSession.value = null
    installError.value = errorMessage(error, 'quickStart.installStep.issueFailed')
  } finally {
    issuingToken.value = false
  }
}

async function refreshCommand() {
  if (!selectedClient.value || !selectedKey.value || issuingToken.value) return
  const priorToken = previousToken()
  issuingToken.value = true
  installError.value = ''
  showNextSteps.value = false
  try {
    installSession.value = await installTokensAPI.issue(
      buildInstallTokenIssueRequest(selectedClient.value, selectedKey.value.id, priorToken)
    )
    now.value = Date.now()
    appStore.showSuccess(t('quickStart.installStep.refresh'))
  } catch (error) {
    if (priorToken) {
      installSession.value = null
    }
    installError.value = errorMessage(error, 'quickStart.installStep.issueFailed')
  } finally {
    issuingToken.value = false
  }
}

async function copyCommand() {
  await copyToClipboard(currentCommand.value, t('quickStart.installStep.copied'))
}

function formatDate(value: string): string {
  const parsed = new Date(value)
  if (Number.isNaN(parsed.getTime())) return value
  return new Intl.DateTimeFormat(locale.value, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  }).format(parsed)
}

function errorMessage(error: unknown, fallbackKey = 'quickStart.errors.generic'): string {
  const candidate = error as { reason?: string; message?: string }
  switch (candidate?.reason) {
    case 'no_credit':
      return t('quickStart.errors.noCredit')
    case 'key_disabled':
      return t('quickStart.errors.keyDisabled')
    case 'client_mismatch':
      return t('quickStart.errors.clientMismatch')
    case 'token_expired':
      return t('quickStart.errors.tokenExpired')
    case 'token_used':
      return t('quickStart.errors.tokenUsed')
    case 'token_revoked':
      return t('quickStart.errors.tokenRevoked')
    default:
      return candidate?.message || t(fallbackKey)
  }
}

function isRelativeMarkdownAsset(src: string): boolean {
  const trimmed = src.trim()
  if (!trimmed || /^[a-z][a-z0-9+.-]*:/i.test(trimmed) || trimmed.startsWith('//') || trimmed.startsWith('/')) {
    return false
  }
  const [pathPart] = trimmed.split(/([?#].*)/, 2)
  return pathPart
    .split('/')
    .filter((part) => part && part !== '.')
    .every((part) => part !== '..' && !part.includes('\\'))
}

function buildPageImageUrl(src: string): string {
  const trimmed = src.trim()
  const [pathPart, suffix = ''] = trimmed.split(/([?#].*)/, 2)
  const encodedPath = pathPart
    .split('/')
    .filter((part) => part && part !== '.')
    .map((part) => encodeURIComponent(part))
    .join('/')
  return buildApiUrl(`/pages/${encodeURIComponent(props.faqSlug)}/images/${encodedPath}${suffix}`)
}

async function toggleFaq() {
  faqOpen.value = !faqOpen.value
  if (faqOpen.value && !faqLoaded.value && !faqLoading.value) {
    await loadFaq()
  }
}

async function loadFaq() {
  faqLoading.value = true
  faqError.value = ''
  try {
    const response = await fetch(buildApiUrl(`/pages/${encodeURIComponent(props.faqSlug)}`), {
      headers: authStore.token ? { Authorization: `Bearer ${authStore.token}` } : {}
    })
    if (!response.ok) {
      throw new Error(`FAQ request failed with status ${response.status}`)
    }
    let raw = await response.text()
    raw = raw.replace(
      /!\[([^\]]*)\]\(([^)]+)\)/g,
      (match, alt, src) => isRelativeMarkdownAsset(src)
        ? `![${alt}](${buildPageImageUrl(src)})`
        : match
    )
    faqHtml.value = DOMPurify.sanitize(marked.parse(raw) as string, {
      ADD_TAGS: ['iframe'],
      ADD_ATTR: ['allowfullscreen', 'frameborder', 'src']
    })
    faqLoaded.value = true
  } catch {
    faqError.value = 'failed'
  } finally {
    faqLoading.value = false
  }
}

watch(selectedClient, () => {
  normalizeKeySelections()
})

watch(selectedInstallMethod, () => {
  showNextSteps.value = false
})

watch([eligibleKeys, fundedCompatibleGroups], () => {
  normalizeKeySelections()
})

onMounted(() => {
  clockTimer = setInterval(() => {
    now.value = Date.now()
  }, 30_000)
  void loadSetupData()
})

onUnmounted(() => {
  if (clockTimer) {
    clearInterval(clockTimer)
    clockTimer = null
  }
})
</script>

<style scoped>
.quickstart-faq-content {
  line-height: 1.75;
}

.quickstart-faq-content :deep(h1) {
  @apply mb-4 mt-2 border-b border-gray-200 pb-3 text-2xl font-bold dark:border-dark-600;
}

.quickstart-faq-content :deep(h2) {
  @apply mb-3 mt-8 text-xl font-bold;
}

.quickstart-faq-content :deep(h3) {
  @apply mb-2 mt-6 text-lg font-semibold;
}

.quickstart-faq-content :deep(h4) {
  @apply mb-2 mt-5 text-base font-semibold;
}

.quickstart-faq-content :deep(p) {
  @apply mb-4;
}

.quickstart-faq-content :deep(ul) {
  @apply mb-4 list-disc pl-6;
}

.quickstart-faq-content :deep(ol) {
  @apply mb-4 list-decimal pl-6;
}

.quickstart-faq-content :deep(li) {
  @apply mb-1;
}

.quickstart-faq-content :deep(a) {
  @apply text-primary-600 underline hover:text-primary-700 dark:text-primary-400 dark:hover:text-primary-300;
}

.quickstart-faq-content :deep(blockquote) {
  @apply my-4 border-l-4 border-gray-300 pl-4 italic text-gray-600 dark:border-dark-500 dark:text-dark-300;
}

.quickstart-faq-content :deep(img) {
  @apply my-5 h-auto max-w-full rounded-lg;
}

.quickstart-faq-content :deep(table) {
  @apply my-5 w-full border-collapse;
}

.quickstart-faq-content :deep(th) {
  @apply border border-gray-300 bg-gray-50 px-3 py-2 text-left font-semibold dark:border-dark-500 dark:bg-dark-700;
}

.quickstart-faq-content :deep(td) {
  @apply border border-gray-300 px-3 py-2 dark:border-dark-500;
}

.quickstart-faq-content :deep(code) {
  @apply rounded bg-gray-100 px-1.5 py-0.5 font-mono text-sm dark:bg-dark-700;
}

.quickstart-faq-content :deep(pre) {
  @apply my-4 overflow-x-auto rounded-lg bg-gray-950 p-4 text-gray-100;
}

.quickstart-faq-content :deep(pre code) {
  @apply bg-transparent p-0 text-inherit;
}
</style>
