<template>
  <div
    v-if="open"
    class="email-auth-layer"
    :class="{ 'is-embedded': isEmbedded }"
    aria-live="polite"
  >
    <button
      v-if="!isEmbedded"
      type="button"
      class="email-auth-backdrop"
      :aria-label="t('common.close')"
      @click="closeDialog"
    />

    <section
      class="email-auth-dialog"
      :role="isEmbedded ? 'region' : 'dialog'"
      :aria-modal="isEmbedded ? undefined : 'true'"
      aria-labelledby="email-auth-title"
      data-testid="email-auth-dialog"
    >
      <button
        v-if="showBackButton"
        type="button"
        class="email-auth-icon-button email-auth-back"
        :aria-label="t('common.back')"
        :disabled="busy"
        @click="goBack"
      >
        <Icon name="arrowLeft" size="sm" />
      </button>
      <button
        v-if="!isEmbedded && canClose"
        type="button"
        class="email-auth-icon-button email-auth-close"
        :aria-label="t('common.close')"
        @click="closeDialog"
      >
        <Icon name="x" size="sm" />
      </button>

      <div class="email-auth-brand" aria-hidden="true">
        <span class="email-auth-orbit email-auth-orbit-one" />
        <span class="email-auth-orbit email-auth-orbit-two" />
        <img v-if="siteLogo" :src="siteLogo" alt="" />
        <Icon v-else name="sparkles" size="lg" />
      </div>

      <div :key="step" class="email-auth-stage">
        <template v-if="step === 'email'">
          <header class="email-auth-copy">
            <p>{{ t('auth.emailFirst.welcome', { siteName }) }}</p>
            <h2 id="email-auth-title">{{ t('auth.emailFirst.title') }}</h2>
            <span>{{ t('auth.emailFirst.emailDescription') }}</span>
          </header>

          <form novalidate @submit.prevent="submitEmail">
            <label class="email-auth-label" for="email-auth-email">
              {{ t('auth.emailLabel') }}
            </label>
            <div class="email-auth-input-shell" :class="{ 'is-error': Boolean(errorMessage) }">
              <Icon name="mail" size="sm" />
              <input
                id="email-auth-email"
                ref="emailInputRef"
                v-model="email"
                type="email"
                autocomplete="email"
                :placeholder="t('auth.emailPlaceholder')"
                :disabled="busy || settingsLoading"
                data-testid="email-auth-email"
                @input="clearError"
              />
            </div>
            <p class="email-auth-message" :class="{ 'is-error': Boolean(errorMessage) }" role="status">
              {{ errorMessage || t('auth.emailFirst.emailHint') }}
            </p>
            <div class="email-auth-choice" :class="{ 'is-single': Boolean(settings) && !canRegister }">
              <button
                type="button"
                :class="intent === 'register' ? 'email-auth-secondary' : 'email-auth-primary'"
                :disabled="busy || settingsLoading"
                data-testid="email-auth-continue"
                @click="continueToLogin"
              >
                <span v-if="settingsLoading" class="email-auth-spinner" aria-hidden="true" />
                {{ settingsLoading ? t('common.loading') : t('auth.signIn') }}
              </button>
              <button
                v-if="!settings || canRegister"
                type="button"
                :class="intent === 'register' ? 'email-auth-primary' : 'email-auth-secondary'"
                :disabled="busy || settingsLoading"
                data-testid="email-auth-create-account"
                @click="continueToRegister"
              >
                {{ t('auth.createAccount') }}
              </button>
            </div>
          </form>
        </template>

        <template v-else-if="step === 'login'">
          <header class="email-auth-copy">
            <p>{{ t('auth.emailFirst.signInEyebrow') }}</p>
            <h2 id="email-auth-title">{{ t('auth.welcomeBack') }}</h2>
            <span class="email-auth-account" :title="email">{{ email }}</span>
          </header>

          <form novalidate @submit.prevent="submitLogin">
            <label class="email-auth-label" for="email-auth-password">
              {{ t('auth.passwordLabel') }}
            </label>
            <div class="email-auth-input-shell" :class="{ 'is-error': Boolean(errorMessage) }">
              <Icon name="lock" size="sm" />
              <input
                id="email-auth-password"
                ref="passwordInputRef"
                v-model="password"
                :type="showPassword ? 'text' : 'password'"
                autocomplete="current-password"
                :placeholder="t('auth.passwordPlaceholder')"
                :disabled="busy"
                data-testid="email-auth-password"
                @input="clearError"
              />
              <button
                type="button"
                class="email-auth-field-action"
                :aria-label="showPassword ? t('auth.emailFirst.hidePassword') : t('auth.emailFirst.showPassword')"
                :disabled="busy"
                @click="showPassword = !showPassword"
              >
                <Icon :name="showPassword ? 'eyeOff' : 'eye'" size="sm" />
              </button>
            </div>
            <div class="email-auth-message-row">
              <p class="email-auth-message" :class="{ 'is-error': Boolean(errorMessage) }" role="status">
                {{ errorMessage || t('auth.emailFirst.passwordHint') }}
              </p>
              <button
                v-if="settings?.password_reset_enabled && !settings?.backend_mode_enabled"
                type="button"
                class="email-auth-inline-link"
                @click="openFullFlow('/forgot-password')"
              >
                {{ t('auth.forgotPassword') }}
              </button>
            </div>
            <button
              type="submit"
              class="email-auth-primary"
              :disabled="busy"
              data-testid="email-auth-login"
            >
              <span v-if="busy" class="email-auth-spinner" aria-hidden="true" />
              {{ busy ? t('auth.signingIn') : t('auth.signIn') }}
            </button>
          </form>

          <div class="email-auth-switch">
            <span>{{ t('auth.dontHaveAccount') }}</span>
            <button
              type="button"
              :disabled="busy"
              data-testid="email-auth-start-register"
              @click="continueToRegister"
            >
              {{ t('auth.createAccount') }}
            </button>
          </div>
          <button
            v-if="hasAlternativeLoginMethods"
            type="button"
            class="email-auth-text-button"
            :disabled="busy"
            data-testid="email-auth-other-methods"
            @click="openAlternativeMethods"
          >
            {{ t('auth.emailFirst.otherLoginMethods') }}
          </button>
        </template>

        <template v-else-if="step === 'register'">
          <header class="email-auth-copy">
            <p>{{ t('auth.emailFirst.createEyebrow') }}</p>
            <h2 id="email-auth-title">{{ t('auth.createAccount') }}</h2>
            <span class="email-auth-account" :title="email">{{ email }}</span>
          </header>

          <form novalidate @submit.prevent="submitRegistrationDetails">
            <label class="email-auth-label" for="email-auth-new-password">
              {{ t('auth.passwordLabel') }}
            </label>
            <div class="email-auth-input-shell" :class="{ 'is-error': Boolean(errorMessage) }">
              <Icon name="lock" size="sm" />
              <input
                id="email-auth-new-password"
                ref="passwordInputRef"
                v-model="password"
                :type="showPassword ? 'text' : 'password'"
                autocomplete="new-password"
                :placeholder="t('auth.createPasswordPlaceholder')"
                :disabled="busy"
                data-testid="email-auth-register-password"
                @input="clearError"
              />
              <button
                type="button"
                class="email-auth-field-action"
                :aria-label="showPassword ? t('auth.emailFirst.hidePassword') : t('auth.emailFirst.showPassword')"
                :disabled="busy"
                @click="showPassword = !showPassword"
              >
                <Icon :name="showPassword ? 'eyeOff' : 'eye'" size="sm" />
              </button>
            </div>

            <label class="email-auth-label email-auth-confirm-label" for="email-auth-confirm-password">
              {{ t('auth.confirmPassword') }}
            </label>
            <div class="email-auth-input-shell" :class="{ 'is-error': Boolean(errorMessage) }">
              <Icon name="shield" size="sm" />
              <input
                id="email-auth-confirm-password"
                v-model="confirmPassword"
                :type="showPassword ? 'text' : 'password'"
                autocomplete="new-password"
                :placeholder="t('auth.confirmPasswordPlaceholder')"
                :disabled="busy"
                data-testid="email-auth-confirm-password"
                @input="clearError"
              />
            </div>

            <p class="email-auth-message" :class="{ 'is-error': Boolean(errorMessage) }" role="status">
              {{ errorMessage || t('auth.passwordHint') }}
            </p>
            <button
              type="submit"
              class="email-auth-primary"
              :disabled="busy"
              data-testid="email-auth-register"
            >
              <span v-if="busy" class="email-auth-spinner" aria-hidden="true" />
              {{ busy ? t('auth.sendingCode') : registrationActionLabel }}
            </button>
          </form>

          <div class="email-auth-switch">
            <span>{{ t('auth.alreadyHaveAccount') }}</span>
            <button type="button" :disabled="busy" @click="switchToLogin">
              {{ t('auth.signIn') }}
            </button>
          </div>
        </template>

        <template v-else-if="step === 'methods'">
          <header class="email-auth-copy">
            <p>{{ t('auth.emailFirst.otherMethodsEyebrow') }}</p>
            <h2 id="email-auth-title">{{ t('auth.emailFirst.otherMethodsTitle') }}</h2>
            <span>{{ t('auth.emailFirst.otherMethodsDescription') }}</span>
          </header>

          <div class="email-auth-method-list" data-testid="email-auth-methods">
            <button
              v-if="passkeyLoginAvailable"
              type="button"
              class="email-auth-method-button"
              :disabled="busy"
              data-testid="email-auth-passkey"
              @click="handlePasskeyLogin"
            >
              <Icon name="key" size="sm" />
              {{ busy ? t('auth.passkeySigningIn') : t('auth.passkeySignIn') }}
            </button>

            <EmailOAuthButtons
              :disabled="busy"
              :github-enabled="settings?.github_oauth_enabled"
              :google-enabled="settings?.google_oauth_enabled"
              :show-divider="false"
              @start="handleOAuthStart"
            />
            <LinuxDoOAuthSection
              v-if="settings?.linuxdo_oauth_enabled"
              :disabled="busy"
              :show-divider="false"
              @start="handleOAuthStart"
            />
            <DingTalkOAuthSection
              v-if="settings?.dingtalk_oauth_enabled"
              :disabled="busy"
              :show-divider="false"
              @start="handleOAuthStart"
            />
            <WechatOAuthSection
              v-if="wechatOAuthAvailable"
              :disabled="busy"
              :show-divider="false"
              @start="handleOAuthStart"
            />
            <OidcOAuthSection
              v-if="settings?.oidc_oauth_enabled"
              :disabled="busy"
              :provider-name="settings?.oidc_oauth_provider_name || 'OIDC'"
              :show-divider="false"
              @start="handleOAuthStart"
            />
          </div>

          <p
            v-if="errorMessage"
            class="email-auth-message email-auth-method-error is-error"
            role="status"
          >
            {{ errorMessage }}
          </p>
          <button
            type="button"
            class="email-auth-text-button"
            :disabled="busy"
            data-testid="email-auth-email-password"
            @click="returnToPasswordLogin"
          >
            {{ t('auth.emailFirst.emailPasswordMethod') }}
          </button>
        </template>

        <template v-else-if="step === 'verification' || step === 'totp'">
          <header class="email-auth-copy">
            <p>{{ step === 'totp' ? t('auth.emailFirst.twoFactorEyebrow') : t('auth.emailFirst.verifyEyebrow') }}</p>
            <h2 id="email-auth-title">
              {{ step === 'totp' ? t('auth.emailFirst.twoFactorTitle') : t('auth.emailFirst.checkInbox') }}
            </h2>
            <span>
              {{ step === 'totp' ? t('auth.emailFirst.twoFactorDescription') : t('auth.emailFirst.codeSentTo') }}
              <strong>{{ step === 'totp' ? maskedEmail || email : email }}</strong>
            </span>
          </header>

          <form @submit.prevent="submitCode">
            <label class="email-auth-label" for="email-auth-code">{{ t('auth.verificationCode') }}</label>
            <div
              class="email-auth-otp"
              :class="{ 'is-error': Boolean(errorMessage) }"
              @click="codeInputRef?.focus()"
            >
              <input
                id="email-auth-code"
                ref="codeInputRef"
                class="email-auth-otp-input"
                :value="code"
                inputmode="numeric"
                autocomplete="one-time-code"
                maxlength="6"
                :disabled="busy"
                :aria-label="t('auth.verificationCode')"
                data-testid="email-auth-code"
                @input="handleCodeInput"
              />
              <span
                v-for="index in 6"
                :key="index"
                :class="{
                  'is-filled': Boolean(code[index - 1]),
                  'is-current': code.length === index - 1
                }"
              >
                {{ code[index - 1] || '' }}
                <i
                  v-if="code.length === index - 1 && !code[index - 1]"
                  class="email-auth-otp-caret"
                  aria-hidden="true"
                />
              </span>
            </div>
            <p class="email-auth-message" :class="{ 'is-error': Boolean(errorMessage) }" role="status">
              {{ errorMessage || t('auth.emailFirst.codeHint') }}
            </p>
            <button
              type="submit"
              class="email-auth-primary"
              :disabled="busy || code.length !== 6"
              data-testid="email-auth-verify"
            >
              <span v-if="busy" class="email-auth-spinner" aria-hidden="true" />
              {{ busy ? t('auth.verifying') : t('auth.emailFirst.verifyCode') }}
            </button>
          </form>

          <button
            v-if="step === 'verification'"
            type="button"
            class="email-auth-text-button"
            :disabled="busy || countdown > 0"
            @click="resendCode"
          >
            {{ countdown > 0 ? t('auth.resendCountdown', { countdown }) : t('auth.resendCode') }}
          </button>
        </template>

        <template v-else-if="step === 'preparing'">
          <div class="email-auth-progress">
            <div class="email-auth-progress-rings" aria-hidden="true">
              <span />
              <span />
              <Icon name="shield" size="lg" />
            </div>
            <p>{{ t('auth.emailFirst.signedIn') }}</p>
            <h2 id="email-auth-title">{{ t('auth.emailFirst.preparingTitle') }}</h2>
            <span>{{ t('auth.emailFirst.preparingDescription') }}</span>
            <div class="email-auth-skeleton" aria-hidden="true">
              <i /><i /><i />
            </div>
          </div>
        </template>

        <template v-else>
          <div class="email-auth-progress email-auth-success">
            <div class="email-auth-success-mark" aria-hidden="true">
              <Icon name="check" size="lg" />
            </div>
            <p>{{ t('auth.emailFirst.ready') }}</p>
            <h2 id="email-auth-title">{{ t('auth.emailFirst.successTitle') }}</h2>
            <span>{{ t('auth.emailFirst.successDescription') }}</span>
          </div>
        </template>
      </div>

      <p class="email-auth-legal">{{ t('auth.emailFirst.legal') }}</p>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import DingTalkOAuthSection from '@/components/auth/DingTalkOAuthSection.vue'
