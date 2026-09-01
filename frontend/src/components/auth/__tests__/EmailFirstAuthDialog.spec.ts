import { flushPromises, mount } from '@vue/test-utils'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import EmailFirstAuthDialog from '@/components/auth/EmailFirstAuthDialog.vue'
import enCommon from '@/i18n/locales/en/common'
import zhCommon from '@/i18n/locales/zh/common'

const {
  appStore,
  authStore,
  pushMock,
  routeState,
  sendVerifyCodeMock,
  clearAffiliateMock,
} = vi.hoisted(() => ({
  appStore: {
    cachedPublicSettings: null as Record<string, unknown> | null,
    fetchPublicSettings: vi.fn(),
  },
  authStore: {
    login: vi.fn(),
    login2FA: vi.fn(),
    loginWithPasskey: vi.fn(),
    register: vi.fn(),
  },
  pushMock: vi.fn(),
  routeState: { path: '/', query: {} as Record<string, string> },
  sendVerifyCodeMock: vi.fn(),
  clearAffiliateMock: vi.fn(),
}))

vi.mock('vue-router', () => ({
  useRoute: () => routeState,
  useRouter: () => ({ push: pushMock }),
}))

vi.mock('vue-i18n', async (importOriginal) => {
  const actual = await importOriginal<typeof import('vue-i18n')>()
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key,
    }),
  }
})

vi.mock('@/stores', () => ({
  useAppStore: () => appStore,
  useAuthStore: () => authStore,
}))

vi.mock('@/api/auth', async () => {
  const actual = await vi.importActual<typeof import('@/api/auth')>('@/api/auth')
  return {
    ...actual,
    sendVerifyCode: (...args: unknown[]) => sendVerifyCodeMock(...args),
  }
})

vi.mock('@/utils/oauthAffiliate', () => ({
  clearAllAffiliateReferralCodes: () => clearAffiliateMock(),
  resolveAffiliateReferralCode: () => '',
}))

const simpleSettings = {
  registration_enabled: true,
  email_verify_enabled: true,
  registration_email_suffix_whitelist: [],
  registration_email_domain_quota_enabled: false,
  promo_code_enabled: false,
  password_reset_enabled: false,
  invitation_code_enabled: false,
  login_agreement_enabled: false,
  turnstile_enabled: false,
  tencent_captcha_enabled: false,
  aliyun_captcha_enabled: false,
  backend_mode_enabled: false,
  passkey_enabled: false,
  linuxdo_oauth_enabled: false,
  dingtalk_oauth_enabled: false,
  wechat_oauth_enabled: false,
  oidc_oauth_enabled: false,
  oidc_oauth_provider_name: 'OIDC',
  github_oauth_enabled: false,
  google_oauth_enabled: false,
}

function mountDialog(props: Record<string, unknown> = {}) {
  return mount(EmailFirstAuthDialog, {
    props: {
      open: true,
      siteName: 'ToCreate',
      siteLogo: '',
      dashboardPath: '/dashboard',
      delays: { preparing: 0, success: 0 },
      ...props,
    },
    global: {
      stubs: {
        Icon: { template: '<span data-testid="icon" />' },
      },
    },
  })
}

async function continueWithEmail(wrapper: ReturnType<typeof mountDialog>, value = ' User@Example.com ') {
  await wrapper.get('[data-testid="email-auth-email"]').setValue(value)
  await wrapper.get('form').trigger('submit')
  await flushPromises()
}

