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
          {{ failed ? t('liheOIDC.authorizationFailed') : t('liheOIDC.connecting') }}
        </h1>
        <p v-if="failed" class="mt-2 text-sm text-gray-500 dark:text-dark-400">
          {{ errorMessage }}
        </p>
      </div>

      <router-link v-if="failed" to="/login" class="btn btn-primary inline-flex items-center">
        <Icon name="arrowLeft" size="sm" class="mr-2" />
        {{ t('liheOIDC.returnToLogin') }}
      </router-link>
    </div>
  </AuthLayout>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { AuthLayout } from '@/components/layout'
import Icon from '@/components/icons/Icon.vue'
import { authorizeOIDC, prepareOIDCAuthorization } from '@/api/liheOIDC'
import { useAuthStore } from '@/stores/auth'
import { extractApiErrorMessage } from '@/utils/apiError'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const { t } = useI18n()
const failed = ref(false)
const errorMessage = ref('')

onMounted(async () => {
  try {
    let requestId = typeof route.query.request_id === 'string' ? route.query.request_id : ''
    let unauthenticatedRedirectTo = ''
    if (!requestId) {
      const params: Record<string, string> = {}
      for (const [key, value] of Object.entries(route.query)) {
        if (typeof value === 'string') params[key] = value
      }
      // Replace the protocol URL before any asynchronous work so state and nonce
      // cannot remain in browser history when preparation fails or is interrupted.
      window.history.replaceState(window.history.state, '', route.path)
      const prepared = await prepareOIDCAuthorization(params)
      requestId = prepared.request_id
      unauthenticatedRedirectTo = prepared.unauthenticated_redirect_to || ''
      await router.replace({ path: '/oidc/authorize', query: { request_id: requestId } })
    }

    if (!authStore.isAuthenticated) {
      if (unauthenticatedRedirectTo) {
        window.location.replace(unauthenticatedRedirectTo)
        return
      }
      const redirect = router.resolve({
        path: '/oidc/authorize',
        query: { request_id: requestId },
      }).fullPath
      await router.replace({ path: '/login', query: { redirect } })
      return
    }

    const result = await authorizeOIDC(requestId)
    if (result.reauthenticate) {
      await authStore.logout()
      const redirect = router.resolve({
        path: '/oidc/authorize',
        query: { request_id: requestId },
      }).fullPath
      await router.replace({ path: '/login', query: { redirect } })
      return
    }
    if (!result.redirect_to) throw new Error('OIDC authorization response is incomplete')
    window.location.replace(result.redirect_to)
  } catch (error: unknown) {
    failed.value = true
    errorMessage.value = extractApiErrorMessage(error, t('liheOIDC.authorizationExpired'))
  }
})
</script>