import EmailOAuthButtons from '@/components/auth/EmailOAuthButtons.vue'
import Icon from '@/components/icons/Icon.vue'
import LinuxDoOAuthSection from '@/components/auth/LinuxDoOAuthSection.vue'
import OidcOAuthSection from '@/components/auth/OidcOAuthSection.vue'
import WechatOAuthSection from '@/components/auth/WechatOAuthSection.vue'
import { useAppStore, useAuthStore } from '@/stores'
import {
  buildOAuthLoginStartURL,
  checkRegistrationEmail,
  isTotp2FARequired,
  isWeChatWebOAuthEnabled,
  sendVerifyCode,
  type OAuthLoginStart
} from '@/api/auth'
import type { PublicSettings, TotpLoginResponse } from '@/types'
import { extractApiErrorCode, extractI18nErrorMessage } from '@/utils/apiError'
import { clearAllAffiliateReferralCodes, resolveAffiliateReferralCode } from '@/utils/oauthAffiliate'
import { isRegistrationEmailSuffixAllowed } from '@/utils/registrationEmailPolicy'
import { storeFullRegistrationDraft } from '@/utils/registrationDraft'

type AuthStep = 'email' | 'login' | 'register' | 'methods' | 'verification' | 'totp' | 'preparing' | 'success'

