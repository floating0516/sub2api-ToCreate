<template>
  <AuthLayout>
    <div class="space-y-6 text-center">
      <div
        class="mx-auto flex h-12 w-12 items-center justify-center rounded-full"
        :class="failed ? 'bg-red-100 text-red-600 dark:bg-red-950/40 dark:text-red-400' : 'bg-primary-100 text-primary-600 dark:bg-primary-950/40 dark:text-primary-400'"
      >
        <Icon v-if="failed" name="exclamationCircle" size="lg" />
        <Icon v-else name="refresh" size="lg" class="animate-spin" />
      </div>

      <div>
        <h1 class="text-xl font-semibold text-gray-900 dark:text-white">
          {{ failed ? t('liheOAuth.authorizationFailed') : t('liheOAuth.connecting') }}
        </h1>
        <p v-if="failed" class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          {{ errorMessage }}
        </p>
      </div>

      <router-link v-if="failed" to="/integrations/lihe" class="btn btn-primary inline-flex items-center">
        <Icon name="arrowLeft" size="sm" class="mr-2" />
        {{ t('liheOAuth.returnToIntegration') }}
      </router-link>
    </div>
  </AuthLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { AuthLayout } from '@/components/layout'
import Icon from '@/components/icons/Icon.vue'
import { authorizeLihe } from '@/api/liheOAuth'
import { extractApiErrorMessage } from '@/utils/apiError'

const route = useRoute()
const { t } = useI18n()
const failed = ref(false)
const errorMessage = ref('')

onMounted(async () => {
  const query: Record<string, string> = {}
  for (const [key, value] of Object.entries(route.query)) {
    if (typeof value === 'string') query[key] = value
  }
  try {
    const result = await authorizeLihe(query)
    window.location.replace(result.redirect_to)
  } catch (error: unknown) {
    failed.value = true
    errorMessage.value = extractApiErrorMessage(error, t('liheOAuth.authorizationExpired'))
  }
})
</script>