describe('EmailFirstAuthDialog', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    appStore.cachedPublicSettings = { ...simpleSettings }
    appStore.fetchPublicSettings.mockReset()
    authStore.login.mockReset()
    authStore.login2FA.mockReset()
    authStore.loginWithPasskey.mockReset()
    authStore.register.mockReset()
    pushMock.mockReset()
    sendVerifyCodeMock.mockReset()
    clearAffiliateMock.mockReset()
    routeState.path = '/'
    routeState.query = {}
    sessionStorage.clear()
    sendVerifyCodeMock.mockResolvedValue({ message: 'sent', countdown: 60 })
    authStore.login.mockResolvedValue({ access_token: 'token', user: {} })
    authStore.loginWithPasskey.mockResolvedValue({})
    authStore.register.mockResolvedValue({})
  })

  afterEach(() => {
    vi.useRealTimers()
    document.body.style.overflow = ''
  })

  it('uses the scoped-selector form that preserves html.dark rules during compilation', () => {
    const filename = resolve('src/components/auth/EmailFirstAuthDialog.vue')
    const source = readFileSync(filename, 'utf8')

    expect(source).toContain(':global(html.dark .email-auth-dialog)')
    expect(source).toContain(':global(html.dark .email-auth-primary)')
    expect(source).toContain(':global(html.dark .email-auth-success-mark)')
    expect(source).not.toMatch(/:global\(html\.dark\)\s+\.email-auth/)
  })

  it('localizes invalid credential responses from the real login API', () => {
    expect(enCommon.auth.errors.INVALID_CREDENTIALS).toBe('Invalid email or password.')
    expect(zhCommon.auth.errors.INVALID_CREDENTIALS).toBe('邮箱或密码不正确')
  })

  it('shows an inline error for an invalid email', async () => {
    const wrapper = mountDialog()

    await continueWithEmail(wrapper, 'invalid-email')

    expect(wrapper.text()).toContain('auth.invalidEmail')
    expect(wrapper.find('[data-testid="email-auth-password"]').exists()).toBe(false)
  })

  it('normalizes the email before showing the real password login step', async () => {
    const wrapper = mountDialog()

    await continueWithEmail(wrapper)

    expect(wrapper.get('[data-testid="email-auth-password"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('user@example.com')
  })

  it('hides the alternative-method entry when no alternative provider is enabled', async () => {
    const wrapper = mountDialog()

    await continueWithEmail(wrapper)

    expect(wrapper.find('[data-testid="email-auth-other-methods"]').exists()).toBe(false)
  })

  it('keeps enabled OAuth methods inside the new dialog', async () => {
    appStore.cachedPublicSettings = {
      ...simpleSettings,
      oidc_oauth_enabled: true,
      oidc_oauth_provider_name: 'Lihe ID',
    }
    const wrapper = mountDialog()

    await continueWithEmail(wrapper)
    await wrapper.get('[data-testid="email-auth-other-methods"]').trigger('click')

    expect(wrapper.get('[data-testid="email-auth-methods"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('auth.oidc.signIn')
    expect(pushMock).not.toHaveBeenCalled()

    await wrapper.get('[data-testid="email-auth-email-password"]').trigger('click')
    expect(wrapper.get('[data-testid="email-auth-password"]').exists()).toBe(true)
  })

  it('uses the real passkey login from the inline method step', async () => {
    Object.defineProperty(window, 'PublicKeyCredential', {
      configurable: true,
      value: class PublicKeyCredential {},
    })
    appStore.cachedPublicSettings = { ...simpleSettings, passkey_enabled: true }
    const wrapper = mountDialog()

    await continueWithEmail(wrapper)
    await wrapper.get('[data-testid="email-auth-other-methods"]').trigger('click')
    await wrapper.get('[data-testid="email-auth-passkey"]').trigger('click')
    await flushPromises()

    expect(authStore.loginWithPasskey).toHaveBeenCalledOnce()
    await vi.runAllTimersAsync()
    expect(pushMock).toHaveBeenCalledWith('/dashboard')
    Reflect.deleteProperty(window, 'PublicKeyCredential')
  })

  it('reuses the email-first flow as an embedded registration page', async () => {
    routeState.path = '/register'
    const wrapper = mountDialog({
      presentation: 'embedded',
      intent: 'register',
      initialEmail: ' Preset@Example.com ',
    })

    expect(wrapper.find('.email-auth-backdrop').exists()).toBe(false)
    expect(wrapper.find('.email-auth-close').exists()).toBe(false)
    expect(document.body.style.overflow).toBe('')
    expect(wrapper.get('[data-testid="email-auth-email"]').element).toHaveProperty(
      'value',
      'preset@example.com',
    )

    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(wrapper.get('[data-testid="email-auth-register-password"]').exists()).toBe(true)
    expect(wrapper.find('[data-testid="email-auth-password"]').exists()).toBe(false)
  })

  it('falls back to the full registration form for complex public settings', async () => {
    routeState.path = '/register'
    appStore.cachedPublicSettings = { ...simpleSettings, invitation_code_enabled: true }
    const wrapper = mountDialog({ presentation: 'embedded', intent: 'register' })

    await continueWithEmail(wrapper)
    await wrapper.get('[data-testid="email-auth-register-password"]').setValue('secret12')
    await wrapper.get('[data-testid="email-auth-confirm-password"]').setValue('secret12')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(sendVerifyCodeMock).not.toHaveBeenCalled()
    expect(pushMock).toHaveBeenCalledWith({
      path: '/register',
      query: { email: 'user@example.com', full: '1' },
    })
  })

  it('routes the homepage dialog directly to the full registration fallback', async () => {
    appStore.cachedPublicSettings = { ...simpleSettings, promo_code_enabled: true }
    const wrapper = mountDialog()

    await continueWithEmail(wrapper)
    await wrapper.get('[data-testid="email-auth-start-register"]').trigger('click')
    await wrapper.get('[data-testid="email-auth-register-password"]').setValue('secret12')
    await wrapper.get('[data-testid="email-auth-confirm-password"]').setValue('secret12')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(pushMock).toHaveBeenCalledWith({
      path: '/register',
      query: { email: 'user@example.com', full: '1' },
    })
  })

  it('uses the real registration email-code and account APIs', async () => {
    const wrapper = mountDialog()
    await continueWithEmail(wrapper)

    await wrapper.get('[data-testid="email-auth-start-register"]').trigger('click')
    await wrapper.get('[data-testid="email-auth-register-password"]').setValue('secret12')
    await wrapper.get('[data-testid="email-auth-confirm-password"]').setValue('secret12')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(sendVerifyCodeMock).toHaveBeenCalledWith({ email: 'user@example.com' })
    expect(wrapper.get('[data-testid="email-auth-code"]').exists()).toBe(true)

    await wrapper.get('[data-testid="email-auth-code"]').setValue('123456')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(authStore.register).toHaveBeenCalledWith({
      email: 'user@example.com',
      password: 'secret12',
      verify_code: '123456',
    })

    await vi.runAllTimersAsync()
    expect(pushMock).toHaveBeenCalledWith('/dashboard')
  })

  it('submits password login through the auth store', async () => {
    const wrapper = mountDialog()
    await continueWithEmail(wrapper)

    await wrapper.get('[data-testid="email-auth-password"]').setValue('secret12')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(authStore.login).toHaveBeenCalledWith({
      email: 'user@example.com',
      password: 'secret12',
    })

    await vi.runAllTimersAsync()
    expect(clearAffiliateMock).toHaveBeenCalledOnce()
    expect(pushMock).toHaveBeenCalledWith('/dashboard')
  })

  it('keeps a failed real login inside the dialog', async () => {
    authStore.login.mockRejectedValueOnce({
      status: 401,
      code: 'INVALID_CREDENTIALS',
      message: 'Invalid email or password',
    })
    const wrapper = mountDialog()
    await continueWithEmail(wrapper)

    await wrapper.get('[data-testid="email-auth-password"]').setValue('wrong-password')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(wrapper.get('[data-testid="email-auth-dialog"]').exists()).toBe(true)
    expect(wrapper.get('[data-testid="email-auth-password"]').exists()).toBe(true)
    expect(wrapper.text()).toContain('Invalid email or password')
    expect(pushMock).not.toHaveBeenCalled()
  })

  it('falls back to the full login page when captcha protection is enabled', async () => {
    appStore.cachedPublicSettings = { ...simpleSettings, turnstile_enabled: true }
    const wrapper = mountDialog()
    await continueWithEmail(wrapper)

    await wrapper.get('[data-testid="email-auth-password"]').setValue('secret12')
    await wrapper.get('form').trigger('submit')
    await flushPromises()

    expect(authStore.login).not.toHaveBeenCalled()
    expect(pushMock).toHaveBeenCalledWith({
      path: '/login',
      query: { email: 'user@example.com', full: '1' },
    })
  })

  it('uses the full login compatibility page for protected alternative methods', async () => {
    appStore.cachedPublicSettings = {
      ...simpleSettings,
      turnstile_enabled: true,
      oidc_oauth_enabled: true,
    }
    const wrapper = mountDialog()
    await continueWithEmail(wrapper)

    await wrapper.get('[data-testid="email-auth-other-methods"]').trigger('click')
    await flushPromises()

    expect(wrapper.find('[data-testid="email-auth-methods"]').exists()).toBe(false)
    expect(pushMock).toHaveBeenCalledWith({
      path: '/login',
      query: { email: 'user@example.com', full: '1' },
    })
  })
})