interface AuthDialogDelays {
  preparing: number
  success: number
}

type AuthPresentation = 'dialog' | 'embedded'
type AuthIntent = 'authenticate' | 'register'

const props = withDefaults(defineProps<{
  open: boolean
  siteName: string
  siteLogo?: string
  dashboardPath: string
  delays?: Partial<AuthDialogDelays>
  presentation?: AuthPresentation
  intent?: AuthIntent
  initialEmail?: string
}>(), {
  siteLogo: '',
  delays: () => ({}),
  presentation: 'dialog',
  intent: 'authenticate',
  initialEmail: ''
})

const emit = defineEmits<{
  'update:open': [open: boolean]
}>()

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const authStore = useAuthStore()
const appStore = useAppStore()

const defaultDelays: AuthDialogDelays = {
  preparing: 850,
  success: 650
}

const step = ref<AuthStep>('email')
const email = ref(normalizeEmail(props.initialEmail))
const password = ref('')
const confirmPassword = ref('')
const code = ref('')
const errorMessage = ref('')
const maskedEmail = ref('')
const tempToken = ref('')
const countdown = ref(0)
const busy = ref(false)
const settingsLoading = ref(false)
const settings = ref<PublicSettings | null>(null)
const showPassword = ref(false)
const emailInputRef = ref<HTMLInputElement | null>(null)
const passwordInputRef = ref<HTMLInputElement | null>(null)
const codeInputRef = ref<HTMLInputElement | null>(null)
let countdownTimer: ReturnType<typeof setInterval> | null = null

