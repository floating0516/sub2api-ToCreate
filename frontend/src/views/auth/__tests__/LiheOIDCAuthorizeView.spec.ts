import { beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import LiheOIDCAuthorizeView from '../LiheOIDCAuthorizeView.vue'

const route = {
  path: '/oidc/authorize',
  query: {} as Record<string, string>
}
const replace = vi.fn()
const resolve = vi.fn()
const prepareOIDCAuthorization = vi.fn()
const authorizeOIDC = vi.fn()
const logout = vi.fn()
const authState = { isAuthenticated: false, logout }

vi.mock('vue-router', () => ({
  useRoute: () => route,
  useRouter: () => ({ replace, resolve })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({ t: (key: string) => key })
  }
})

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => authState
}))

vi.mock('@/api/liheOIDC', () => ({
  prepareOIDCAuthorization: (...args: unknown[]) => prepareOIDCAuthorization(...args),
  authorizeOIDC: (...args: unknown[]) => authorizeOIDC(...args)
}))

vi.mock('@/utils/apiError', () => ({
  extractApiErrorMessage: () => 'authorization failed'
}))

function mountView() {
  return mount(LiheOIDCAuthorizeView, {
    global: {
      stubs: {
        AuthLayout: { template: '<div><slot /></div>' },
        Icon: true,
        RouterLink: { template: '<a><slot /></a>' }
      }
    }
  })
}

describe('LiheOIDCAuthorizeView', () => {
  beforeEach(() => {
    route.query = {}
    replace.mockReset()
    resolve.mockReset()
    prepareOIDCAuthorization.mockReset()
    authorizeOIDC.mockReset()
    logout.mockReset()
    authState.isAuthenticated = false
    resolve.mockReturnValue({ fullPath: '/oidc/authorize?request_id=prepared-request' })
    window.history.replaceState({}, '', '/oidc/authorize')
  })

  it('removes protocol parameters before waiting for request preparation', async () => {
    route.query = {
      response_type: 'code',
      client_id: 'lihe-chat-login',
      state: 'sensitive-state',
      nonce: 'sensitive-nonce'
    }
    window.history.replaceState(
      {},
      '',
      '/oidc/authorize?response_type=code&state=sensitive-state&nonce=sensitive-nonce'
    )
    let finishPreparation: ((value: { request_id: string; expires_in: number }) => void) | undefined
    prepareOIDCAuthorization.mockReturnValue(
      new Promise<{ request_id: string; expires_in: number }>((resolvePromise) => {
        finishPreparation = resolvePromise
      })
    )

    mountView()
    await Promise.resolve()

    expect(window.location.pathname).toBe('/oidc/authorize')
    expect(window.location.search).toBe('')
    expect(prepareOIDCAuthorization).toHaveBeenCalledWith(route.query)

    finishPreparation?.({ request_id: 'prepared-request', expires_in: 300 })
    await flushPromises()
    expect(replace).toHaveBeenCalledWith({
      path: '/oidc/authorize',
      query: { request_id: 'prepared-request' }
    })
  })

  it('restores an opaque request handle through API login without re-preparing', async () => {
    route.query = { request_id: 'prepared-request' }

    mountView()
    await flushPromises()

    expect(prepareOIDCAuthorization).not.toHaveBeenCalled()
    expect(authorizeOIDC).not.toHaveBeenCalled()
    expect(resolve).toHaveBeenCalledWith({
      path: '/oidc/authorize',
      query: { request_id: 'prepared-request' }
    })
    expect(replace).toHaveBeenCalledWith({
      path: '/login',
      query: { redirect: '/oidc/authorize?request_id=prepared-request' }
    })
  })

  it('forces a fresh API login when the provider requests reauthentication', async () => {
    route.query = { request_id: 'prepared-request' }
    authState.isAuthenticated = true
    authorizeOIDC.mockResolvedValue({ reauthenticate: true })

    mountView()
    await flushPromises()

    expect(authorizeOIDC).toHaveBeenCalledWith('prepared-request')
    expect(logout).toHaveBeenCalledOnce()
    expect(replace).toHaveBeenCalledWith({
      path: '/login',
      query: { redirect: '/oidc/authorize?request_id=prepared-request' }
    })
  })
})