const delays = computed(() => ({ ...defaultDelays, ...props.delays }))
const isEmbedded = computed(() => props.presentation === 'embedded')
const canClose = computed(() => !busy.value && step.value !== 'preparing' && step.value !== 'success')
const showBackButton = computed(() => ['login', 'register', 'methods', 'verification', 'totp'].includes(step.value))
const hasCaptcha = computed(() => Boolean(
  settings.value?.turnstile_enabled ||
  settings.value?.tencent_captcha_enabled ||
  settings.value?.aliyun_captcha_enabled
))
const loginNeedsFullPage = computed(() => Boolean(
  hasCaptcha.value || settings.value?.login_agreement_enabled
))
const passkeyLoginAvailable = computed(() => Boolean(
  settings.value?.passkey_enabled && typeof window.PublicKeyCredential !== 'undefined'
))
const wechatOAuthAvailable = computed(() => isWeChatWebOAuthEnabled(settings.value))
const hasAlternativeLoginMethods = computed(() => Boolean(
  !settings.value?.backend_mode_enabled && (
    passkeyLoginAvailable.value ||
    settings.value?.linuxdo_oauth_enabled ||
    settings.value?.dingtalk_oauth_enabled ||
    wechatOAuthAvailable.value ||
    settings.value?.oidc_oauth_enabled ||
    settings.value?.github_oauth_enabled ||
    settings.value?.google_oauth_enabled
  )
))
const registrationNeedsFullPage = computed(() => Boolean(
  loginNeedsFullPage.value ||
  settings.value?.invitation_code_enabled ||
  settings.value?.promo_code_enabled ||
  settings.value?.affiliate_enabled ||
  settings.value?.linuxdo_oauth_enabled ||
  settings.value?.wechat_oauth_enabled ||
  settings.value?.oidc_oauth_enabled ||
  settings.value?.github_oauth_enabled ||
  settings.value?.google_oauth_enabled
))
const canRegister = computed(() => Boolean(
  settings.value?.registration_enabled && !settings.value?.backend_mode_enabled
))
const registrationActionLabel = computed(() =>
  settings.value?.email_verify_enabled ? t('auth.emailFirst.sendCode') : t('auth.createAccount')
)

function normalizeEmail(value: string): string {
  return value.trim().toLowerCase()
}

function isValidEmail(value: string): boolean {
  return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(normalizeEmail(value))
}

function wait(duration: number): Promise<void> {
  return new Promise(resolve => window.setTimeout(resolve, duration))
}

function clearError(): void {
  if (errorMessage.value) errorMessage.value = ''
}

function clearCountdown(): void {
  if (countdownTimer) {
    clearInterval(countdownTimer)
    countdownTimer = null
  }
}

function startCountdown(seconds: number): void {
  clearCountdown()
  countdown.value = Math.max(0, seconds)
  if (countdown.value === 0) return

  countdownTimer = setInterval(() => {
    countdown.value = Math.max(0, countdown.value - 1)
    if (countdown.value === 0) clearCountdown()
  }, 1000)
}

async function ensureSettings(): Promise<void> {
  if (appStore.cachedPublicSettings) {
    settings.value = appStore.cachedPublicSettings
    return
  }

  settingsLoading.value = true
  try {
    settings.value = await appStore.fetchPublicSettings()
    if (!settings.value) {
      errorMessage.value = t('auth.emailFirst.settingsUnavailable')
    }
  } catch {
    errorMessage.value = t('auth.emailFirst.settingsUnavailable')
  } finally {
    settingsLoading.value = false
  }
}

function resetDialog(): void {
  step.value = 'email'
  email.value = normalizeEmail(props.initialEmail)
  password.value = ''
  confirmPassword.value = ''
  code.value = ''
  errorMessage.value = ''
  maskedEmail.value = ''
  tempToken.value = ''
  countdown.value = 0
  busy.value = false
  showPassword.value = false
  clearCountdown()
}

function closeDialog(): void {
  if (!canClose.value) return
  emit('update:open', false)
  window.setTimeout(resetDialog, 220)
}

function goBack(): void {
  if (busy.value) return
  clearError()
  code.value = ''

  if (step.value === 'methods') {
    step.value = 'login'
    return
  }
  if (step.value === 'verification') {
    clearCountdown()
    step.value = 'register'
    return
  }
  if (step.value === 'totp') {
    tempToken.value = ''
    maskedEmail.value = ''
    step.value = 'login'
    return
  }

  password.value = ''
  confirmPassword.value = ''
  step.value = 'email'
}

async function acceptEmail(): Promise<boolean> {
  clearError()
  if (!isValidEmail(email.value)) {
    errorMessage.value = t('auth.invalidEmail')
    return false
  }
  if (!settings.value) {
    await ensureSettings()
    if (!settings.value) return false
  }

  email.value = normalizeEmail(email.value)
  return true
}

async function submitEmail(): Promise<void> {
  if (props.intent === 'register') {
    await continueToRegister()
    return
  }
  await continueToLogin()
}

function handleExistingEmail(): void {
  switchToLogin()
  errorMessage.value = t('auth.errors.EMAIL_EXISTS')
}

function handleUnregisteredEmail(): void {
  errorMessage.value = t('auth.errors.EMAIL_NOT_REGISTERED')
}

function isExistingEmailError(error: unknown): boolean {
  return extractApiErrorCode(error) === 'EMAIL_EXISTS'
}

function isUnregisteredEmailError(error: unknown): boolean {
  const code = extractApiErrorCode(error)
  return code === 'REGISTRATION_DISABLED' || code === 'EMAIL_RESERVED'
}

async function ensureEmailAvailableForRegistration(): Promise<boolean> {
  busy.value = true
  try {
    await checkRegistrationEmail(email.value)
    return true
  } catch (error: unknown) {
    if (isExistingEmailError(error)) {
      handleExistingEmail()
      return false
    }
    errorMessage.value = extractI18nErrorMessage(error, t, 'auth.errors', t('auth.registrationFailed'))
    return false
  } finally {
    busy.value = false
  }
}

async function ensureEmailRegisteredForLogin(): Promise<boolean> {
  busy.value = true
  try {
    await checkRegistrationEmail(email.value)
    handleUnregisteredEmail()
    return false
  } catch (error: unknown) {
    if (isExistingEmailError(error)) {
      return true
    }
    if (isUnregisteredEmailError(error)) {
      handleUnregisteredEmail()
      return false
    }
    errorMessage.value = extractI18nErrorMessage(error, t, 'auth.errors', t('auth.loginFailed'))
    return false
  } finally {
    busy.value = false
  }
}

async function continueToLogin(): Promise<void> {
  if (!await acceptEmail()) return
  if (!await ensureEmailRegisteredForLogin()) return
  switchToLogin()
}

async function continueToRegister(): Promise<void> {
  if (!await acceptEmail()) return
  if (!await ensureEmailAvailableForRegistration()) return
  startRegistration()
}

function startRegistration(): void {
  clearError()
  if (!canRegister.value) {
    errorMessage.value = t('auth.registrationDisabled')
    return
  }
  password.value = ''
  confirmPassword.value = ''
  showPassword.value = false
  step.value = 'register'
}

function switchToLogin(): void {
  clearError()
  password.value = ''
  confirmPassword.value = ''
  showPassword.value = false
  step.value = 'login'
}

async function openAlternativeMethods(): Promise<void> {
  clearError()
  if (loginNeedsFullPage.value) {
    await openFullFlow('/login')
    return
  }
  step.value = 'methods'
}

function returnToPasswordLogin(): void {
  clearError()
  step.value = 'login'
}

async function openFullFlow(path: string): Promise<void> {
  const preservedEmail = email.value
  resolveAffiliateReferralCode(route.query.aff, route.query.aff_code)
  const query: Record<string, string> = {}
  for (const key of ['promo', 'aff', 'aff_code', 'redirect']) {
    const value = route.query[key]
    if (typeof value === 'string' && value) query[key] = value
  }
  query.email = preservedEmail
  if (path === '/login' && loginNeedsFullPage.value) {
    query.full = '1'
  }
  if (path === '/register' && registrationNeedsFullPage.value) {
    query.full = '1'
    storeFullRegistrationDraft({ email: preservedEmail, password: password.value })
  }
  if (!isEmbedded.value) {
    emit('update:open', false)
    resetDialog()
  }
  await router.push({ path, query })
}

async function handlePasskeyLogin(): Promise<void> {
  if (busy.value) return
  clearError()
  busy.value = true
  try {
    await authStore.loginWithPasskey()
    await finishAuthentication()
  } catch (error: unknown) {
    const fallback = error instanceof DOMException && error.name === 'NotAllowedError'
      ? t('auth.passkeyCancelled')
      : t('auth.passkeyFailed')
    errorMessage.value = extractI18nErrorMessage(error, t, 'auth.errors', fallback)
    busy.value = false
  }
}

function handleOAuthStart(request: OAuthLoginStart): void {
  if (busy.value) return
  clearError()
  busy.value = true
  window.location.href = buildOAuthLoginStartURL(request)
}

async function submitLogin(): Promise<void> {
  clearError()
  if (!password.value) {
    errorMessage.value = t('auth.passwordRequired')
    return
  }
  if (password.value.length < 6) {
    errorMessage.value = t('auth.passwordMinLength')
    return
  }
  if (loginNeedsFullPage.value) {
    await openFullFlow('/login')
    return
  }

  busy.value = true
  try {
    const response = await authStore.login({ email: email.value, password: password.value })
    if (isTotp2FARequired(response)) {
      const challenge = response as TotpLoginResponse
      tempToken.value = challenge.temp_token || ''
      maskedEmail.value = challenge.user_email_masked || ''
      code.value = ''
      step.value = 'totp'
      busy.value = false
      return
    }
    await finishAuthentication()
  } catch (error: unknown) {
    errorMessage.value = extractI18nErrorMessage(error, t, 'auth.errors', t('auth.loginFailed'))
    busy.value = false
  }
}

function validateRegistrationDetails(): boolean {
  if (!password.value) {
    errorMessage.value = t('auth.passwordRequired')
    return false
  }
  if (password.value.length < 6) {
    errorMessage.value = t('auth.passwordMinLength')
    return false
  }
  if (!confirmPassword.value) {
    errorMessage.value = t('auth.confirmPasswordRequired')
    return false
  }
  if (password.value !== confirmPassword.value) {
    errorMessage.value = t('auth.passwordsDoNotMatch')
    return false
  }
  if (
    !settings.value?.registration_email_domain_quota_enabled &&
    !isRegistrationEmailSuffixAllowed(email.value, settings.value?.registration_email_suffix_whitelist || [])
  ) {
    errorMessage.value = t('auth.emailSuffixNotAllowed')
    return false
  }
  return true
}

async function submitRegistrationDetails(): Promise<void> {
  clearError()
  if (!validateRegistrationDetails()) return
  if (registrationNeedsFullPage.value) {
    await openFullFlow('/register')
    return
  }

  if (!settings.value?.email_verify_enabled) {
    await createAccount()
    return
  }

  await sendRegistrationCode()
}

async function sendRegistrationCode(): Promise<void> {
  busy.value = true
  try {
    const response = await sendVerifyCode({ email: email.value })
    startCountdown(response.countdown)
    code.value = ''
    step.value = 'verification'
  } catch (error: unknown) {
    if (extractApiErrorCode(error) === 'EMAIL_EXISTS') {
      handleExistingEmail()
      return
    }
    errorMessage.value = extractI18nErrorMessage(error, t, 'auth.errors', t('auth.sendCodeFailed'))
  } finally {
    busy.value = false
  }
}

async function resendCode(): Promise<void> {
  if (busy.value || countdown.value > 0) return
  clearError()
  await sendRegistrationCode()
}

function handleCodeInput(event: Event): void {
  const input = event.target as HTMLInputElement
  code.value = input.value.replace(/\D/g, '').slice(0, 6)
  input.value = code.value
  clearError()
  if (/^\d{6}$/.test(code.value) && !busy.value) {
    void submitCode()
  }
}

async function submitCode(): Promise<void> {
  if (busy.value) return
  clearError()
  if (!/^\d{6}$/.test(code.value)) {
    errorMessage.value = t('auth.invalidCode')
    return
  }

  if (step.value === 'totp') {
    busy.value = true
    try {
      await authStore.login2FA(tempToken.value, code.value)
      await finishAuthentication()
    } catch (error: unknown) {
      errorMessage.value = extractI18nErrorMessage(error, t, 'auth.errors', t('auth.verifyFailed'))
      code.value = ''
      busy.value = false
    }
    return
  }

  await createAccount(code.value)
}

async function createAccount(verifyCode?: string): Promise<void> {
  busy.value = true
  try {
    const affCode = resolveAffiliateReferralCode(route.query.aff, route.query.aff_code)
    await authStore.register({
      email: email.value,
      password: password.value,
      verify_code: verifyCode,
      ...(affCode ? { aff_code: affCode } : {})
    })
    await finishAuthentication()
  } catch (error: unknown) {
    if (extractApiErrorCode(error) === 'EMAIL_EXISTS') {
      handleExistingEmail()
      busy.value = false
      return
    }
    errorMessage.value = extractI18nErrorMessage(error, t, 'auth.errors', t('auth.registrationFailed'))
    busy.value = false
  }
}

async function finishAuthentication(): Promise<void> {
  clearAllAffiliateReferralCodes()
  step.value = 'preparing'
  await wait(delays.value.preparing)
  step.value = 'success'
  await wait(delays.value.success)
  if (!isEmbedded.value) {
    emit('update:open', false)
  }
  resetDialog()
  await router.push(resolvePostAuthPath())
}

function resolvePostAuthPath(): string {
  const redirect = route.query.redirect
  if (typeof redirect === 'string' && redirect.startsWith('/') && !redirect.startsWith('//')) {
    return redirect
  }
  return props.dashboardPath
}

function handleKeydown(event: KeyboardEvent): void {
  if (event.key === 'Escape' && props.open && !isEmbedded.value) closeDialog()
}

watch(
  () => props.open,
  async open => {
    if (!open) {
      if (!isEmbedded.value) document.body.style.overflow = ''
      return
    }
    if (!isEmbedded.value) document.body.style.overflow = 'hidden'
    await ensureSettings()
    await nextTick()
    window.setTimeout(() => emailInputRef.value?.focus(), 60)
  },
  { immediate: true }
)

watch(
  () => props.initialEmail,
  value => {
    if (step.value === 'email') email.value = normalizeEmail(value)
  }
)

watch(step, async currentStep => {
  await nextTick()
  window.setTimeout(() => {
    if (currentStep === 'login' || currentStep === 'register') passwordInputRef.value?.focus()
    if (currentStep === 'verification' || currentStep === 'totp') codeInputRef.value?.focus()
  }, 60)
})

window.addEventListener('keydown', handleKeydown)

onBeforeUnmount(() => {
  if (!isEmbedded.value) document.body.style.overflow = ''
  window.removeEventListener('keydown', handleKeydown)
  clearCountdown()
})
</script>

<style scoped>
.email-auth-layer {
  position: fixed;
  z-index: 100;
  inset: 0;
  display: grid;
  padding: 20px;
  place-items: center;
}

.email-auth-layer.is-embedded {
  position: relative;
  z-index: auto;
  inset: auto;
  display: block;
  padding: 0;
}

.email-auth-backdrop {
  position: absolute;
  inset: 0;
  border: 0;
  background: rgba(39, 35, 30, 0.34);
  backdrop-filter: blur(8px) saturate(0.85);
  animation: email-auth-backdrop-in 180ms ease-out both;
}

.email-auth-dialog {
  --auth-bg: #fffefb;
  --auth-surface: #fbfaf7;
  --auth-ink: #2c2c2c;
  --auth-muted: #9a9080;
  --auth-subtle: #b0a898;
  --auth-line: rgba(61, 54, 47, 0.13);
  --auth-brand: #c4956a;
  --auth-brand-deep: #a87d5a;
  --auth-brand-soft: #f7f2ec;
  --auth-danger: #b45050;
  position: relative;
  z-index: 1;
  width: min(432px, calc(100vw - 32px));
  min-height: 568px;
  overflow: hidden;
  padding: 34px 38px 22px;
  border: 1px solid rgba(69, 58, 48, 0.12);
  border-radius: 20px;
  color: var(--auth-ink);
  background:
    radial-gradient(circle at 50% -13%, rgba(218, 184, 153, 0.23), transparent 170px),
    var(--auth-bg);
  box-shadow:
    0 36px 90px rgba(35, 29, 24, 0.22),
    0 8px 26px rgba(35, 29, 24, 0.1),
    inset 0 1px rgba(255, 255, 255, 0.85);
  animation: email-auth-dialog-in 280ms cubic-bezier(0.22, 1, 0.36, 1) both;
}

.email-auth-layer.is-embedded .email-auth-dialog {
  width: 100%;
  min-height: 0;
  padding: 0;
  overflow: visible;
  border: 0;
  border-radius: 0;
  background: transparent;
  box-shadow: none;
  animation: none;
}

:global(html.dark .email-auth-dialog) {
  box-shadow:
    0 36px 90px rgba(0, 0, 0, 0.46),
    0 8px 26px rgba(0, 0, 0, 0.25),
    inset 0 1px rgba(255, 255, 255, 0.85);
}

:global(html.dark .email-auth-layer.is-embedded .email-auth-dialog) {
  box-shadow: none;
}

.email-auth-dialog *,
.email-auth-dialog *::before,
.email-auth-dialog *::after {
  box-sizing: border-box;
  letter-spacing: 0;
}

.email-auth-icon-button {
  position: absolute;
  z-index: 3;
  top: 16px;
  width: 32px;
  height: 32px;
  display: grid;
  border: 1px solid transparent;
  border-radius: 7px;
  color: var(--auth-muted);
  background: transparent;
  cursor: pointer;
  place-items: center;
  transition: color 150ms ease, border-color 150ms ease, background 150ms ease;
}

.email-auth-icon-button:disabled {
  cursor: default;
  opacity: 0.45;
}

.email-auth-back { left: 16px; }
.email-auth-close { right: 16px; }

.email-auth-brand {
  position: relative;
  width: 57px;
  height: 57px;
  display: grid;
  margin: 2px auto 19px;
  border: 1px solid rgba(185, 130, 88, 0.2);
  border-radius: 16px;
  color: #a8754f;
  background: linear-gradient(145deg, #f5e9dc, #ead5c2);
  box-shadow: 0 9px 22px rgba(151, 103, 67, 0.13), inset 0 1px rgba(255, 255, 255, 0.72);
  place-items: center;
}

.email-auth-brand img {
  width: 38px;
  height: 38px;
  border-radius: 7px;
  object-fit: contain;
}

.email-auth-orbit {
  position: absolute;
  width: 72px;
  height: 24px;
  border: 1px solid color-mix(in srgb, var(--auth-brand) 25%, transparent);
  border-radius: 50%;
}

.email-auth-orbit-one { transform: rotate(27deg); }
.email-auth-orbit-two { transform: rotate(-27deg); }

.email-auth-stage {
  min-height: 340px;
  animation: email-auth-stage-in 220ms cubic-bezier(0.2, 0.8, 0.2, 1) both;
}

.email-auth-copy {
  margin-bottom: 23px;
  text-align: center;
}

.email-auth-copy > p,
.email-auth-progress > p {
  margin: 0 0 8px;
  color: var(--auth-brand);
  font-size: 9px;
  font-weight: 760;
  text-transform: uppercase;
}

.email-auth-copy h2,
.email-auth-progress h2 {
  margin: 0;
  font-family: Georgia, "Times New Roman", serif;
  font-size: 29px;
  font-weight: 500;
  line-height: 1.12;
}

.email-auth-copy > span,
.email-auth-progress > span {
  display: block;
  max-width: 310px;
  margin: 10px auto 0;
  color: var(--auth-muted);
  font-size: 12px;
  line-height: 1.55;
}

.email-auth-account {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.email-auth-copy strong {
  display: inline-block;
  max-width: 245px;
  overflow: hidden;
  color: var(--auth-ink);
  font-weight: 650;
  text-overflow: ellipsis;
  vertical-align: bottom;
  white-space: nowrap;
}

.email-auth-label {
  display: block;
  margin: 0 0 7px 2px;
  color: var(--auth-muted);
  font-size: 10px;
  font-weight: 700;
}

.email-auth-confirm-label { margin-top: 12px; }

.email-auth-input-shell {
  height: 47px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 13px;
  border: 1px solid var(--auth-line);
  border-radius: 10px;
  color: var(--auth-subtle);
  background: var(--auth-surface);
  box-shadow: inset 0 1px 2px rgba(54, 44, 36, 0.025);
  transition: border-color 160ms ease, box-shadow 160ms ease, background 160ms ease;
}

.email-auth-input-shell:focus-within {
  border-color: rgba(181, 126, 84, 0.55);
  background: var(--auth-bg);
  box-shadow: 0 0 0 3px rgba(185, 130, 88, 0.09);
}

.email-auth-input-shell.is-error,
.email-auth-otp.is-error {
  border-color: color-mix(in srgb, var(--auth-danger) 58%, transparent);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--auth-danger) 8%, transparent);
  animation: email-auth-shake 230ms ease-out;
}

.email-auth-input-shell input {
  min-width: 0;
  height: 100%;
  flex: 1;
  border: 0;
  outline: 0;
  color: var(--auth-ink);
  background: transparent;
  font-size: 13px;
}

.email-auth-input-shell input::placeholder { color: var(--auth-subtle); }

.email-auth-field-action {
  width: 30px;
  height: 30px;
  display: grid;
  flex: none;
  border: 0;
  color: var(--auth-subtle);
  background: transparent;
  cursor: pointer;
  place-items: center;
}

.email-auth-message-row {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.email-auth-message {
  min-height: 28px;
  margin: 0;
  padding: 6px 2px 0;
  color: var(--auth-subtle);
  font-size: 9px;
  line-height: 1.45;
}

.email-auth-message.is-error { color: var(--auth-danger); }

.email-auth-inline-link,
.email-auth-switch button,
.email-auth-text-button {
  border: 0;
  color: var(--auth-brand);
  background: transparent;
  cursor: pointer;
  font-weight: 650;
}

.email-auth-inline-link {
  flex: none;
  padding: 6px 2px 0;
  font-size: 9px;
}

.email-auth-primary {
  width: 100%;
  height: 47px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 9px;
  border: 0;
  border-radius: 10px;
  color: #fff;
  background: linear-gradient(135deg, var(--auth-brand), var(--auth-brand-deep));
  box-shadow: 0 9px 19px rgba(137, 89, 53, 0.16), inset 0 1px rgba(255, 255, 255, 0.18);
  cursor: pointer;
  font-size: 12px;
  font-weight: 700;
  transition: filter 150ms ease, box-shadow 150ms ease, transform 150ms ease;
}

:global(html.dark .email-auth-primary) {
  color: #fff;
  background: linear-gradient(135deg, var(--auth-brand), var(--auth-brand-deep));
  box-shadow: 0 9px 19px rgba(0, 0, 0, 0.28), inset 0 1px rgba(255, 255, 255, 0.18);
}

.email-auth-choice {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px;
}

.email-auth-choice.is-single {
  grid-template-columns: 1fr;
}

.email-auth-secondary {
  width: 100%;
  height: 47px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 9px;
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 10px;
  color: var(--auth-ink);
  background: rgba(255, 255, 255, 0.6);
  cursor: pointer;
  font-size: 12px;
  font-weight: 700;
  transition: border-color 150ms ease, background 150ms ease, transform 150ms ease;
}

.email-auth-primary:disabled,
.email-auth-secondary:disabled,
.email-auth-text-button:disabled,
.email-auth-switch button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
  transform: none;
}

.email-auth-spinner {
  width: 15px;
  height: 15px;
  border: 2px solid currentColor;
  border-right-color: transparent;
  border-radius: 50%;
  animation: email-auth-spin 750ms linear infinite;
}

.email-auth-trust {
  display: flex;
  justify-content: center;
  gap: 15px;
  margin-top: 18px;
  color: var(--auth-subtle);
  font-size: 8px;
}

.email-auth-trust span {
  display: inline-flex;
  align-items: center;
  gap: 5px;
}

.email-auth-switch {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  margin-top: 16px;
  color: var(--auth-muted);
  font-size: 10px;
}

.email-auth-switch button { padding: 3px; font-size: inherit; }

.email-auth-text-button {
  display: block;
  margin: 9px auto 0;
  padding: 4px;
  font-size: 9px;
}

.email-auth-method-list {
  display: grid;
  gap: 10px;
}

.email-auth-method-button,
:deep(.email-auth-method-list .btn) {
  width: 100%;
  min-height: 47px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 9px;
  margin: 0;
  padding: 0 14px;
  border: 1px solid var(--auth-line);
  border-radius: 8px;
  color: var(--auth-ink);
  background: var(--auth-surface);
  box-shadow: none;
  cursor: pointer;
  font-size: 11px;
  font-weight: 680;
  transition: border-color 150ms ease, background 150ms ease, transform 150ms ease;
}

.email-auth-method-button:disabled,
:deep(.email-auth-method-list .btn:disabled) {
  cursor: not-allowed;
  opacity: 0.55;
}

:deep(.email-auth-method-list > div),
:deep(.email-auth-method-list .grid) {
  gap: 10px;
  margin: 0;
}

.email-auth-method-error {
  min-height: 0;
  text-align: center;
}

.email-auth-otp {
  position: relative;
  display: grid;
  grid-template-columns: repeat(6, minmax(0, 1fr));
  gap: 7px;
  border-radius: 9px;
  cursor: text;
}

.email-auth-otp-input {
  position: absolute;
  z-index: 2;
  inset: 0;
  width: 100%;
  height: 100%;
  border: 0;
  border-radius: 9px;
  outline: 0;
  opacity: 0;
  cursor: text;
}

.email-auth-otp > span {
  height: 52px;
  display: grid;
  border: 1px solid rgba(61, 54, 47, 0.14);
  border-radius: 9px;
  color: #423e38;
  background: #fbfaf7;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 18px;
  font-weight: 650;
  letter-spacing: 0.02em;
  place-items: center;
  transition: border-color 150ms ease, background 150ms ease, transform 150ms cubic-bezier(0.22, 1, 0.36, 1);
}

.email-auth-otp:focus-within > span.is-current,
.email-auth-otp > span.is-filled + span:not(.is-filled) {
  border-color: rgba(181, 126, 84, 0.48);
  background: #fffefb;
  box-shadow: 0 0 0 2px rgba(185, 130, 88, 0.07);
}

.email-auth-otp > span.is-filled {
  border-color: rgba(143, 113, 88, 0.24);
  background: #f7f2ec;
  transform: translateY(-1px);
}

.email-auth-otp-caret {
  width: 1.5px;
  height: 18px;
  border-radius: 1px;
  background: #c4956a;
  animation: email-auth-caret 1s steps(1) infinite;
}

.email-auth-progress {
  min-height: 326px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
}

.email-auth-progress-rings {
  position: relative;
  width: 84px;
  height: 84px;
  display: grid;
  margin-bottom: 28px;
  color: var(--auth-brand);
  place-items: center;
}

.email-auth-progress-rings span {
  position: absolute;
  inset: 0;
  border: 1px solid color-mix(in srgb, var(--auth-brand) 18%, transparent);
  border-top-color: var(--auth-brand);
  border-radius: 50%;
  animation: email-auth-spin 1.7s linear infinite;
}

.email-auth-progress-rings span:nth-child(2) {
  inset: 11px;
  animation-direction: reverse;
  animation-duration: 1.2s;
}

.email-auth-skeleton {
  width: 220px;
  margin-top: 24px;
}

.email-auth-skeleton i {
  display: block;
  height: 7px;
  margin-top: 8px;
  border-radius: 4px;
  background: var(--auth-line);
  animation: email-auth-pulse 1.3s ease-in-out infinite;
}

.email-auth-skeleton i:nth-child(2) { width: 78%; }
.email-auth-skeleton i:nth-child(3) { width: 56%; }

.email-auth-success-mark {
  width: 62px;
  height: 62px;
  display: grid;
  margin-bottom: 25px;
  border: 1px solid color-mix(in srgb, #237a70 30%, transparent);
  border-radius: 50%;
  color: #237a70;
  background: color-mix(in srgb, #237a70 11%, var(--auth-bg));
  place-items: center;
  animation: email-auth-success-in 280ms cubic-bezier(0.2, 0.8, 0.2, 1) both;
}

:global(html.dark .email-auth-success-mark) { color: #79bdb0; }

.email-auth-legal {
  margin: 8px 0 0;
  color: var(--auth-subtle);
  font-size: 8px;
  line-height: 1.4;
  text-align: center;
}

@media (hover: hover) {
  .email-auth-icon-button:hover {
    border-color: var(--auth-line);
    color: var(--auth-ink);
    background: var(--auth-surface);
  }

  .email-auth-primary:not(:disabled):hover {
    filter: brightness(1.03);
    box-shadow: 0 11px 24px rgba(137, 89, 53, 0.22), inset 0 1px rgba(255, 255, 255, 0.18);
    transform: translateY(-1px);
  }

  .email-auth-secondary:not(:disabled):hover {
    border-color: color-mix(in srgb, var(--auth-brand) 40%, transparent);
    background: var(--auth-bg);
    transform: translateY(-1px);
  }

  :global(html.dark .email-auth-primary:not(:disabled):hover) {
    filter: brightness(1.05);
  }

  .email-auth-method-button:not(:disabled):hover,
  :deep(.email-auth-method-list .btn:not(:disabled):hover) {
    border-color: color-mix(in srgb, var(--auth-brand) 38%, transparent);
    background: var(--auth-bg);
    transform: translateY(-1px);
  }
}

@media (max-width: 520px) {
  .email-auth-layer {
    align-items: end;
    padding: 8px;
  }

  .email-auth-dialog {
    width: 100%;
    min-height: 500px;
    padding: 34px 24px 20px;
    border-radius: 20px 20px 14px 14px;
    animation-name: email-auth-sheet-in;
  }

  .email-auth-otp { gap: 5px; }
}

@keyframes email-auth-dialog-in {
  from { opacity: 0; transform: translateY(10px) scale(0.98); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}

@keyframes email-auth-sheet-in {
  from { opacity: 0; transform: translateY(36px); }
  to { opacity: 1; transform: translateY(0); }
}

@keyframes email-auth-backdrop-in {
  from { opacity: 0; }
  to { opacity: 1; }
}

@keyframes email-auth-stage-in {
  from { opacity: 0; transform: translateX(7px); }
  to { opacity: 1; transform: translateX(0); }
}

@keyframes email-auth-shake {
  0%, 100% { transform: translateX(0); }
  35% { transform: translateX(-3px); }
  70% { transform: translateX(2px); }
}

@keyframes email-auth-spin {
  to { transform: rotate(360deg); }
}

@keyframes email-auth-pulse {
  0%, 100% { opacity: 0.45; }
  50% { opacity: 0.9; }
}

@keyframes email-auth-success-in {
  from { opacity: 0; transform: scale(0.76); }
  to { opacity: 1; transform: scale(1); }
}

@keyframes email-auth-caret {
  0%, 45% { opacity: 1; }
  50%, 100% { opacity: 0; }
}

@media (prefers-reduced-motion: reduce) {
  .email-auth-dialog,
  .email-auth-backdrop,
  .email-auth-stage,
  .email-auth-input-shell.is-error,
  .email-auth-otp.is-error,
  .email-auth-progress-rings span,
  .email-auth-skeleton i,
  .email-auth-success-mark,
  .email-auth-spinner,
  .email-auth-otp-caret {
    animation-duration: 0.01ms !important;
    animation-iteration-count: 1 !important;
  }
}
</style>
